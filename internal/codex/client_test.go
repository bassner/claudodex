package codex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientCreateResponseBuildsCodexHeaders(t *testing.T) {
	client := Client{Version: "1.2.3"}
	headers := client.headers(Credentials{
		AccessToken:    "access-1",
		AccountID:      "acc_123",
		InstallationID: "install-1",
		FedRAMP:        true,
	}, Route{SessionID: "session-1", ThreadID: "thread-1", TurnID: "turn-1", RootTurnID: "root-1", ParentThreadID: "parent-1", Subagent: "collab_spawn"}, &Request{Model: "gpt-5.6-terra", ServiceTier: "priority", ClientMetadata: map[string]string{"x-codex-turn-metadata": `{"root_turn_id":"root-1"}`}}, true)

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
		"x-codex-parent-thread-id":          "parent-1",
		"x-openai-subagent":                 "collab_spawn",
		"x-codex-turn-metadata":             `{"root_turn_id":"root-1"}`,
		"x-openai-internal-codex-residency": "us",
		"x-openai-fedramp":                  "true",
		"openai-beta":                       "responses_websockets=2026-02-06",
		"x-codex-routing-hint":              "model=gpt-5.6-terra;tier=priority",
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

func TestApplyResponsesMetadataIncludesCanonicalRootTurnIdentity(t *testing.T) {
	request := Request{}
	route := Route{
		SessionID:      "session-1",
		ThreadID:       "thread-1",
		TurnID:         "turn-child",
		RootTurnID:     "turn-root",
		ParentThreadID: "thread-parent",
		Subagent:       "collab_spawn",
	}
	ApplyResponsesMetadata(&request, "install-1", route)
	for key, want := range map[string]string{
		"x-codex-installation-id":  "install-1",
		"session_id":               "session-1",
		"thread_id":                "thread-1",
		"turn_id":                  "turn-child",
		"root_turn_id":             "turn-root",
		"x-codex-parent-thread-id": "thread-parent",
		"x-openai-subagent":        "collab_spawn",
	} {
		if got := request.ClientMetadata[key]; got != want {
			t.Fatalf("client_metadata[%q] = %q, want %q", key, got, want)
		}
	}
	var canonical map[string]any
	if err := json.Unmarshal([]byte(request.ClientMetadata["x-codex-turn-metadata"]), &canonical); err != nil {
		t.Fatal(err)
	}
	if canonical["root_turn_id"] != "turn-root" || canonical["turn_id"] != "turn-child" || canonical["request_kind"] != "turn" {
		t.Fatalf("canonical turn metadata = %#v", canonical)
	}
}

func TestCodexRoutingHintRequiresChatGPTResponsesRequest(t *testing.T) {
	client := Client{}
	credentials := Credentials{AccountID: "account-1"}
	request := &Request{Model: "gpt-5.6-sol"}
	if got := client.headers(credentials, Route{}, request, false)["x-codex-routing-hint"]; got != "model=gpt-5.6-sol" {
		t.Fatalf("model-only routing hint = %q", got)
	}
	if got := client.headers(credentials, Route{}, nil, false)["x-codex-routing-hint"]; got != "" {
		t.Fatalf("non-Responses routing hint = %q, want omitted", got)
	}
	if got := client.headers(Credentials{}, Route{}, request, false)["x-codex-routing-hint"]; got != "" {
		t.Fatalf("provider-owned credential routing hint = %q, want omitted", got)
	}
}

func TestClientCreateResponseRetriesConnectFailuresOutsideAttemptBudget(t *testing.T) {
	var transportAttempts atomic.Int32
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempt := transportAttempts.Add(1)
		if attempt <= 3 {
			return nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection unavailable")}
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("ok")),
			Request:    req,
		}, nil
	})
	client := Client{
		HTTPClient:             &http.Client{Transport: transport},
		ResponseHeaderAttempts: 1,
		ConnectRetryInitial:    time.Millisecond,
		ConnectRetryMax:        time.Millisecond,
	}
	resp, err := client.CreateResponse(context.Background(), Request{Model: "gpt-5.6-terra"}, Credentials{AccountID: "account-1"}, Route{})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if got := transportAttempts.Load(); got != 4 {
		t.Fatalf("transport attempts = %d, want three connect retries plus one generation attempt", got)
	}
	if got := CreateResponseAttempts(resp, nil); got != 1 {
		t.Fatalf("generation attempts = %d, want 1", got)
	}
}

func TestClientCreateResponseConnectRetryStopsOnCallerCancellation(t *testing.T) {
	started := make(chan struct{}, 1)
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		select {
		case started <- struct{}{}:
		default:
		}
		return nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection unavailable")}
	})
	client := Client{
		HTTPClient:          &http.Client{Transport: transport},
		ConnectRetryInitial: time.Hour,
		ConnectRetryMax:     time.Hour,
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := client.CreateResponse(ctx, Request{Model: "gpt-5.6-terra"}, Credentials{AccountID: "account-1"}, Route{})
		done <- err
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("connect retry did not stop on caller cancellation")
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

func TestClientCreateResponseSharesAttemptBudgetWithHTTPBetaFallback(t *testing.T) {
	t.Setenv("CLAUDODEX_HTTP_BETA_HEADER", "1")
	var attempts atomic.Int32
	client := Client{
		BaseURL: "https://codex.invalid",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			attempt := attempts.Add(1)
			notifyRequestWritten(r)
			if attempt == 1 {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"unknown beta header"}}`)),
					Request:    r,
				}, nil
			}
			<-r.Context().Done()
			return nil, r.Context().Err()
		})},
		ResponseHeaderTimeout: 20 * time.Millisecond,
	}

	_, err := client.CreateResponse(context.Background(), Request{Model: "gpt-5.6-sol", Stream: true}, Credentials{}, Route{})
	var timeoutErr *ResponseHeaderTimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("error = %v, want ResponseHeaderTimeoutError", err)
	}
	if timeoutErr.Attempts != 2 {
		t.Fatalf("timeout attempts = %d, want shared total of 2", timeoutErr.Attempts)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("attempts = %d, want HTTP beta fallback and timeout capped at 2", got)
	}
}

func TestCreateResponseAttemptsCountsHTTPBetaFallbackError(t *testing.T) {
	t.Setenv("CLAUDODEX_HTTP_BETA_HEADER", "1")
	var attempts atomic.Int32
	client := Client{
		BaseURL: "https://codex.invalid",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			attempt := attempts.Add(1)
			status := http.StatusBadRequest
			if attempt == 2 {
				status = http.StatusServiceUnavailable
			}
			return &http.Response{
				StatusCode: status,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"failed"}}`)),
				Request:    r,
			}, nil
		})},
	}

	resp, err := client.CreateResponse(context.Background(), Request{Model: "gpt-5.6-sol", Stream: true}, Credentials{}, Route{})
	if got := CreateResponseAttempts(resp, err); got != 2 {
		t.Fatalf("CreateResponseAttempts = %d, want 2", got)
	}
	var upstreamErr *UpstreamError
	if !errors.As(err, &upstreamErr) || upstreamErr.Status != http.StatusServiceUnavailable {
		t.Fatalf("error = %v, want wrapped HTTP 503 UpstreamError", err)
	}
}

func TestClientCreateResponseRetriesOnceAfterResponseHeaderTimeout(t *testing.T) {
	var attempts atomic.Int32
	var routeHeaders [][3]string
	client := Client{
		BaseURL: "https://codex.invalid",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			attempt := attempts.Add(1)
			routeHeaders = append(routeHeaders, [3]string{
				r.Header.Get("x-client-request-id"),
				r.Header.Get("session-id"),
				r.Header.Get("thread-id"),
			})
			notifyRequestWritten(r)
			if attempt == 1 {
				<-r.Context().Done()
				return nil, r.Context().Err()
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("event: response.completed\n\n")),
				Request:    r,
			}, nil
		})},
		ResponseHeaderTimeout: 20 * time.Millisecond,
	}
	resp, err := client.CreateResponse(context.Background(), Request{Model: "gpt-5.6-sol", Stream: true}, Credentials{}, Route{})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if got := attempts.Load(); got != 2 {
		t.Fatalf("attempts = %d, want exactly 2", got)
	}
	if got := resp.Header.Get(responseHeaderRetriesHeader); got != "1" {
		t.Fatalf("%s = %q, want 1", responseHeaderRetriesHeader, got)
	}
	if len(routeHeaders) != 2 || routeHeaders[0][0] == "" || routeHeaders[0] != routeHeaders[1] {
		t.Fatalf("route headers changed across retry: %#v", routeHeaders)
	}
	if routeHeaders[0][0] != routeHeaders[0][1] || routeHeaders[0][1] != routeHeaders[0][2] {
		t.Fatalf("generated route headers are inconsistent: %#v", routeHeaders[0])
	}
}

func TestClientCreateResponseStopsAfterOneResponseHeaderTimeoutRetry(t *testing.T) {
	var attempts atomic.Int32
	client := Client{
		BaseURL: "https://codex.invalid",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			attempts.Add(1)
			notifyRequestWritten(r)
			<-r.Context().Done()
			return nil, r.Context().Err()
		})},
		ResponseHeaderTimeout: 20 * time.Millisecond,
	}
	_, err := client.CreateResponse(context.Background(), Request{Model: "gpt-5.6-sol", Stream: true}, Credentials{}, Route{})
	var timeoutErr *ResponseHeaderTimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("error = %v, want ResponseHeaderTimeoutError", err)
	}
	if timeoutErr.Attempts != 2 {
		t.Fatalf("timeout attempts = %d, want 2", timeoutErr.Attempts)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("attempts = %d, want exactly 2", got)
	}
}

func TestClientCreateResponseHeaderTimeoutCoversTransportWithoutHTTPTrace(t *testing.T) {
	var attempts atomic.Int32
	client := Client{
		BaseURL: "https://codex.invalid",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			attempts.Add(1)
			<-req.Context().Done()
			return nil, req.Context().Err()
		})},
		ResponseHeaderTimeout: 20 * time.Millisecond,
	}

	_, err := client.CreateResponse(context.Background(), Request{Model: "gpt-5.6-sol", Stream: true}, Credentials{}, Route{})
	var timeoutErr *ResponseHeaderTimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("error = %v, want ResponseHeaderTimeoutError", err)
	}
	if timeoutErr.Attempts != 2 {
		t.Fatalf("timeout attempts = %d, want 2", timeoutErr.Attempts)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("attempts = %d, want exactly 2", got)
	}
}

func TestClientCreateResponseHeaderTimeoutDoesNotLimitStreamingBody(t *testing.T) {
	const headerTimeout = 20 * time.Millisecond
	bodyRelease := make(chan struct{})
	requestCanceled := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		select {
		case <-bodyRelease:
			_, _ = io.WriteString(w, "event: response.completed\n")
			_, _ = io.WriteString(w, `data: {"type":"response.completed","response":{}}`+"\n\n")
		case <-r.Context().Done():
			close(requestCanceled)
		}
	}))
	defer upstream.Close()
	defer func() {
		select {
		case <-bodyRelease:
		default:
			close(bodyRelease)
		}
	}()

	client := Client{
		BaseURL:               upstream.URL,
		HTTPClient:            upstream.Client(),
		ResponseHeaderTimeout: headerTimeout,
	}
	resp, err := client.CreateResponse(context.Background(), Request{Model: "gpt-5.6-sol", Stream: true}, Credentials{}, Route{})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	select {
	case <-requestCanceled:
		t.Fatal("request context was canceled after response headers arrived")
	case <-time.After(3 * headerTimeout):
	}
	close(bodyRelease)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read delayed response body: %v", err)
	}
	if !strings.Contains(string(body), "response.completed") {
		t.Fatalf("delayed response body = %q", body)
	}
}

func TestClientCreateResponsePreservesCallerCancellationWhileWaitingForHeaders(t *testing.T) {
	var attempts atomic.Int32
	requestStarted := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	client := Client{
		BaseURL: "https://codex.invalid",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			attempts.Add(1)
			notifyRequestWritten(r)
			close(requestStarted)
			<-r.Context().Done()
			return nil, r.Context().Err()
		})},
		ResponseHeaderTimeout: time.Second,
	}
	done := make(chan error, 1)
	go func() {
		_, err := client.CreateResponse(ctx, Request{Model: "gpt-5.6-sol", Stream: true}, Credentials{}, Route{})
		done <- err
	}()
	<-requestStarted
	cancel()
	err := <-done
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if got := attempts.Load(); got != 1 {
		t.Fatalf("attempts = %d, want no retry after caller cancellation", got)
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
		_, _ = io.WriteString(w, `{"models":[{"slug":"gpt-5.5","display_name":"GPT-5.5","context_window":272000,"max_context_window":272000,"supported_reasoning_levels":[{"effort":"low","description":"fast"},{"effort":"max","description":"deep"}],"supported_in_api":true,"visibility":"list"}]}`)
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
	if got := models[0].SupportedReasoningLevels; len(got) != 2 || got[0].Effort != "low" || got[1].Effort != "max" {
		t.Fatalf("supported reasoning levels = %#v", got)
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

func TestModelsClientVersionUsesGPT56CatalogMinimum(t *testing.T) {
	if got := modelsClientVersion("0.1.4"); got != "0.144.3" {
		t.Fatalf("modelsClientVersion(0.1.4) = %q, want GPT-5.6 catalog minimum", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func notifyRequestWritten(req *http.Request) {
	if trace := httptrace.ContextClientTrace(req.Context()); trace != nil && trace.WroteRequest != nil {
		trace.WroteRequest(httptrace.WroteRequestInfo{})
	}
}
