package modelcatalog

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
)

func resolverModel(slug string) codex.ModelInfo {
	return codex.ModelInfo{Slug: slug, SupportedInAPI: true, Visibility: "list", ContextWindow: 123456, MaxContextWindow: 234567,
		DefaultReasoningLevel: "medium", SupportedReasoningLevels: []codex.ReasoningEffortPreset{{Effort: "low"}, {Effort: "medium"}, {Effort: "high"}, {Effort: "max"}, {Effort: "ultra"}}}
}

func resolverModels() []codex.ModelInfo {
	return []codex.ModelInfo{resolverModel("gpt-6.1-sol"), resolverModel("gpt-6.1-terra"), resolverModel("gpt-6-luna"), resolverModel("gpt-6-astra")}
}

func TestResolveFindsFutureGenerationWithinEachFamily(t *testing.T) {
	models := append(resolverModels(), resolverModel("gpt-9.9-sol"), resolverModel("gpt-9.10-sol"), resolverModel("gpt-8-terra"), resolverModel("gpt-10-luna"), resolverModel("gpt-7.0-astra"))
	got, err := Resolve(models, modelconfig.Default(), 0, "dev")
	if err != nil {
		t.Fatal(err)
	}
	want := modelconfig.Config{Opus: "gpt-9.10-sol", Sonnet: "gpt-8-terra", Haiku: "gpt-10-luna", Fable: "gpt-7.0-astra"}
	if got != want {
		t.Fatalf("Resolve = %#v, want %#v", got, want)
	}
}

func TestResolvePreservesExplicitDefaultPins(t *testing.T) {
	models := append(resolverModels(), resolverModel("gpt-7-sol"), resolverModel("gpt-7-terra"), resolverModel("gpt-7-luna"), resolverModel("gpt-7-astra"))
	got, err := Resolve(models, modelconfig.Default(), modelconfig.AllPins, "")
	if err != nil || got != modelconfig.Default() {
		t.Fatalf("explicit defaults = %#v, %v", got, err)
	}
	got, err = Resolve(models, modelconfig.Default(), modelconfig.PinOpus, "")
	if err != nil || got.Opus != modelconfig.DefaultOpusModel || got.Sonnet != "gpt-7-terra" {
		t.Fatalf("partial pins = %#v, %v", got, err)
	}
}

func TestResolveExcludesHiddenUnsupportedSuffixAndNewClient(t *testing.T) {
	hidden := resolverModel("gpt-20-sol")
	hidden.Visibility = "hide"
	unsupported := resolverModel("gpt-21-sol")
	unsupported.SupportedInAPI = false
	newClient := resolverModel("gpt-22-sol")
	newClient.MinimalClientVersion = "0.200.0"
	models := append(resolverModels(), hidden, unsupported, newClient, resolverModel("gpt-23-sol-fast"), resolverModel("gpt-24-sol-preview"))
	got, err := Resolve(models, modelconfig.Default(), 0, "0.100.0")
	if err != nil || got.Opus != modelconfig.DefaultOpusModel {
		t.Fatalf("eligible selection = %#v, %v", got, err)
	}
	got, err = Resolve(models, modelconfig.Default(), 0, "0.200.0")
	if err != nil || got.Opus != "gpt-22-sol" {
		t.Fatalf("compatible new client = %#v, %v", got, err)
	}
	requested := modelconfig.Config{Opus: "gpt-23-sol-fast"}.Normalize()
	got, err = Resolve(models, requested, modelconfig.PinOpus, "")
	if err != nil || got.Opus != requested.Opus {
		t.Fatalf("explicit suffix selection = %#v, %v", got, err)
	}
}

func TestResolvePriorityAndStableTieBreak(t *testing.T) {
	zero, one := 0, 1
	models := resolverModels()
	models[0].Priority = &one
	preferred := resolverModel("gpt-6.1.0-sol")
	preferred.Priority = &zero
	models = append(models, preferred, resolverModel("gpt-6.01-sol"))
	got, err := Resolve(models, modelconfig.Default(), 0, "")
	if err != nil || got.Opus != preferred.Slug {
		t.Fatalf("priority winner = %#v, %v", got, err)
	}
	models[0].Priority = &zero
	got, err = Resolve(models, modelconfig.Default(), 0, "")
	if err != nil || got.Opus != "gpt-6.1-sol" {
		t.Fatalf("stable slug winner = %#v, %v", got, err)
	}
}

func TestResolveMissingFamilyAndPinsAreActionable(t *testing.T) {
	models := resolverModels()
	_, err := Resolve(models[:3], modelconfig.Default(), 0, "")
	if err == nil || !strings.Contains(err.Error(), "--claudodex-fable-model") {
		t.Fatalf("missing family error = %v", err)
	}
	_, err = Resolve(models, modelconfig.Config{Opus: "gpt-missing"}, modelconfig.PinOpus, "")
	if err == nil || !strings.Contains(err.Error(), "explicit opus override") || !strings.Contains(err.Error(), "not available") {
		t.Fatalf("missing pin error = %v", err)
	}
	_, err = Resolve(models, modelconfig.Config{}, modelconfig.PinOpus, "")
	if err == nil || !strings.Contains(err.Error(), "override is empty") {
		t.Fatalf("empty pin error = %v", err)
	}
}

func TestResolveDoesNotConcealUnusableNewestModel(t *testing.T) {
	newest := resolverModel("gpt-7-sol")
	newest.SupportedReasoningLevels = nil
	_, err := Resolve(append(resolverModels(), newest), modelconfig.Default(), 0, "")
	if err == nil || !strings.Contains(err.Error(), "gpt-7-sol") {
		t.Fatalf("unusable newest model was concealed: %v", err)
	}
}

func TestValidateTargetRejectsUnusableMetadata(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*codex.ModelInfo)
	}{
		{"hidden", func(m *codex.ModelInfo) { m.Visibility = "hide" }},
		{"not API", func(m *codex.ModelInfo) { m.SupportedInAPI = false }},
		{"new client", func(m *codex.ModelInfo) { m.MinimalClientVersion = "9.0.0" }},
		{"no context", func(m *codex.ModelInfo) { m.ContextWindow = 0; m.MaxContextWindow = 0 }},
		{"inconsistent context", func(m *codex.ModelInfo) { m.MaxContextWindow = 1 }},
		{"no ordinary effort", func(m *codex.ModelInfo) {
			m.SupportedReasoningLevels = []codex.ReasoningEffortPreset{{Effort: "ultra"}}
		}},
		{"unsupported default", func(m *codex.ModelInfo) { m.DefaultReasoningLevel = "xhigh" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := resolverModel("gpt-6.1-sol")
			test.mutate(&model)
			if _, err := ValidateTarget([]codex.ModelInfo{model}, model.Slug, ""); err == nil {
				t.Fatal("unusable target was accepted")
			}
		})
	}
}

func TestValidateCatalogRejectsMalformedShape(t *testing.T) {
	for name, models := range map[string][]codex.ModelInfo{
		"empty":             {},
		"duplicate":         {resolverModel("gpt-6-sol"), resolverModel("gpt-6-sol")},
		"case duplicate":    {resolverModel("gpt-6-sol"), resolverModel("GPT-6-sol")},
		"invalid ID":        {resolverModel("gpt-bad/model")},
		"invalid version":   {{Slug: "gpt-6-sol", MinimalClientVersion: "latest"}},
		"negative context":  {{Slug: "gpt-6-sol", ContextWindow: -1}},
		"oversized display": {{Slug: "gpt-6-sol", DisplayName: strings.Repeat("a", 1025)}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateCatalog(models); err == nil {
				t.Fatal("malformed catalog accepted")
			}
		})
	}
}

func TestUnknownDefaultAndHiddenUnusedCapabilitiesAreAllowed(t *testing.T) {
	model := resolverModel("gpt-6-sol")
	model.DefaultReasoningLevel = "future-workflow"
	if _, err := ValidateTarget([]codex.ModelInfo{model}, model.Slug, ""); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCatalog([]codex.ModelInfo{{Slug: "hidden-no-capabilities", Visibility: "hide"}}); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogMetadataDecodeAndFastSupport(t *testing.T) {
	var models []codex.ModelInfo
	if err := json.Unmarshal([]byte(`[{"slug":"gpt-6-sol","priority":0,"minimal_client_version":"0.166.0","default_reasoning_level":"medium","service_tiers":[{"id":"priority","name":"Fast"}],"additional_speed_tiers":["fast"]},{"slug":"gpt-6-luna"}]`), &models); err != nil {
		t.Fatal(err)
	}
	if models[0].Priority == nil || *models[0].Priority != 0 || models[1].Priority != nil || models[0].MinimalClientVersion != "0.166.0" || !SupportsFast(models[0]) || SupportsFast(models[1]) {
		t.Fatalf("catalog capability decoding failed: %#v", models)
	}
	if SupportsFast(codex.ModelInfo{AdditionalSpeedTiers: []string{"fast"}}) {
		t.Fatal("a speed label authorized an unadvertised priority tier")
	}
}

func TestNumericVersionComparisonDoesNotOverflow(t *testing.T) {
	if compareNumericVersion("9999999999999999999999999.1", "9.999") <= 0 || compareNumericVersion("6.01.0", "6.1") != 0 {
		t.Fatal("numeric component comparison failed")
	}
}
