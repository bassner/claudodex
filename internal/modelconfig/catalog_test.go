package modelconfig

import "testing"

func TestConfigMapModelUsesOverrides(t *testing.T) {
	cfg := Config{Opus: "gpt-opus-next", Sonnet: "gpt-sonnet-next", Haiku: "gpt-haiku-next"}
	tests := map[string]string{
		"opus":              "gpt-opus-next",
		"claude-opus-5":     "gpt-opus-next",
		"gpt-5.6-sol":       "gpt-opus-next",
		"claude-sonnet-4-6": "gpt-sonnet-next",
		"gpt-5.6-terra[1m]": "gpt-sonnet-next",
		"small-fast":        "gpt-haiku-next",
		"gpt-5.6-luna":      "gpt-haiku-next",
		"gpt-haiku-next":    "gpt-haiku-next",
	}
	for input, want := range tests {
		if got := cfg.MapModel(input); got != want {
			t.Fatalf("MapModel(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestClaudeAliasSpecsIncludeCurrentOpusDefault(t *testing.T) {
	specs := ClaudeAliasSpecs(Default())
	if len(specs) == 0 {
		t.Fatal("ClaudeAliasSpecs returned no aliases")
	}
	first := specs[0]
	if first.ID != "claude-opus-5" || first.Family != FamilyOpus {
		t.Fatalf("first Claude alias = %#v, want claude-opus-5 Opus alias", first)
	}
	if DefaultClaudeRequestModel != "claude-opus-5" {
		t.Fatalf("DefaultClaudeRequestModel = %q, want claude-opus-5", DefaultClaudeRequestModel)
	}
}

func TestFamilyForModelRejectsRetiredFourthTierAliases(t *testing.T) {
	for _, model := range []string{"fable", "claude-fable-5[1m]", "mythos-5"} {
		if family, ok := FamilyForModel(model); ok {
			t.Fatalf("FamilyForModel(%q) = %q, true; want no family", model, family)
		}
	}
}

func TestDirectModelSpecsUseConfiguredTargets(t *testing.T) {
	specs := DirectModelSpecs(Config{Opus: "gpt-opus-next", Sonnet: "gpt-sonnet-next", Haiku: "gpt-haiku-next"})
	var found bool
	for _, spec := range specs {
		if spec.ID == "gpt-sonnet-next" && spec.Family == FamilySonnet {
			found = true
		}
		if spec.ID == "gpt-sonnet-next[1m]" {
			t.Fatalf("long-context runtime suffix leaked into direct specs: %#v", specs)
		}
		if spec.ID == DefaultSonnetModel {
			t.Fatalf("default sonnet target leaked into direct specs: %#v", specs)
		}
	}
	if !found {
		t.Fatalf("configured sonnet runtime target missing: %#v", specs)
	}
}

func TestDirectRuntimeModelSpecsUseLongContextAliases(t *testing.T) {
	specs := DirectRuntimeModelSpecs(Config{Opus: "gpt-opus-next", Sonnet: "gpt-sonnet-next", Haiku: "gpt-haiku-next"})
	var found bool
	for _, spec := range specs {
		if spec.ID == "gpt-sonnet-next[1m]" && spec.DisplayName == "gpt-sonnet-next" && spec.Family == FamilySonnet {
			found = true
		}
		if spec.ID == "gpt-sonnet-next" {
			t.Fatalf("plain target leaked into runtime specs: %#v", specs)
		}
	}
	if !found {
		t.Fatalf("configured sonnet runtime alias missing: %#v", specs)
	}
}

func TestStripLongContextRemovesRepeatedSuffixes(t *testing.T) {
	tests := map[string]string{
		"gpt-5.6-sol[1m]":      "gpt-5.6-sol",
		"gpt-5.6-sol[1m][1m]":  "gpt-5.6-sol",
		"gpt-5.6-sol[1M] [1m]": "gpt-5.6-sol",
		" gpt-5.6-luna[1m]":    "gpt-5.6-luna",
	}
	for input, want := range tests {
		if got := StripLongContext(input); got != want {
			t.Fatalf("StripLongContext(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSupportsXHighPreservesKnownLegacyModels(t *testing.T) {
	cfg := Default()
	for _, model := range []string{"gpt-5.5", "gpt-5.4[1m]", "gpt-5.4-mini"} {
		if !cfg.SupportsXHigh(model) {
			t.Errorf("SupportsXHigh(%q) = false, want true", model)
		}
	}
	if cfg.SupportsXHigh("gpt-5.1") {
		t.Error("SupportsXHigh(gpt-5.1) = true, want false")
	}
}
