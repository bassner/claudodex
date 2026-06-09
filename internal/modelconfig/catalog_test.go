package modelconfig

import "testing"

func TestConfigMapModelUsesOverrides(t *testing.T) {
	cfg := Config{Opus: "gpt-opus-next", Sonnet: "gpt-sonnet-next", Haiku: "gpt-haiku-next"}
	tests := map[string]string{
		"opus":              "gpt-opus-next",
		"fable":             "gpt-opus-next",
		"fable[1m]":         "gpt-opus-next",
		"claude-fable-5":    "gpt-opus-next",
		"mythos-5":          "gpt-opus-next",
		"gpt-5.5":           "gpt-opus-next",
		"claude-sonnet-4-6": "gpt-sonnet-next",
		"gpt-5.4[1m]":       "gpt-sonnet-next",
		"small-fast":        "gpt-haiku-next",
		"gpt-haiku-next":    "gpt-haiku-next",
	}
	for input, want := range tests {
		if got := cfg.MapModel(input); got != want {
			t.Fatalf("MapModel(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFamilyForModelPreservesFableAlias(t *testing.T) {
	family, ok := FamilyForModel("claude-fable-5[1m]")
	if !ok || family != FamilyFable {
		t.Fatalf("FamilyForModel(claude-fable-5[1m]) = %q, %v; want fable, true", family, ok)
	}
	if got := (Config{Opus: "gpt-opus-next"}).Target(family); got != "gpt-opus-next" {
		t.Fatalf("Target(fable) = %q, want gpt-opus-next", got)
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
		"gpt-5.5[1m]":       "gpt-5.5",
		"gpt-5.5[1m][1m]":   "gpt-5.5",
		"gpt-5.5[1M] [1m]":  "gpt-5.5",
		" gpt-5.4-mini[1m]": "gpt-5.4-mini",
	}
	for input, want := range tests {
		if got := StripLongContext(input); got != want {
			t.Fatalf("StripLongContext(%q) = %q, want %q", input, got, want)
		}
	}
}
