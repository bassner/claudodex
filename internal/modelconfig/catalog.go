package modelconfig

import (
	"fmt"
	"log"
	"strings"
)

const (
	DefaultOpusModel   = "gpt-5.5"
	DefaultSonnetModel = "gpt-5.4"
	DefaultHaikuModel  = "gpt-5.4-mini"

	LongContextSuffix         = "[1m]"
	DefaultClaudeRequestModel = "claude-opus-4-8"
)

type Family string

const (
	FamilyOpus   Family = "opus"
	FamilySonnet Family = "sonnet"
	FamilyHaiku  Family = "haiku"
	FamilyFable  Family = "fable"
)

type Config struct {
	Opus   string
	Sonnet string
	Haiku  string
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
	c.Opus = cleanTargetModel(c.Opus)
	c.Sonnet = cleanTargetModel(c.Sonnet)
	c.Haiku = cleanTargetModel(c.Haiku)
	return c
}

func (c Config) RequiredModels() []string {
	c = c.Normalize()
	seen := map[string]struct{}{}
	out := make([]string, 0, 3)
	for _, model := range []string{c.Opus, c.Sonnet, c.Haiku} {
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
	case FamilyFable:
		return c.Opus
	case FamilySonnet:
		return c.Sonnet
	case FamilyHaiku:
		return c.Haiku
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
	switch {
	case normalized == normalizeModelName(c.Opus):
		return c.Opus
	case normalized == normalizeModelName(c.Sonnet):
		return c.Sonnet
	case normalized == normalizeModelName(c.Haiku):
		return c.Haiku
	case normalized == DefaultOpusModel:
		return c.Opus
	case normalized == DefaultSonnetModel:
		return c.Sonnet
	case normalized == DefaultHaikuModel:
		return c.Haiku
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
	default:
		return "", false
	}
}

func (c Config) SupportsXHigh(model string) bool {
	c = c.Normalize()
	normalized := normalizeModelName(model)
	if normalized == normalizeModelName(c.Opus) ||
		normalized == normalizeModelName(c.Sonnet) ||
		normalized == normalizeModelName(c.Haiku) {
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
		{ID: "claude-fable-5", DisplayName: fmt.Sprintf("Fable 5 (%s)", c.Opus), Family: FamilyFable},
		{ID: "claude-opus-4-8", DisplayName: fmt.Sprintf("Opus 4.8 (%s)", c.Opus), Family: FamilyOpus},
		{ID: "claude-opus-4-6", DisplayName: fmt.Sprintf("Opus (%s)", c.Opus), Family: FamilyOpus},
		{ID: "claude-opus-4-7", DisplayName: fmt.Sprintf("Opus 4.7 (%s)", c.Opus), Family: FamilyOpus},
		{ID: "claude-sonnet-4-6", DisplayName: fmt.Sprintf("Sonnet (%s)", c.Sonnet), Family: FamilySonnet},
		{ID: "claude-haiku-4-5", DisplayName: fmt.Sprintf("Haiku (%s)", c.Haiku), Family: FamilyHaiku},
	}
}

func FamilyAliasSpecs() []ClaudeModelSpec {
	return []ClaudeModelSpec{
		{ID: string(FamilyFable), DisplayName: "Fable", Family: FamilyFable},
		{ID: string(FamilyOpus), DisplayName: "Opus", Family: FamilyOpus},
		{ID: string(FamilySonnet), DisplayName: "Sonnet", Family: FamilySonnet},
		{ID: string(FamilyHaiku), DisplayName: "Haiku", Family: FamilyHaiku},
	}
}

func DirectModelSpecs(c Config) []ClaudeModelSpec {
	c = c.Normalize()
	specs := []ClaudeModelSpec{
		{ID: c.Opus, DisplayName: c.Opus, Family: FamilyOpus},
		{ID: c.Sonnet, DisplayName: c.Sonnet, Family: FamilySonnet},
		{ID: c.Haiku, DisplayName: c.Haiku, Family: FamilyHaiku},
	}
	return dedupeSpecs(specs)
}

func DirectRuntimeModelSpecs(c Config) []ClaudeModelSpec {
	c = c.Normalize()
	specs := []ClaudeModelSpec{
		{ID: WithLongContext(c.Opus), DisplayName: c.Opus, Family: FamilyOpus},
		{ID: WithLongContext(c.Sonnet), DisplayName: c.Sonnet, Family: FamilySonnet},
		{ID: WithLongContext(c.Haiku), DisplayName: c.Haiku, Family: FamilyHaiku},
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
	case strings.Contains(normalized, "fable"),
		strings.Contains(normalized, "mythos"):
		return FamilyFable
	case strings.Contains(normalized, "haiku"),
		strings.Contains(normalized, "small-fast"),
		normalized == "small":
		return FamilyHaiku
	case strings.Contains(normalized, "opus"),
		normalized == "best",
		normalized == "opusplan":
		return FamilyOpus
	case strings.Contains(normalized, "sonnet"):
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
