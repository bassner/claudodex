package convert

import (
	"testing"

	"github.com/bassner/claudodex/internal/codex"
)

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

func TestGPT6FourTierEfforts(t *testing.T) {
	for _, model := range []string{"gpt-6.1-sol", "gpt-6.1-terra", "gpt-6-luna", "gpt-6-astra"} {
		for _, effort := range []string{"low", "medium", "high", "xhigh", "max"} {
			if got := MapReasoningEffort(model, effort, 0); string(got) != effort {
				t.Errorf("%s effort %s = %s", model, effort, got)
			}
		}
		if got := MapReasoningEffort(model, "ultracode", 0); got != EffortMax {
			t.Errorf("%s ultracode = %s", model, got)
		}
	}
}

func TestMapReasoningEffortUsesActualAdvertisedLevels(t *testing.T) {
	model := codex.ModelInfo{Slug: "gpt-99-sol", DefaultReasoningLevel: "low", SupportedReasoningLevels: []codex.ReasoningEffortPreset{
		{Effort: "low"}, {Effort: "high"}, {Effort: "max"}, {Effort: "ultra"}, {Effort: "unknown"},
	}}
	for _, test := range []struct {
		effort string
		budget int
		want   ReasoningEffort
	}{
		{"", 0, EffortLow}, {"auto", 0, EffortLow}, {"auto", 32000, EffortLow},
		{"minimal", 0, EffortLow}, {"medium", 0, EffortLow}, {"xhigh", 0, EffortHigh},
		{"max", 0, EffortMax}, {"ultracode", 0, EffortMax}, {"ultra", 0, EffortLow},
		{"", 1000, EffortLow}, {"", 8000, EffortLow}, {"", 20000, EffortHigh}, {"", 32000, EffortHigh},
	} {
		got, err := MapReasoningEffortForModel(model, test.effort, test.budget)
		if err != nil || got != test.want {
			t.Errorf("effort %q budget %d = %q, %v; want %q", test.effort, test.budget, got, err, test.want)
		}
	}
}

func TestMapReasoningEffortSupportsNoneAndMinimal(t *testing.T) {
	model := codex.ModelInfo{Slug: "gpt-99-luna", DefaultReasoningLevel: "none", SupportedReasoningLevels: []codex.ReasoningEffortPreset{{Effort: "none"}, {Effort: "minimal"}, {Effort: "high"}}}
	for input, want := range map[string]ReasoningEffort{"": EffortNone, "auto": EffortNone, "none": EffortNone, "minimal": EffortMinimal, "medium": EffortMinimal, "max": EffortHigh} {
		got, err := MapReasoningEffortForModel(model, input, 0)
		if err != nil || got != want {
			t.Errorf("%q = %q, %v; want %q", input, got, err, want)
		}
	}
}

func TestMapReasoningEffortUnknownDefaultsAndMinimumClamp(t *testing.T) {
	for _, test := range []struct {
		levels []codex.ReasoningEffortPreset
		want   ReasoningEffort
	}{
		{[]codex.ReasoningEffortPreset{{Effort: "low"}, {Effort: "max"}}, EffortLow},
		{[]codex.ReasoningEffortPreset{{Effort: "xhigh"}, {Effort: "max"}}, EffortXHigh},
	} {
		model := codex.ModelInfo{Slug: "gpt-99-sol", DefaultReasoningLevel: "ultra", SupportedReasoningLevels: test.levels}
		got, err := MapReasoningEffortForModel(model, "auto", 0)
		if err != nil || got != test.want {
			t.Errorf("unknown default = %q, %v; want %q", got, err, test.want)
		}
	}
	if _, err := MapReasoningEffortForModel(codex.ModelInfo{Slug: "gpt-workflow", SupportedReasoningLevels: []codex.ReasoningEffortPreset{{Effort: "ultra"}}}, "", 0); err == nil {
		t.Fatal("workflow-only model accepted")
	}
}
