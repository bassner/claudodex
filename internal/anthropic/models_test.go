package anthropic

import "testing"

func TestMapModel(t *testing.T) {
	tests := map[string]string{
		"claude-opus-4-8":      ModelOpus,
		"claude-opus-4-7":      ModelOpus,
		"best":                 ModelOpus,
		"claude-sonnet-4-6":    ModelSonnet,
		"sonnet[1m]":           ModelSonnet,
		"claude-haiku-4-5":     ModelHaiku,
		"small-fast":           ModelHaiku,
		"gpt-5.5":              ModelOpus,
		"gpt-5.5[1m]":          ModelOpus,
		"gpt-5.4":              ModelSonnet,
		"gpt-5.4[1m]":          ModelSonnet,
		"gpt-5.4-mini":         ModelHaiku,
		"gpt-5.4-mini[1m]":     ModelHaiku,
		"future-unknown-model": ModelOpus,
	}
	for input, want := range tests {
		if got := MapModel(input); got != want {
			t.Fatalf("MapModel(%q) = %q, want %q", input, got, want)
		}
	}
}
