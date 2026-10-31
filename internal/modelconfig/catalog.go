package modelconfig

import (
	"fmt"
	"log"
	"strings"
)

const (
	DefaultOpusModel   = "gpt-6.1-sol"
	DefaultSonnetModel = "gpt-6.1-terra"
	DefaultHaikuModel  = "gpt-6-luna"
	DefaultFableModel  = "gpt-6-astra"

	LongContextSuffix         = "[1m]"
	DefaultClaudeRequestModel = "claude-opus-5"
)

type Family string

const (
	FamilyOpus   Family = "opus"
	FamilySonnet Family = "sonnet"
	FamilyHaiku  Family = "haiku"
	FamilyFable  Family = "fable"
)

// Pins records explicit role choices independently of Config. Config remains a
// comparable value because verified Claude patches use it as their identity.
type Pins uint8

const (
	PinOpus Pins = 1 << iota
	PinSonnet
	PinHaiku
	PinFable
	AllPins = PinOpus | PinSonnet | PinHaiku | PinFable
)

func (p Pins) Has(family Family) bool {
	return p&pinForFamily(family) != 0
}

func (p Pins) With(family Family) Pins {
	return p | pinForFamily(family)
}

func pinForFamily(family Family) Pins {
	switch family {
	case FamilyOpus:
		return PinOpus
	case FamilySonnet:
		return PinSonnet
	case FamilyHaiku:
		return PinHaiku
	case FamilyFable:
		return PinFable
	default:
		return 0
	}
}

type Config struct {
	Opus   string
	Sonnet string
	Haiku  string
	Fable  string
}

type ClaudeModelSpec struct {
	ID          string
	DisplayName string
	Family      Family
}

func Default() Config {
	return Config{
		Opus:   DefaultOpusModel,
		Sonnet: DefaultSonnetModel,
		Haiku:  DefaultHaikuModel,
		Fable:  DefaultFableModel,
	}
}

func (c Config) Normalize() Config {
	if strings.TrimSpace(c.Opus) == "" {
		c.Opus = DefaultOpusModel
	}
	if strings.TrimSpace(c.Sonnet) == "" {
		c.Sonnet = DefaultSonnetModel
	}
	if strings.TrimSpace(c.Haiku) == "" {
		c.Haiku = DefaultHaikuModel
	}
	if strings.TrimSpace(c.Fable) == "" {
		c.Fable = DefaultFableModel
	}
	c.Opus = cleanTargetModel(c.Opus)
	c.Sonnet = cleanTargetModel(c.Sonnet)
	c.Haiku = cleanTargetModel(c.Haiku)
	c.Fable = cleanTargetModel(c.Fable)
	return c
}

func (c Config) RequiredModels() []string {
	c = c.Normalize()
	seen := map[string]struct{}{}
	out := make([]string, 0, 4)
	for _, model := range []string{c.Opus, c.Sonnet, c.Haiku, c.Fable} {
		key := strings.ToLower(model)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, model)
	}
	return out
}

func (c Config) Target(family Family) string {
	c = c.Normalize()
	switch family {
	case FamilyOpus:
		return c.Opus
	case FamilySonnet:
		return c.Sonnet
	case FamilyHaiku:
		return c.Haiku
	case FamilyFable:
		return c.Fable
	default:
		return c.Opus
	}
}

func (c Config) MapModel(model string) string {
	c = c.Normalize()
	if strings.TrimSpace(model) == "" {
		model = DefaultClaudeRequestModel
	}
	normalized := normalizeModelName(model)
	// A direct GPT ID is an exact choice, including an old compiled default or
	// an as-yet unknown successor. Catalog validation decides availability.
	if strings.HasPrefix(normalized, "gpt-") {
		return cleanTargetModel(model)
	}
	switch {
	case normalized == normalizeModelName(c.Opus):
		return c.Opus
	case normalized == normalizeModelName(c.Sonnet):
		return c.Sonnet
	case normalized == normalizeModelName(c.Haiku):
		return c.Haiku
	case normalized == normalizeModelName(c.Fable):
		return c.Fable
	}
	if family, ok := FamilyForModel(model); ok {
		return c.Target(family)
	}
	log.Printf("claudodex: unknown Claude model %q; falling back to %s", model, c.Opus)
	return c.Opus
}

func (c Config) RuntimeModel(model string) string {
	return c.MapModel(model)
}

func (c Config) SettingsAliasForTarget(model string) (string, bool) {
	c = c.Normalize()
	normalized := normalizeModelName(model)
	switch normalized {
	case normalizeModelName(c.Opus):
		return string(FamilyOpus), true
	case normalizeModelName(c.Sonnet):
		return string(FamilySonnet), true
	case normalizeModelName(c.Haiku):
		return string(FamilyHaiku), true
	case normalizeModelName(c.Fable):
		return string(FamilyFable), true
	default:
		return "", false
	}
}

func (c Config) SupportsXHigh(model string) bool {
	c = c.Normalize()
	normalized := normalizeModelName(model)
	if normalized == normalizeModelName(c.Opus) ||
		normalized == normalizeModelName(c.Sonnet) ||
		normalized == normalizeModelName(c.Haiku) ||
		normalized == normalizeModelName(c.Fable) {
		return true
	}
	switch normalized {
	case "gpt-5.5", "gpt-5.4", "gpt-5.4-mini":
		return true
	}
	_, ok := FamilyForModel(model)
	return ok
}

func (c Config) IsLowEffortDefault(model string) bool {
	c = c.Normalize()
	normalized := normalizeModelName(model)
	return normalized == normalizeModelName(c.Haiku) || modelFamily(normalized) == FamilyHaiku
}

func ClaudeAliasSpecs(c Config) []ClaudeModelSpec {
	c = c.Normalize()
	return []ClaudeModelSpec{
		{ID: "claude-opus-5", DisplayName: fmt.Sprintf("Opus 5 (%s)", c.Opus), Family: FamilyOpus},
		{ID: "claude-opus-4-8", DisplayName: fmt.Sprintf("Opus 4.8 (%s)", c.Opus), Family: FamilyOpus},
		{ID: "claude-opus-4-6", DisplayName: fmt.Sprintf("Opus (%s)", c.Opus), Family: FamilyOpus},
		{ID: "claude-opus-4-7", DisplayName: fmt.Sprintf("Opus 4.7 (%s)", c.Opus), Family: FamilyOpus},
		{ID: "claude-sonnet-4-6", DisplayName: fmt.Sprintf("Sonnet (%s)", c.Sonnet), Family: FamilySonnet},
		{ID: "claude-haiku-4-5", DisplayName: fmt.Sprintf("Haiku (%s)", c.Haiku), Family: FamilyHaiku},
		{ID: "claude-fable-5", DisplayName: fmt.Sprintf("Fable (%s)", c.Fable), Family: FamilyFable},
		{ID: "claude-fable-5-5", DisplayName: fmt.Sprintf("Fable 5.5 (%s)", c.Fable), Family: FamilyFable},
	}
}

func FamilyAliasSpecs() []ClaudeModelSpec {
	return []ClaudeModelSpec{
		{ID: string(FamilyOpus), DisplayName: "Opus", Family: FamilyOpus},
		{ID: string(FamilySonnet), DisplayName: "Sonnet", Family: FamilySonnet},
		{ID: string(FamilyHaiku), DisplayName: "Haiku", Family: FamilyHaiku},
		{ID: string(FamilyFable), DisplayName: "Fable", Family: FamilyFable},
	}
}

func DirectModelSpecs(c Config) []ClaudeModelSpec {
	c = c.Normalize()
	specs := []ClaudeModelSpec{
		{ID: c.Opus, DisplayName: c.Opus, Family: FamilyOpus},
		{ID: c.Sonnet, DisplayName: c.Sonnet, Family: FamilySonnet},
		{ID: c.Haiku, DisplayName: c.Haiku, Family: FamilyHaiku},
		{ID: c.Fable, DisplayName: c.Fable, Family: FamilyFable},
	}
	return dedupeSpecs(specs)
}

func DirectRuntimeModelSpecs(c Config) []ClaudeModelSpec {
	c = c.Normalize()
	specs := []ClaudeModelSpec{
		{ID: WithLongContext(c.Opus), DisplayName: c.Opus, Family: FamilyOpus},
		{ID: WithLongContext(c.Sonnet), DisplayName: c.Sonnet, Family: FamilySonnet},
		{ID: WithLongContext(c.Haiku), DisplayName: c.Haiku, Family: FamilyHaiku},
		{ID: WithLongContext(c.Fable), DisplayName: c.Fable, Family: FamilyFable},
	}
	return dedupeSpecs(specs)
}

func FamilyForModel(model string) (Family, bool) {
	family := modelFamily(normalizeModelName(model))
	return family, family != ""
}

func WithLongContext(model string) string {
	model = cleanTargetModel(model)
	if strings.HasSuffix(strings.ToLower(model), LongContextSuffix) {
		return model
	}
	return model + LongContextSuffix
}

func StripLongContext(model string) string {
	model = strings.TrimSpace(model)
	for strings.HasSuffix(strings.ToLower(model), LongContextSuffix) {
		model = strings.TrimSpace(model[:len(model)-len(LongContextSuffix)])
	}
	return model
}

func cleanTargetModel(model string) string {
	return StripLongContext(model)
}

func normalizeModelName(model string) string {
	return strings.ToLower(StripLongContext(model))
}

func modelFamily(normalized string) Family {
	switch {
	case strings.Contains(normalized, "fable"), strings.Contains(normalized, "mythos"), normalized == DefaultFableModel:
		return FamilyFable
	case strings.Contains(normalized, "haiku"),
		normalized == DefaultHaikuModel, normalized == "gpt-5.6-luna",
		strings.Contains(normalized, "small-fast"),
		normalized == "small":
		return FamilyHaiku
	case strings.Contains(normalized, "opus"),
		normalized == DefaultOpusModel, normalized == "gpt-5.6-sol",
		normalized == "best",
		normalized == "opusplan":
		return FamilyOpus
	case strings.Contains(normalized, "sonnet"), normalized == DefaultSonnetModel, normalized == "gpt-5.6-terra":
		return FamilySonnet
	default:
		return ""
	}
}

func dedupeSpecs(specs []ClaudeModelSpec) []ClaudeModelSpec {
	seen := map[string]struct{}{}
	out := make([]ClaudeModelSpec, 0, len(specs))
	for _, spec := range specs {
		key := strings.ToLower(spec.ID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, spec)
	}
	return out
}
