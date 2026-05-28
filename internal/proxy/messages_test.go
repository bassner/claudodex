package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bassner/claudodex/internal/auth"
)

func TestMessagesStreamsCodexResponseAndBuildsUpstreamRequest(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	var captured map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/codex/responses" {
			t.Fatalf("unexpected upstream path %s", r.URL.Path)
		}
		if got := r.Header.Get("authorization"); got != "Bearer access-1" {
			t.Fatalf("authorization = %q", got)
		}
		if got := r.Header.Get("chatgpt-account-id"); got != "acc_123" {
			t.Fatalf("chatgpt-account-id = %q", got)
		}
		if got := r.Header.Get("x-anthropic-billing-header"); got != "" {
			t.Fatalf("forwarded billing header = %q", got)
		}
		if got := r.Header.Get("session-id"); got != "session-123" {
			t.Fatalf("session-id = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("content-type", "text/event-stream")
		w.Header().Set("x-codex-primary-used-percent", "42")
		w.Header().Set("x-codex-primary-window-minutes", "300")
		w.Header().Set("x-codex-primary-reset-at", "1770000000")
		w.Header().Set("x-codex-secondary-used-percent", "17")
		w.Header().Set("x-codex-secondary-window-minutes", "10080")
		w.Header().Set("x-codex-secondary-reset-at", "1770500000")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: response.created`,
			`data: {"type":"response.created","response":{"id":"resp_1"}}`,
			``,
			`event: response.output_item.added`,
			`data: {"type":"response.output_item.added","item":{"type":"message","id":"item_1"}}`,
			``,
			`event: response.output_text.delta`,
			`data: {"type":"response.output_text.delta","delta":"ok"}`,
			``,
			`event: response.output_item.done`,
			`data: {"type":"response.output_item.done","item":{"type":"message","id":"item_1","content":[{"type":"output_text","text":"ok"}]}}`,
			``,
			`event: response.completed`,
			`data: {"type":"response.completed","response":{"usage":{"input_tokens":4,"output_tokens":1}}}`,
			``,
			``,
		}, "\n")))
	}))
	defer upstream.Close()

	server := New(Config{Version: "test", Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	body := `{"model":"claude-sonnet-4-6","system":"hello\nx-anthropic-billing-header: secret\nworld","output_config":{"effort":"max"},"stream":true,"messages":[{"role":"user","content":"say ok"}]}`
	req, err := http.NewRequest(http.MethodPost, "http://"+addr+"/v1/messages?beta=true", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-claude-code-session-id", "session-123")
	req.Header.Set("x-anthropic-billing-header", "must-not-forward")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if got := resp.Header.Get("anthropic-ratelimit-unified-5h-utilization"); got != "0.42" {
		t.Fatalf("5h utilization header = %q", got)
	}
	if got := resp.Header.Get("anthropic-ratelimit-unified-7d-utilization"); got != "0.17" {
		t.Fatalf("7d utilization header = %q", got)
	}
	sse := readAllString(t, resp)
	if !strings.Contains(sse, "event: message_start") || !strings.Contains(sse, `"model":"claude-sonnet-4-6"`) {
		t.Fatalf("missing message_start/model in SSE:\n%s", sse)
	}
	if !strings.Contains(sse, `"text":"ok"`) || !strings.Contains(sse, "event: message_stop") {
		t.Fatalf("missing text/stop in SSE:\n%s", sse)
	}
	if captured["model"] != "gpt-5.4" {
		t.Fatalf("upstream model = %#v", captured["model"])
	}
	instructions, _ := captured["instructions"].(string)
	if !strings.HasPrefix(instructions, "hello\nworld\n\nClaude Code compatibility:\n") {
		t.Fatalf("instructions = %#v", captured["instructions"])
	}
	if strings.Contains(instructions, "x-anthropic-billing-header") || strings.Contains(instructions, "must-not-forward") {
		t.Fatalf("billing header leaked into instructions: %q", instructions)
	}
	reasoning := captured["reasoning"].(map[string]any)
	if reasoning["effort"] != "xhigh" {
		t.Fatalf("reasoning = %#v", reasoning)
	}
	if captured["stream"] != true || captured["store"] != false || captured["parallel_tool_calls"] != false {
		t.Fatalf("upstream stream/store/parallel = %#v", captured)
	}
}

func TestMessagesNonStreamingAssemblesMessage(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte("event: response.output_item.done\n"))
		_, _ = w.Write([]byte(`data: {"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"done"}]}}` + "\n\n"))
		_, _ = w.Write([]byte("event: response.completed\n"))
		_, _ = w.Write([]byte(`data: {"type":"response.completed","response":{"stop_reason":"stop","usage":{"input_tokens":1,"output_tokens":1}}}` + "\n\n"))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","stream":false,"messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Content) != 1 || body.Content[0].Text != "done" || body.StopReason != "end_turn" {
		t.Fatalf("body = %#v", body)
	}
}

func TestMessagesStreamingMalformedUpstreamSSEEmitsAnthropicErrorEvent(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte("event: response.output_text.delta\n"))
		_, _ = w.Write([]byte("data: {not-json}\n\n"))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","stream":true,"messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	sse := readAllString(t, resp)
	if !strings.Contains(sse, "event: error") || !strings.Contains(sse, "malformed SSE") {
		t.Fatalf("missing Anthropic error event in SSE:\n%s", sse)
	}
}

func TestMessagesStreamingForwardsNamedUpstreamErrorEvent(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte("event: error\n"))
		_, _ = w.Write([]byte(`data: {"error":{"type":"api_error","message":"upstream boom"}}` + "\n\n"))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","stream":true,"messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	sse := readAllString(t, resp)
	if !strings.Contains(sse, "event: error") || !strings.Contains(sse, "upstream boom") {
		t.Fatalf("missing upstream error event in SSE:\n%s", sse)
	}
}

func TestMessagesStreamingCleanCloseWithoutTerminalEventEmitsError(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte("event: response.created\n"))
		_, _ = w.Write([]byte(`data: {"type":"response.created","response":{"id":"resp_1"}}` + "\n\n"))
		_, _ = w.Write([]byte("event: response.output_text.delta\n"))
		_, _ = w.Write([]byte(`data: {"type":"response.output_text.delta","delta":"partial"}` + "\n\n"))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","stream":true,"messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	sse := readAllString(t, resp)
	if !strings.Contains(sse, "event: error") || !strings.Contains(sse, "ended before completion") {
		t.Fatalf("missing premature-close error event in SSE:\n%s", sse)
	}
}

func TestMessagesNonStreamingMalformedUpstreamSSEReturns502(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte("event: response.output_text.delta\n"))
		_, _ = w.Write([]byte("data: {not-json}\n\n"))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","stream":false,"messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body struct {
		Type  string `json:"type"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Type != "error" || body.Error.Type != "api_error" || !strings.Contains(body.Error.Message, "malformed SSE") {
		t.Fatalf("body = %#v", body)
	}
}

func TestMessagesNonStreamingCleanCloseWithoutTerminalEventReturns502(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte("event: response.output_text.delta\n"))
		_, _ = w.Write([]byte(`data: {"type":"response.output_text.delta","delta":"partial"}` + "\n\n"))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","stream":false,"messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body.Error.Message, "ended before completion") {
		t.Fatalf("body = %#v", body)
	}
}

func TestMessagesRefreshesAndRetriesOnceOn401(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "expired-access")
	var attempts atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/codex/responses":
			attempt := attempts.Add(1)
			if attempt == 1 {
				http.Error(w, `{"error":{"message":"expired"}}`, http.StatusUnauthorized)
				return
			}
			if got := r.Header.Get("authorization"); got != "Bearer fresh-access" {
				t.Fatalf("retry authorization = %q", got)
			}
			w.Header().Set("content-type", "text/event-stream")
			_, _ = w.Write([]byte("event: response.completed\n"))
			_, _ = w.Write([]byte(`data: {"type":"response.completed","response":{"usage":{"input_tokens":0,"output_tokens":0}}}` + "\n\n"))
		case "/oauth/token":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"fresh-access","refresh_token":"refresh-2"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	server := New(Config{Home: home, CodexBaseURL: upstream.URL, TokenEndpoint: upstream.URL + "/oauth/token", HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.StatusCode, readAllString(t, resp))
	}
	if attempts.Load() != 2 {
		t.Fatalf("attempts = %d, want 2", attempts.Load())
	}
}

func TestMessagesMapsNonJSONUpstreamError(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html>nope</html>"))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Type != "permission_error" || body.Error.Message != "Codex upstream returned HTTP 403" {
		t.Fatalf("body = %#v", body)
	}
}

func TestMessagesMaps429RateLimitHeaders(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-codex-primary-used-percent", "100")
		w.Header().Set("x-codex-primary-window-minutes", "300")
		w.Header().Set("x-codex-primary-reset-at", "1770000000")
		w.Header().Set("retry-after", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"quota exhausted"}}`))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if got := resp.Header.Get("anthropic-ratelimit-unified-status"); got != "rejected" {
		t.Fatalf("status header = %q", got)
	}
	if got := resp.Header.Get("anthropic-ratelimit-unified-representative-claim"); got != "five_hour" {
		t.Fatalf("claim header = %q", got)
	}
	var body struct {
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Type != "rate_limit_error" {
		t.Fatalf("body = %#v", body)
	}
}

func TestUsageFetchesWhamUsageAndMapsResponse(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wham/usage" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("authorization"); got != "Bearer access-1" {
			t.Fatalf("authorization = %q", got)
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"account_id":"hidden",
			"email":"hidden@example.com",
			"rate_limit":{
				"primary_window":{"used_percent":12,"limit_window_seconds":18000,"reset_at":1770000000},
				"secondary_window":{"used_percent":34,"limit_window_seconds":604800,"reset_at":1770500000}
			},
			"credits":{"has_credits":true,"unlimited":false},
			"spend_control":{"individual_limit":null}
		}`))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Get("http://" + addr + "/api/oauth/usage")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["account_id"]; ok {
		t.Fatalf("account_id leaked: %#v", body)
	}
	five := body["five_hour"].(map[string]any)
	if five["utilization"] != float64(12) {
		t.Fatalf("five_hour = %#v", five)
	}
	extra := body["extra_usage"].(map[string]any)
	if extra["is_enabled"] != true {
		t.Fatalf("extra_usage = %#v", extra)
	}
}

func TestUsageRefreshesAndRetriesOnceOn401(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "expired-access")
	var attempts atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wham/usage":
			attempt := attempts.Add(1)
			if attempt == 1 {
				http.Error(w, `{"error":{"message":"expired"}}`, http.StatusUnauthorized)
				return
			}
			if got := r.Header.Get("authorization"); got != "Bearer fresh-access" {
				t.Fatalf("retry authorization = %q", got)
			}
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{
				"rate_limit":{
					"primary_window":{"used_percent":1,"limit_window_seconds":18000,"reset_at":1770000000},
					"secondary_window":{"used_percent":2,"limit_window_seconds":604800,"reset_at":1770500000}
				}
			}`))
		case "/oauth/token":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"fresh-access","refresh_token":"refresh-2"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, TokenEndpoint: upstream.URL + "/oauth/token", HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Get("http://" + addr + "/api/oauth/usage")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.StatusCode, readAllString(t, resp))
	}
	if attempts.Load() != 2 {
		t.Fatalf("attempts = %d, want 2", attempts.Load())
	}
}

func saveTestAuth(t *testing.T, home, accessToken string) {
	t.Helper()
	err := auth.NewStore(home).Save(auth.File{
		AuthMode: "chatgpt",
		Issuer:   auth.Issuer,
		ClientID: auth.ClientID,
		Tokens: auth.Tokens{
			AccessToken:  accessToken,
			RefreshToken: "refresh-1",
			AccountID:    "acc_123",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func readAllString(t *testing.T, resp *http.Response) string {
	t.Helper()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
