package modelconfig

import "testing"

func TestConfigMapModelUsesOverrides(t *testing.T) {
	cfg := Config{Opus: "gpt-opus-next", Sonnet: "gpt-sonnet-next", Haiku: "gpt-haiku-next", Fable: "gpt-fable-next"}
	tests := map[string]string{
		"opus":              "gpt-opus-next",
		"claude-opus-5":     "gpt-opus-next",
		"gpt-6.1-sol":       "gpt-opus-next",
		"claude-sonnet-4-6": "gpt-sonnet-next",
		"gpt-6.1-terra[1m]": "gpt-sonnet-next",
		"small-fast":        "gpt-haiku-next",
		"gpt-6-luna":        "gpt-haiku-next",
		"gpt-haiku-next":    "gpt-haiku-next",
		"claude-fable-5":    "gpt-fable-next",
		"mythos-5":          "gpt-fable-next",
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

func TestFamilyForModelRecognizesFourthTierAliases(t *testing.T) {
	for _, model := range []string{"fable", "claude-fable-5[1m]", "mythos-5"} {
		if family, ok := FamilyForModel(model); !ok || family != FamilyFable {
			t.Fatalf("FamilyForModel(%q) = %q, true; want fable family", model, family)
		}
	}
}

func TestDirectModelSpecsUseConfiguredTargets(t *testing.T) {
	specs := DirectModelSpecs(Config{Opus: "gpt-opus-next", Sonnet: "gpt-sonnet-next", Haiku: "gpt-haiku-next", Fable: "gpt-fable-next"})
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
	specs := DirectRuntimeModelSpecs(Config{Opus: "gpt-opus-next", Sonnet: "gpt-sonnet-next", Haiku: "gpt-haiku-next", Fable: "gpt-fable-next"})
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
		"gpt-6.1-sol[1m]":      "gpt-6.1-sol",
		"gpt-6.1-sol[1m][1m]":  "gpt-6.1-sol",
		"gpt-6.1-sol[1M] [1m]": "gpt-6.1-sol",
		" gpt-6-luna[1m]":      "gpt-6-luna",
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

func TestDefaultFourTierRoutes(t *testing.T) {
	cfg := Default()
	for alias, want := range map[string]string{
		"haiku": "gpt-6-luna", "sonnet": "gpt-6.1-terra", "opus": "gpt-6.1-sol", "fable": "gpt-6-astra",
		"claude-fable-5-5[1m]": "gpt-6-astra", "gpt-6-astra[1m]": "gpt-6-astra",
	} {
		if got := cfg.MapModel(alias); got != want {
			t.Errorf("MapModel(%q) = %s, want %s", alias, got, want)
		}
	}
	if len(cfg.RequiredModels()) != 4 {
		t.Fatal("four distinct default models must supply live metadata")
	}
	overridden := Config{Fable: "astra-custom"}.Normalize()
	if overridden.MapModel("fable") != "astra-custom" || overridden.MapModel("gpt-6-astra") != "astra-custom" {
		t.Fatal("Fable override is not used by aliases and direct default IDs")
	}
	if alias, ok := overridden.SettingsAliasForTarget("astra-custom[1m]"); !ok || alias != "fable" {
		t.Fatal("Fable override cannot round-trip through settings")
	}
}
