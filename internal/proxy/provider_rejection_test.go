package proxy

import (
	"fmt"
	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/convert"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderRejectionsStopOpenSSEBeforeRetryAdvice(t *testing.T) {
	for _, code := range []string{"context_length_exceeded", "inference_hook_denied", "invalid_access_program", "unsupported_access_program", "access_program_not_enabled"} {
		for _, shape := range []string{"error", "response.failed"} {
			for _, streaming := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/stream=%t", code, shape, streaming), func(t *testing.T) {
					failure := fmt.Sprintf(`{"error":{"code":%q,"message":"rejected","retryable":true,"headers":{"Retry-After":"0"}}}`, code)
					if shape == "response.failed" {
						failure = `{"response":` + failure + `}`
					}
					reader := &budgetFailureReader{first: "event: " + shape + "\ndata: " + failure + "\n\n"}
					rec := httptest.NewRecorder()
					server := New(Config{})
					headers := map[string][]string{"Retry-After": {"0"}}
					if streaming {
						if err := server.streamAnthropicWithSchemas(rec, reader, headers, "sonnet", nil, "", 0, 0, convert.Usage{}, "", codex.Request{}, nil, true, nil); err != nil {
							t.Fatal(err)
						}
					} else {
						err := server.writeNonStreamingMessageWithSchemas(rec, reader, headers, "sonnet", nil, "", 0, 0, convert.Usage{}, "", codex.Request{}, nil, nil)
						if err == nil || shouldRetryStream(nil, err, true) || shouldRetryStream(nil, err, false) {
							t.Fatalf("terminal rejection retried: %v", err)
						}
					}
					if reader.laterRead {
						t.Fatal("read again after terminal rejection")
					}
					if streaming && (!strings.Contains(rec.Body.String(), "rejected") || strings.Contains(rec.Body.String(), "message_stop")) {
						t.Fatal("terminal error missing or completed")
					}
				})
			}
		}
	}
}

func TestWrappedProviderFailureStatusControlsRetry(t *testing.T) {
	for _, status := range []int{400, 503} {
		for _, shape := range []string{"error", "response.failed"} {
			payload := fmt.Sprintf(`{"status":%d,"error":{"code":"context_length_exceeded","message":"context"}}`, status)
			if shape == "response.failed" {
				payload = fmt.Sprintf(`{"status_code":%d,"response":{"error":{"code":"context_length_exceeded","message":"context"}}}`, status)
			}
			// Reduction and retry must retain the envelope status through both modes.
			rec := httptest.NewRecorder()
			server := New(Config{})
			reader := strings.NewReader("event: " + shape + "\ndata: " + payload + "\n\n")
			err := server.writeNonStreamingMessageWithSchemas(rec, reader, nil, "sonnet", nil, "", 0, 0, convert.Usage{}, "", codex.Request{}, nil, nil)
			if err == nil || shouldRetryStream(nil, err, false) != (status == 503) {
				t.Fatalf("status=%d shape=%s retry=%t err=%v", status, shape, shouldRetryStream(nil, err, false), err)
			}
			streamReader := &budgetFailureReader{first: "event: " + shape + "\ndata: " + payload + "\n\n"}
			streamErr := server.streamAnthropicWithSchemas(httptest.NewRecorder(), streamReader, nil, "sonnet", nil, "", 0, 0, convert.Usage{}, "", codex.Request{}, nil, false, nil)
			if streamReader.laterRead || (status == 503 && (streamErr == nil || !shouldRetryStream(nil, streamErr, false))) || (status == 400 && streamErr != nil) {
				t.Fatalf("streaming wrapped status=%d: readPastFailure=%t err=%v", status, streamReader.laterRead, streamErr)
			}
			for _, implicit := range []bool{false, true} {
				event := upstreamStreamEventError{code: "context_length_exceeded", status: status, retryable: true, retrySet: true}
				if shouldRetryStream(nil, event, implicit) != (status == 503) {
					t.Fatalf("status=%d implicit=%t contradicted", status, implicit)
				}
			}
		}
	}
}
