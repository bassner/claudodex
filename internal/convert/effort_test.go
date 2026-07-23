package convert

import "testing"

func TestMapReasoningEffort(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		outputEffort string
		budget       int
		want         ReasoningEffort
	}{
		{"output xhigh", "gpt-5.5", "xhigh", 0, EffortXHigh},
		{"claude opus 4.8 xhigh", "claude-opus-4-8", "xhigh", 0, EffortXHigh},
		{"claude opus xhigh", "claude-opus-4-7", "xhigh", 0, EffortXHigh},
		{"claude sonnet legacy xhigh", "sonnet[1m]", "", 32000, EffortXHigh},
		{"output max", "gpt-5.4", "max", 0, EffortXHigh},
		{"output ultracode", "gpt-5.5", "ultracode", 0, EffortXHigh},
		{"sol max", "gpt-5.6-sol", "max", 0, EffortMax},
		{"terra ultracode", "gpt-5.6-terra", "ultracode", 0, EffortMax},
		{"luna ultracode", "gpt-5.6-luna", "ultracode", 0, EffortMax},
		{"legacy xhigh", "gpt-5.4-mini", "", 32000, EffortXHigh},
		{"legacy high", "gpt-5.5", "", 20000, EffortHigh},
		{"legacy medium", "gpt-5.5", "", 8000, EffortMedium},
		{"legacy low", "gpt-5.5", "", 1000, EffortLow},
		{"mini default", "gpt-5.4-mini", "", 0, EffortLow},
		{"claude haiku default", "claude-haiku-4-5", "", 0, EffortLow},
		{"small fast default", "small-fast", "", 0, EffortLow},
		{"sonnet default", "gpt-5.4", "", 0, EffortMedium},
		{"unsupported xhigh", "gpt-5.1", "max", 0, EffortHigh},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MapReasoningEffort(tt.model, tt.outputEffort, tt.budget); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGPT56ClaudeEffortsMatchAdvertisedLevels(t *testing.T) {
	advertised := map[ReasoningEffort]bool{
		EffortLow:    true,
		EffortMedium: true,
		EffortHigh:   true,
		EffortXHigh:  true,
		EffortMax:    true,
	}
	for _, model := range []string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"} {
		for _, claudeEffort := range []string{"low", "medium", "high", "xhigh", "max"} {
			got := MapReasoningEffort(model, claudeEffort, 0)
			if !advertised[got] {
				t.Fatalf("%s effort %s mapped to unadvertised Codex level %q", model, claudeEffort, got)
			}
			if string(got) != claudeEffort {
				t.Fatalf("%s effort %s mapped to %q", model, claudeEffort, got)
			}
		}
		if got := MapReasoningEffort(model, "ultracode", 0); got != EffortMax {
			t.Fatalf("%s ultracode mapped to %q, want Codex max", model, got)
		}
	}
}
