package convert

import "regexp"

var billingHeaderLine = regexp.MustCompile(`(?mi)^x-anthropic-billing-header:[^\r\n]*(?:\r?\n|$)`)

func StripAnthropicBillingHeader(text string) string {
	return billingHeaderLine.ReplaceAllString(text, "")
}
