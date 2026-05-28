package convert

import "testing"

func TestStripAnthropicBillingHeader(t *testing.T) {
	first := "prefix\nx-anthropic-billing-header: cc_version=2.1.92; cch=abc;\nsuffix"
	second := "prefix\nX-Anthropic-Billing-Header: cc_version=2.1.92; cch=def;\r\nsuffix"
	want := "prefix\nsuffix"
	if got := StripAnthropicBillingHeader(first); got != want {
		t.Fatalf("first = %q, want %q", got, want)
	}
	if got := StripAnthropicBillingHeader(second); got != want {
		t.Fatalf("second = %q, want %q", got, want)
	}
}
