package anthropic

import "testing"

func TestMapModel(t *testing.T) {
	tests := map[string]string{
		"claude-opus-5":        ModelOpus,
		"claude-opus-4-8":      ModelOpus,
		"claude-opus-4-7":      ModelOpus,
		"best":                 ModelOpus,
		"claude-sonnet-4-6":    ModelSonnet,
		"sonnet[1m]":           ModelSonnet,
		"claude-haiku-4-5":     ModelHaiku,
		"small-fast":           ModelHaiku,
		"gpt-6.1-sol":          ModelOpus,
		"gpt-6.1-sol[1m]":      ModelOpus,
		"gpt-6.1-terra":        ModelSonnet,
		"gpt-6.1-terra[1m]":    ModelSonnet,
		"gpt-6-luna":           ModelHaiku,
		"gpt-6-luna[1m]":       ModelHaiku,
		"future-unknown-model": ModelOpus,
	}
	for input, want := range tests {
		if got := MapModel(input); got != want {
			t.Fatalf("MapModel(%q) = %q, want %q", input, got, want)
		}
	}
}
