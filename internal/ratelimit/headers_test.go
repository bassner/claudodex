package ratelimit

import (
	"net/http"
	"testing"
	"time"
)

func TestFromCodexHeadersAndApplyAnthropicHeaders(t *testing.T) {
	upstream := http.Header{}
	upstream.Set("x-codex-primary-used-percent", "42")
	upstream.Set("x-codex-primary-window-minutes", "300")
	upstream.Set("x-codex-primary-reset-at", "1770000000")
	upstream.Set("x-codex-secondary-used-percent", "17")
	upstream.Set("x-codex-secondary-window-minutes", "10080")
	upstream.Set("x-codex-secondary-reset-at", "1770500000")

	snapshot := FromCodexHeaders(upstream)
	if snapshot == nil || snapshot.FiveHour == nil || snapshot.SevenDay == nil {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	out := http.Header{}
	ApplyAnthropicHeaders(out, snapshot, false)
	if out.Get("anthropic-ratelimit-unified-status") != "allowed" {
		t.Fatalf("status = %q", out.Get("anthropic-ratelimit-unified-status"))
	}
	if out.Get("anthropic-ratelimit-unified-5h-utilization") != "0.42" {
		t.Fatalf("5h utilization = %q", out.Get("anthropic-ratelimit-unified-5h-utilization"))
	}
	if out.Get("anthropic-ratelimit-unified-7d-utilization") != "0.17" {
		t.Fatalf("7d utilization = %q", out.Get("anthropic-ratelimit-unified-7d-utilization"))
	}
	if out.Get("anthropic-ratelimit-unified-representative-claim") != "five_hour" {
		t.Fatalf("claim = %q", out.Get("anthropic-ratelimit-unified-representative-claim"))
	}
}

func TestSnapshotFromRetryAfter(t *testing.T) {
	headers := http.Header{}
	headers.Set("retry-after", "60")
	snapshot := SnapshotFromRetryAfter(headers, time.Unix(1000, 0))
	if snapshot == nil || snapshot.FiveHour == nil || snapshot.FiveHour.ResetAt != 1060 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	out := http.Header{}
	ApplyAnthropicHeaders(out, snapshot, true)
	if out.Get("anthropic-ratelimit-unified-status") != "rejected" {
		t.Fatalf("status = %q", out.Get("anthropic-ratelimit-unified-status"))
	}
	if out.Get("anthropic-ratelimit-unified-reset") != "1060" {
		t.Fatalf("reset = %q", out.Get("anthropic-ratelimit-unified-reset"))
	}
}
