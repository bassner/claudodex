package codex

import (
	"testing"
	"time"
)

func TestParseResponseFailureReadsEmbeddedRetryAdvice(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantDelay    time.Duration
		wantDelaySet bool
		wantRetry    bool
	}{
		{
			name:         "response error mixed case header",
			body:         `{"type":"response.failed","response":{"error":{"code":"server_is_overloaded","message":"busy","headers":{"rEtRy-AfTeR":"0.04"}}}}`,
			wantDelay:    40 * time.Millisecond,
			wantDelaySet: true,
		},
		{
			name:         "top level websocket error array header",
			body:         `{"type":"error","code":"internal_error","message":"retryable","retryable":true,"headers":{"Retry-After":["0"]}}`,
			wantDelaySet: true,
			wantRetry:    true,
		},
		{
			name:         "nested websocket numeric header",
			body:         `{"type":"error","error":{"status":429,"code":"rate_limit_exceeded","message":"busy","headers":{"Retry-After":0.025}}}`,
			wantDelay:    25 * time.Millisecond,
			wantDelaySet: true,
		},
		{
			name:         "malformed nested header falls back to top level",
			body:         `{"type":"error","error":{"status":"429","code":"rate_limit_exceeded","message":"busy","headers":{"Retry-After":"later"}},"headers":{"retry-after":"0.03"}}`,
			wantDelay:    30 * time.Millisecond,
			wantDelaySet: true,
		},
		{
			name:         "case insensitive duplicate uses valid value",
			body:         `{"type":"error","error":{"status":503,"code":"server_is_overloaded","message":"busy","headers":{"RETRY-AFTER":"invalid","retry-after":"0.02"}}}`,
			wantDelay:    20 * time.Millisecond,
			wantDelaySet: true,
		},
		{
			name: "malformed header",
			body: `{"type":"response.failed","response":{"error":{"code":"slow_down","message":"retry in 25ms","headers":{"retry-after":"later"}}}}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ParseResponseFailure([]byte(test.body))
			if got.RetryAfter != test.wantDelay || got.RetryAfterSet != test.wantDelaySet || got.Retryable != test.wantRetry {
				t.Fatalf("retry advice = delay %s set=%v retryable=%v", got.RetryAfter, got.RetryAfterSet, got.Retryable)
			}
		})
	}
}

func TestParseResponseFailureReadsWebSocketStatusWithoutMakingHeaderlessErrorRetryable(t *testing.T) {
	failure := ParseResponseFailure([]byte(`{"type":"error","error":{"status":429,"code":"unknown_error","message":"terminal"}}`))
	if failure.Status != 429 {
		t.Fatalf("status = %d, want 429", failure.Status)
	}
	if failure.RetryAfterSet || failure.Retryable {
		t.Fatalf("headerless failure unexpectedly became retryable: %#v", failure)
	}
}

func TestParseResponseFailureRecognizesBioPolicyShapes(t *testing.T) {
	for _, test := range []struct {
		name        string
		body        string
		wantMessage string
	}{
		{name: "direct missing message", body: `{"code":"bio_policy"}`, wantMessage: BioPolicyFallbackMessage},
		{name: "direct blank message", body: `{"code":"bio_policy","message":"   "}`, wantMessage: BioPolicyFallbackMessage},
		{name: "wrapped custom message", body: `{"error":{"code":"bio_policy","message":"Custom biological safety message"}}`, wantMessage: "Custom biological safety message"},
		{name: "response failed wrapped", body: `{"response":{"error":{"code":"bio_policy"}}}`, wantMessage: BioPolicyFallbackMessage},
	} {
		t.Run(test.name, func(t *testing.T) {
			failure := ParseResponseFailure([]byte(test.body))
			if failure.Code != "bio_policy" || failure.Message != test.wantMessage {
				t.Fatalf("failure = %#v", failure)
			}
		})
	}
}

func TestParseResponseFailureRecognizesFlexUnavailableTransportShapes(t *testing.T) {
	tests := map[string]string{
		"http":      `{"error":{"code":"flex_unavailable","message":"Flex unavailable"}}`,
		"sse error": `{"type":"error","code":"flex_unavailable","message":"Flex unavailable"}`,
		"response":  `{"type":"response.failed","response":{"error":{"code":"flex_unavailable","message":"Flex unavailable"}}}`,
		"websocket": `{"type":"error","error":{"code":"flex_unavailable","message":"Flex unavailable"}}`,
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			failure := ParseResponseFailure([]byte(raw))
			if failure.Code != "flex_unavailable" || failure.Message != "Flex unavailable" {
				t.Fatalf("failure = %#v", failure)
			}
		})
	}
}
