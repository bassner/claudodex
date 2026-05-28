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
		{"claude opus xhigh", "claude-opus-4-7", "xhigh", 0, EffortXHigh},
		{"claude sonnet legacy xhigh", "sonnet[1m]", "", 32000, EffortXHigh},
		{"output max", "gpt-5.4", "max", 0, EffortXHigh},
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
