package codex

import "testing"

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
