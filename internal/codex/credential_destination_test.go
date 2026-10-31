package codex

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestBorrowedCredentialDestinationAllowsOnlyOfficialOrigin(t *testing.T) {
	credentials := Credentials{AllowedOrigin: "https://chatgpt.com", AccessToken: "fixture"}
	for _, destination := range []string{"https://chatgpt.com/backend-api", "https://chatgpt.com:443/backend-api", "wss://chatgpt.com/backend-api/codex/responses"} {
		if err := CheckCredentialDestination(credentials, destination); err != nil {
			t.Fatalf("official destination rejected: %v", err)
		}
	}
	for _, destination := range []string{"https://example.invalid/backend-api", "http://chatgpt.com/backend-api", "https://chatgpt.com.evil.invalid", "https://chatgpt.com:444/backend-api", "https://fixture@chatgpt.com/backend-api", "ws://chatgpt.com/backend-api"} {
		if err := CheckCredentialDestination(credentials, destination); err == nil {
			t.Fatalf("unsafe destination accepted: %s", destination)
		}
	}
	if err := CheckCredentialDestination(Credentials{AccessToken: "synthetic-direct"}, "http://127.0.0.1:8080"); err != nil {
		t.Fatal(err)
	}
}

func TestBorrowedCredentialHTTPClientRejectsRedirectWithoutSending(t *testing.T) {
	calls := 0
	base := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.Host != "chatgpt.com" {
			t.Fatal("borrowed token sent to external host")
		}
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"https://example.invalid/capture"}}, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
	})}
	client := CredentialHTTPClient(base, Credentials{AllowedOrigin: "https://chatgpt.com"})
	if _, err := client.Get("https://chatgpt.com/backend-api/codex/models"); err == nil {
		t.Fatal("external redirect accepted")
	}
	if calls != 1 {
		t.Fatalf("transport calls=%d", calls)
	}
	if _, err := client.Get("https://example.invalid/capture"); err == nil {
		t.Fatal("initial custom destination accepted")
	}
	if calls != 1 {
		t.Fatal("custom destination reached transport")
	}
}

func TestBorrowedCredentialHTTPClientPreservesCallerPolicy(t *testing.T) {
	policyError := errors.New("caller redirect policy")
	base := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return policyError }, Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"https://chatgpt.com/next"}}, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
	})}
	client := CredentialHTTPClient(base, Credentials{AllowedOrigin: "https://chatgpt.com"})
	if _, err := client.Get("https://chatgpt.com/start"); !errors.Is(err, policyError) {
		t.Fatalf("policy err=%v", err)
	}
	if base.Transport == client.Transport {
		t.Fatal("base client mutated")
	}
}

func TestBorrowedCredentialClientRejectsCustomProviderBeforeNetwork(t *testing.T) {
	called := false
	client := Client{BaseURL: "https://example.invalid", HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { called = true; return nil, errors.New("should not run") })}}
	credentials := Credentials{AllowedOrigin: "https://chatgpt.com", AccessToken: "fixture", AccountID: "account"}
	if _, err := client.CreateResponse(context.Background(), Request{Model: "fixture"}, credentials, Route{}); err == nil {
		t.Fatal("custom response destination accepted")
	}
	if _, err := client.FetchUsage(context.Background(), credentials); err == nil {
		t.Fatal("custom usage destination accepted")
	}
	var conversation WebSocketConversation
	if _, err := conversation.CreateResponse(context.Background(), client, Request{Model: "fixture"}, credentials, Route{}, false); err == nil {
		t.Fatal("custom websocket destination accepted")
	}
	if called {
		t.Fatal("custom provider received credentials")
	}
}

func TestBorrowedCredentialWorkspaceRejectsCustomBootstrapAndResolvedHost(t *testing.T) {
	credentials := Credentials{AllowedOrigin: "https://chatgpt.com", AccessToken: "fixture", AccountID: "account"}
	calls := 0
	client := Client{ChatGPTBaseURL: "https://example.invalid", HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"accounts":[{"id":"account","workspace_backend_origin":"https://example.invalid","account_routing_override":"NO_CONSTRAINT"}]}`)), Request: req}, nil
	})}}
	if _, err := client.resolveResponsesDestination(context.Background(), credentials); err == nil {
		t.Fatal("custom bootstrap accepted")
	}
	if calls != 0 {
		t.Fatal("unsafe bootstrap reached network")
	}
	client.ChatGPTBaseURL = ""
	if _, err := client.resolveResponsesDestination(context.Background(), credentials); err == nil {
		t.Fatal("custom resolved workspace accepted")
	}
	if calls != 1 {
		t.Fatalf("workspace network calls=%d", calls)
	}
}

func TestBorrowedCredentialWebsocketRejectsBeforeDial(t *testing.T) {
	called := false
	client := Client{BaseURL: "wss://example.invalid/backend-api", WebSocketDialer: &websocket.Dialer{NetDialContext: func(context.Context, string, string) (net.Conn, error) {
		called = true
		return nil, errors.New("should not dial")
	}}}
	var conversation WebSocketConversation
	if _, err := conversation.CreateResponse(context.Background(), client, Request{}, Credentials{AllowedOrigin: "https://chatgpt.com"}, Route{}, false); err == nil {
		t.Fatal("custom websocket origin accepted")
	}
	if called {
		t.Fatal("unsafe websocket destination dialed")
	}
}
