package codex

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestClientCreateResponseBuildsCodexHeaders(t *testing.T) {
	client := Client{Version: "1.2.3"}
	headers := client.headers(Credentials{
		AccessToken:    "access-1",
		AccountID:      "acc_123",
		InstallationID: "install-1",
		FedRAMP:        true,
	}, Route{SessionID: "session-1", ThreadID: "thread-1"}, true)

	want := map[string]string{
		"authorization":                     "Bearer access-1",
		"content-type":                      "application/json",
		"accept":                            "text/event-stream",
		"originator":                        "codex_cli_rs",
		"chatgpt-account-id":                "acc_123",
		"x-codex-installation-id":           "install-1",
		"x-codex-turn-state":                "null",
		"x-client-request-id":               "session-1",
		"session-id":                        "session-1",
		"thread-id":                         "thread-1",
		"x-openai-internal-codex-residency": "us",
		"x-openai-fedramp":                  "true",
		"openai-beta":                       "responses_websockets=2026-02-06",
	}
	for key, value := range want {
		if headers[key] != value {
			t.Fatalf("%s = %q, want %q in %#v", key, headers[key], value, headers)
		}
	}
	if !strings.Contains(headers["user-agent"], "claudodex/1.2.3") {
		t.Fatalf("user-agent = %q", headers["user-agent"])
	}
}

func TestContentPartMarshalIncludesRequiredEmptyText(t *testing.T) {
	data, err := json.Marshal(Request{
		Model: "gpt-5.5",
		Input: []InputItem{{
			Type: "message",
			Role: "user",
			Content: []ContentPart{
				{Type: "input_text", Text: ""},
				{Type: "input_image", ImageURL: "data:image/png;base64,AAA", Detail: "high"},
				{Type: "output_text", Text: ""},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	for _, want := range []string{
		`{"type":"input_text","text":""}`,
		`{"type":"input_image","image_url":"data:image/png;base64,AAA","detail":"high"}`,
		`{"type":"output_text","text":""}`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("request JSON missing %s:\n%s", want, body)
		}
	}
	if strings.Contains(body, `"input_image","text"`) {
		t.Fatalf("image part should not include text field:\n%s", body)
	}
}

func TestClientCreateResponseRetriesWithoutHTTPBetaHeaderOn400(t *testing.T) {
	t.Setenv("CLAUDODEX_HTTP_BETA_HEADER", "1")
	var attempts atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := attempts.Add(1)
		if r.URL.Path != "/codex/responses" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if attempt == 1 {
			if got := r.Header.Get("openai-beta"); got != "responses_websockets=2026-02-06" {
				t.Fatalf("first openai-beta = %q", got)
			}
			http.Error(w, `{"error":{"message":"unknown beta header"}}`, http.StatusBadRequest)
			return
		}
		if got := r.Header.Get("openai-beta"); got != "" {
			t.Fatalf("retry openai-beta = %q", got)
		}
		w.Header().Set("content-type", "text/event-stream")
		_, _ = io.WriteString(w, "event: response.completed\n")
		_, _ = io.WriteString(w, `data: {"type":"response.completed","response":{}}`+"\n\n")
	}))
	defer upstream.Close()

	client := Client{BaseURL: upstream.URL, HTTPClient: upstream.Client()}
	resp, err := client.CreateResponse(context.Background(), Request{
		Model:  "gpt-5.5",
		Input:  []InputItem{{Type: "message", Role: "user", Content: []ContentPart{{Type: "input_text", Text: "hi"}}}},
		Stream: true,
	}, Credentials{AccessToken: "access-1"}, Route{SessionID: "session-1"})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if attempts.Load() != 2 {
		t.Fatalf("attempts = %d, want 2", attempts.Load())
	}
}

func TestClientFetchModels(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/codex/models" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("client_version"); got != "1.2.3" {
			t.Fatalf("client_version = %q", got)
		}
		if got := r.Header.Get("authorization"); got != "Bearer access-1" {
			t.Fatalf("authorization = %q", got)
		}
		if got := r.Header.Get("accept"); got != "application/json" {
			t.Fatalf("accept = %q", got)
		}
		_, _ = io.WriteString(w, `{"models":[{"slug":"gpt-5.5","display_name":"GPT-5.5","context_window":272000,"max_context_window":272000,"supported_in_api":true,"visibility":"list"}]}`)
	}))
	defer upstream.Close()

	client := Client{BaseURL: upstream.URL, HTTPClient: upstream.Client(), Version: "1.2.3"}
	models, err := client.FetchModels(context.Background(), Credentials{AccessToken: "access-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].Slug != "gpt-5.5" || models[0].ContextWindow != 272000 {
		t.Fatalf("models = %#v", models)
	}
}

func TestClientFetchModelsUsesProtocolVersionForInvalidDevVersion(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("client_version"); got != DefaultModelsClientVersion {
			t.Fatalf("client_version = %q", got)
		}
		_, _ = io.WriteString(w, `{"models":[]}`)
	}))
	defer upstream.Close()

	client := Client{BaseURL: upstream.URL, HTTPClient: upstream.Client(), Version: "dev"}
	if _, err := client.FetchModels(context.Background(), Credentials{AccessToken: "access-1"}); err != nil {
		t.Fatal(err)
	}
}

func TestClientFetchModelsUsesProtocolVersionForOlderProductVersion(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("client_version"); got != DefaultModelsClientVersion {
			t.Fatalf("client_version = %q", got)
		}
		_, _ = io.WriteString(w, `{"models":[]}`)
	}))
	defer upstream.Close()

	client := Client{BaseURL: upstream.URL, HTTPClient: upstream.Client(), Version: "0.1.0"}
	if _, err := client.FetchModels(context.Background(), Credentials{AccessToken: "access-1"}); err != nil {
		t.Fatal(err)
	}
}
