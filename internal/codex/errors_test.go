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
