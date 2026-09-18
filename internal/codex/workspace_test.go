package codex

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestClientCreateResponseUsesSelectedWorkspaceRoutingAcrossAccountAndTokenChanges(t *testing.T) {
	type observedRequest struct {
		path    string
		routing string
		account string
		token   string
	}
	var mu sync.Mutex
	var observed []observedRequest
	backend := func() *httptest.Server {
		return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			observed = append(observed, observedRequest{
				path:    r.URL.Path,
				routing: r.Header.Get(accountRoutingHeader),
				account: r.Header.Get("chatgpt-account-id"),
				token:   r.Header.Get("authorization"),
			})
			mu.Unlock()
			w.Header().Set("content-type", "text/event-stream")
			_, _ = io.WriteString(w, "event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{}}\n\n")
		}))
	}
	backendA := backend()
	defer backendA.Close()
	backendB := backend()
	defer backendB.Close()

	var discovery *httptest.Server
	discovery = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/backend-api/wham/accounts/check" {
			t.Fatalf("accounts check path = %q", r.URL.Path)
		}
		token := strings.TrimPrefix(r.Header.Get("authorization"), "Bearer ")
		account := r.Header.Get("chatgpt-account-id")
		origin, override := backendA.URL, "us"
		switch {
		case account == "account-2" && token == "token-2":
			origin, override = backendB.URL, "us_cr"
		case account == "account-2" && token == "token-3":
			origin, override = backendA.URL, "NO_CONSTRAINT"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"accounts": []map[string]any{{
			"id":                       account,
			"workspace_backend_origin": origin,
			"account_routing_override": override,
		}}})
	}))
	defer discovery.Close()

	httpClient := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}} // test certificates
	client := Client{BaseURL: discovery.URL + "/backend-api", ChatGPTBaseURL: discovery.URL, HTTPClient: httpClient}
	for _, credentials := range []Credentials{
		{AccessToken: "token-1", AccountID: "account-1"},
		{AccessToken: "token-2", AccountID: "account-2"},
		{AccessToken: "token-3", AccountID: "account-2"},
	} {
		resp, err := client.CreateResponse(context.Background(), Request{Model: "gpt-5.6-sol", Stream: true}, credentials, Route{})
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
	}

	want := []observedRequest{
		{path: "/backend-api/codex/responses", routing: "us", account: "account-1", token: "Bearer token-1"},
		{path: "/backend-api/codex/responses", routing: "us_cr", account: "account-2", token: "Bearer token-2"},
		{path: "/backend-api/codex/responses", routing: "", account: "account-2", token: "Bearer token-3"},
	}
	mu.Lock()
	defer mu.Unlock()
	if len(observed) != len(want) {
		t.Fatalf("observed requests = %#v", observed)
	}
	for i := range want {
		if observed[i] != want[i] {
			t.Fatalf("request %d = %#v, want %#v", i, observed[i], want[i])
		}
	}
}

func TestClientCreateResponseRejectsWorkspaceRedirect(t *testing.T) {
	var redirected atomic.Bool
	redirectTarget := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		redirected.Store(true)
	}))
	defer redirectTarget.Close()
	backend := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectTarget.URL, http.StatusTemporaryRedirect)
	}))
	defer backend.Close()
	var discovery *httptest.Server
	discovery = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"accounts": []map[string]any{{
			"id":                       "account-1",
			"workspace_backend_origin": backend.URL,
			"account_routing_override": "NO_CONSTRAINT",
		}}})
	}))
	defer discovery.Close()
	client := Client{
		BaseURL:        discovery.URL,
		ChatGPTBaseURL: discovery.URL,
		HTTPClient: &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // test certificates
		}},
	}
	_, err := client.CreateResponse(context.Background(), Request{Model: "gpt-5.6-sol"}, Credentials{AccessToken: "token", AccountID: "account-1"}, Route{})
	var upstream *UpstreamError
	if !errors.As(err, &upstream) || upstream.Status != http.StatusTemporaryRedirect {
		t.Fatalf("error = %v, want HTTP 307 UpstreamError", err)
	}
	if redirected.Load() {
		t.Fatal("routed request followed redirect")
	}
}

func TestApplyWorkspaceRoutingValidation(t *testing.T) {
	baseURL := "https://chatgpt.com/backend-api"
	gotBaseURL, gotHeader, err := applyWorkspaceRouting(baseURL, "NO_CONSTRAINT", "NO_CONSTRAINT")
	if err != nil || gotBaseURL != baseURL || gotHeader != "" {
		t.Fatalf("NO_CONSTRAINT routing = (%q, %q, %v), want unchanged base URL and no header", gotBaseURL, gotHeader, err)
	}

	for _, test := range []struct {
		name     string
		origin   string
		override string
		wantErr  string
	}{
		{name: "http", origin: "http://workspace.example", override: "us", wantErr: "backend origin"},
		{name: "path", origin: "https://workspace.example/path", override: "us", wantErr: "backend origin"},
		{name: "userinfo", origin: "https://user@workspace.example", override: "us", wantErr: "backend origin"},
		{name: "unknown routing", origin: "https://workspace.example", override: "eu", wantErr: "routing override"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := applyWorkspaceRouting("https://chatgpt.com/backend-api", test.origin, test.override)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestAccountsCheckURLSupportsChatGPTPathStyles(t *testing.T) {
	for _, test := range []struct {
		base string
		want string
	}{
		{base: "https://chatgpt.com/backend-api", want: "https://chatgpt.com/backend-api/wham/accounts/check"},
		{base: "https://chatgpt.com/api/codex", want: "https://chatgpt.com/api/codex/accounts/check"},
		{base: "https://chatgpt.com", want: "https://chatgpt.com/api/codex/accounts/check"},
	} {
		got, err := accountsCheckURL(test.base)
		if err != nil || got != test.want {
			t.Fatalf("accountsCheckURL(%q) = (%q, %v), want %q", test.base, got, err, test.want)
		}
	}
}
