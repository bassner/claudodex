package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/convert"
	"github.com/gorilla/websocket"
)

func TestMessagesLargeWebSocketFallsBackToHTTP(t *testing.T) {
	for _, tc := range []struct {
		name         string
		resume       bool
		largeInput   bool
		largeHistory bool
		missingID    bool
		httpStatus   int
	}{
		{name: "large_full_request", largeInput: true},
		{name: "early_close_1009"},
		{name: "resumed_close_1009", resume: true},
		{name: "large_delta", resume: true, largeInput: true},
		{name: "missing_reference_with_large_replay", resume: true, largeHistory: true, missingID: true},
		{name: "http_also_rejects_size", httpStatus: http.StatusRequestEntityTooLarge},
		{name: "http_rejects_input", httpStatus: http.StatusBadRequest},
		{name: "http_rate_limit", httpStatus: http.StatusTooManyRequests},
	} {
		for _, stream := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/stream=%t", tc.name, stream), func(t *testing.T) {
				t.Setenv("CLAUDODEX_FORCE_CODEX_WEBSOCKET", "1")
				home := t.TempDir()
				saveTestAuth(t, home, "access-1")
				var wsCalls, httpCalls atomic.Int32
				opaque := "retained-encrypted-reasoning"
				if tc.largeHistory {
					opaque = strings.Repeat("A", 9<<20)
				}
				upgrader := websocket.Upgrader{}
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if websocket.IsWebSocketUpgrade(r) {
						conn, err := upgrader.Upgrade(w, r, nil)
						if err != nil {
							t.Error(err)
							return
						}
						defer conn.Close()
						for {
							var req map[string]any
							if err := conn.ReadJSON(&req); err != nil {
								return
							}
							n := wsCalls.Add(1)
							if tc.resume && n == 1 {
								writeWSJSON(t, conn, map[string]any{"type": "response.created", "response": map[string]any{"id": "resp-size-1"}})
								writeWSJSON(t, conn, map[string]any{"type": "response.output_item.done", "output_index": 0, "item": map[string]any{"type": "reasoning", "id": "reason-size", "encrypted_content": opaque}})
								writeWSJSON(t, conn, map[string]any{"type": "response.output_item.done", "output_index": 1, "item": map[string]any{"type": "function_call", "call_id": "call_size", "name": "Read", "arguments": `{"file_path":"fixture.pdf"}`}})
								writeWSJSON(t, conn, map[string]any{"type": "response.completed", "response": map[string]any{"id": "resp-size-1", "stop_reason": "tool_calls", "usage": map[string]any{"input_tokens": 10, "output_tokens": 1}}})
								continue
							}
							if tc.resume && req["previous_response_id"] != "resp-size-1" {
								t.Errorf("delta lost response reference: %v", req["previous_response_id"])
							}
							if tc.missingID {
								writeWSJSON(t, conn, map[string]any{"type": "error", "error": map[string]any{"message": "Invalid `previous_response_id`."}})
							} else {
								_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseMessageTooBig, "message too big"), time.Now().Add(time.Second))
							}
							return
						}
					}
					httpCalls.Add(1)
					var req map[string]any
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Error(err)
						return
					}
					if req["previous_response_id"] != nil {
						t.Errorf("HTTP received a WebSocket-only delta reference")
					}
					input, _ := json.Marshal(req["input"])
					if !strings.Contains(string(input), "Read the fixture") || !strings.Contains(string(input), "stable-system-guidance") {
						t.Error("HTTP fallback lost initial context or developer guidance")
					}
					if tc.resume && (!strings.Contains(string(input), opaque) || !strings.Contains(string(input), "function_call_output")) {
						t.Error("HTTP fallback lost encrypted reasoning or tool results")
					}
					if tc.largeInput && len(input) < 9<<20 {
						t.Errorf("large payload was truncated: %d bytes", len(input))
					}
					if tc.httpStatus != 0 {
						w.WriteHeader(tc.httpStatus)
						_, _ = io.WriteString(w, `{"error":{"message":"HTTP request is too large"}}`)
						return
					}
					w.Header().Set("content-type", "text/event-stream")
					_, _ = io.WriteString(w, "event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"size-recovered\"}\n\nevent: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp-http-size\",\"usage\":{\"input_tokens\":20,\"output_tokens\":1}}}\n\n")
				}))
				defer upstream.Close()
				server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
				defer server.Close()
				messages := []any{map[string]any{"role": "user", "content": "Read the fixture"}}
				request := func() *httptest.ResponseRecorder {
					body, err := json.Marshal(map[string]any{"model": "opus", "system": "stable-system-guidance", "stream": stream, "messages": messages, "tools": []any{map[string]any{"name": "Read", "input_schema": map[string]any{"type": "object"}}}})
					if err != nil {
						t.Fatal(err)
					}
					r := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(body)))
					r.Header.Set("x-claude-code-session-id", "size-session")
					w := httptest.NewRecorder()
					server.handleMessages(w, r)
					return w
				}
				if tc.resume {
					first := request()
					if first.Code != http.StatusOK || !strings.Contains(first.Body.String(), "tool_use") {
						t.Fatalf("setup tool call failed: %d %s", first.Code, first.Body.String())
					}
					result := "fixture-result"
					if tc.largeInput {
						result += strings.Repeat("x", 9<<20)
					}
					messages = append(messages,
						map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "tool_use", "id": "call_size", "name": "Read", "input": map[string]any{"file_path": "fixture.pdf"}}}},
						map[string]any{"role": "user", "content": []any{map[string]any{"type": "tool_result", "tool_use_id": "call_size", "content": result}}})
				} else if tc.largeInput {
					messages = append(messages, map[string]any{"role": "user", "content": strings.Repeat("x", 18<<20)})
				}
				got := request()
				wantStatus := http.StatusOK
				if tc.httpStatus != 0 {
					wantStatus = tc.httpStatus
				} else if !strings.Contains(got.Body.String(), "size-recovered") {
					t.Errorf("fallback response missing: %.500s", got.Body.String())
				}
				if got.Code != wantStatus || httpCalls.Load() != 1 {
					t.Errorf("status=%d HTTP calls=%d, want %d and 1: %.500s", got.Code, httpCalls.Load(), wantStatus, got.Body.String())
				}
				wantWS := int32(1)
				if tc.resume && !tc.largeInput {
					wantWS = 2
				} else if !tc.resume && tc.largeInput {
					wantWS = 0
				}
				if wsCalls.Load() != wantWS {
					t.Errorf("WS generation requests=%d, want %d", wsCalls.Load(), wantWS)
				}
				if !tc.resume && tc.httpStatus == 0 {
					messages = append(messages, map[string]any{"role": "user", "content": "A new turn after size recovery"})
					followup := request()
					if followup.Code != http.StatusOK || !strings.Contains(followup.Body.String(), "size-recovered") || wsCalls.Load() != wantWS || httpCalls.Load() != 2 {
						t.Errorf("later turn repeated a rejected WS request: status=%d WS=%d HTTP=%d", followup.Code, wsCalls.Load(), httpCalls.Load())
					}
					if !server.shouldUseCodexWebSocket(codex.Request{Tools: []codex.Tool{{Name: "Read"}}}, codex.Route{ThreadID: "unrelated-session"}) {
						t.Error("size fallback disabled WebSocket for an unrelated thread")
					}
				}
			})
		}
	}
}

func TestCommittedWebSocketSizeFailureDoesNotReplayOutput(t *testing.T) {
	closed := &websocket.CloseError{Code: websocket.CloseMessageTooBig, Text: "message too big"}
	reader := io.MultiReader(strings.NewReader("event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"already-visible\"}\n\n"), sizeFailureReader{closed})
	server := New(Config{})
	w := httptest.NewRecorder()
	if err := server.streamAnthropicWithSchemas(w, reader, nil, "opus", nil, "", 0, 0, convert.Usage{}, "", codex.Request{}, nil, false, nil); err != nil {
		t.Fatalf("committed response must not trigger a replay: %v", err)
	}
	if strings.Count(w.Body.String(), "already-visible") != 1 || !strings.Contains(w.Body.String(), "invalid_request_error") || strings.Contains(w.Body.String(), "overloaded_error") {
		t.Fatalf("expected visible output once and a useful terminal size error: %s", w.Body.String())
	}
	for _, mapError := range []func(http.ResponseWriter, error){writeMappedStreamError, writeMappedUpstreamError} {
		w := httptest.NewRecorder()
		mapError(w, closed)
		if w.Code != http.StatusRequestEntityTooLarge || !strings.Contains(w.Body.String(), "too large") {
			t.Fatalf("size error mapped to retryable failure: %d %s", w.Code, w.Body.String())
		}
	}
}

type sizeFailureReader struct{ err error }

func (r sizeFailureReader) Read([]byte) (int, error) { return 0, r.err }
