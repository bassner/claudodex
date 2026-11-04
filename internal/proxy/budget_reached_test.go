package proxy

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/convert"
)

func TestBudgetReachedNeverRetriesWithAdviceOrImplicitResume(t *testing.T) {
	for _, implicit := range []bool{false, true} {
		if shouldRetryStream(nil, upstreamStreamEventError{code: "budget_reached", retryable: true, retrySet: true}, implicit) {
			t.Fatalf("exhausted budget retried with implicit resume=%t", implicit)
		}
	}
	for _, streaming := range []bool{false, true} {
		t.Run(map[bool]string{false: "json", true: "sse"}[streaming], func(t *testing.T) {
			home := t.TempDir()
			saveTestAuth(t, home, "access-1")
			var calls atomic.Int32
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"error":{"code":"budget_reached","retryable":true,"headers":{"retry-after":"0"}}}`)
			}))
			defer upstream.Close()
			server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
			addr, err := server.Start("127.0.0.1", 0)
			if err != nil {
				t.Fatal(err)
			}
			defer server.Close()
			body := `{"model":"sonnet","stream":` + map[bool]string{false: "false", true: "true"}[streaming] + `,"messages":[{"role":"user","content":"x"}]}`
			resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			output := readAllString(t, resp)
			if calls.Load() != 1 || resp.StatusCode != 400 || !strings.Contains(output, "invalid_request_error") || !strings.Contains(output, codex.BudgetReachedFallbackMessage) {
				t.Fatalf("calls=%d status=%d body=%s", calls.Load(), resp.StatusCode, output)
			}
		})
	}
}

type budgetFailureReader struct {
	first     string
	laterRead bool
}

func (r *budgetFailureReader) Read(p []byte) (int, error) {
	if r.first != "" {
		n := copy(p, r.first)
		r.first = r.first[n:]
		return n, nil
	}
	r.laterRead = true
	return 0, errors.New("read after terminal failure instead of ending immediately")
}

func TestBudgetReachedStopsBeforeLaterCompletion(t *testing.T) {
	for _, shape := range []string{
		"event: error\ndata: {\"error\":{\"code\":\"budget_reached\",\"headers\":{\"Retry-After\":\"0\"},\"retryable\":true}}\n\n",
		"event: response.failed\ndata: {\"type\":\"response.failed\",\"response\":{\"error\":{\"code\":\"budget_reached\",\"message\":\"Session budget exhausted\",\"headers\":{\"retry-after\":\"0\"},\"retryable\":true}}}\n\n",
	} {
		for _, streaming := range []bool{false, true} {
			reader := &budgetFailureReader{first: shape}
			recorder := httptest.NewRecorder()
			server := New(Config{})
			var err error
			if streaming {
				err = server.streamAnthropicWithSchemas(recorder, reader, nil, "sonnet", nil, "", 0, 0, convert.Usage{}, "", codex.Request{}, nil, true, nil)
			} else {
				err = server.writeNonStreamingMessageWithSchemas(recorder, reader, nil, "sonnet", nil, "", 0, 0, convert.Usage{}, "", codex.Request{}, nil, nil)
				if err == nil || shouldRetryStream(nil, err, true) {
					t.Fatalf("nonstream failure missing or retryable: %v", err)
				}
				writeMappedStreamError(recorder, err)
			}
			if reader.laterRead {
				t.Fatal("waited for more data after terminal budget failure")
			}
			if streaming && err != nil {
				t.Fatal(err)
			}
			body := recorder.Body.String()
			if strings.Count(body, `"type":"invalid_request_error"`) != 1 || strings.Contains(body, "message_stop") || strings.Contains(body, "api_error") {
				t.Fatalf("wrong terminal error: %s", body)
			}
		}
	}
}
