// Package modelcatalog validates Codex metadata and resolves Claude's four roles.
package modelcatalog

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
)

const MaxModels = 2000

var (
	modelIDPattern        = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,199}$`)
	automaticModelPattern = regexp.MustCompile(`^gpt-([0-9]+(?:\.[0-9]+)*)-(luna|terra|sol|astra)$`)
	numericVersionPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)*$`)
)

var ordinaryEfforts = []string{"none", "minimal", "low", "medium", "high", "xhigh", "max"}

// OrdinaryReasoningLevels returns the supported ordinary efforts in increasing
// order. Workflow levels such as ultra are deliberately excluded.
func OrdinaryReasoningLevels(model codex.ModelInfo) []string {
	seen := make(map[string]bool, len(model.SupportedReasoningLevels))
	for _, level := range model.SupportedReasoningLevels {
		seen[level.Effort] = true
	}
	var out []string
	for _, effort := range ordinaryEfforts {
		if seen[effort] {
			out = append(out, effort)
		}
	}
	return out
}

// SupportsFast requires the backend's priority service tier. A speed label by
// itself cannot authorize a service tier that the model does not advertise.
func SupportsFast(model codex.ModelInfo) bool {
	for _, tier := range model.ServiceTiers {
		if tier.ID == "priority" {
			return true
		}
	}
	return false
}

// ValidateCatalog checks the common shape without requiring hidden or unused
// models to expose usable generation capabilities.
func ValidateCatalog(models []codex.ModelInfo) error {
	if len(models) == 0 || len(models) > MaxModels {
		return fmt.Errorf("Codex model catalog must contain between 1 and %d models", MaxModels)
	}
	seen := make(map[string]bool, len(models))
	for _, model := range models {
		if !modelIDPattern.MatchString(model.Slug) {
			return fmt.Errorf("Codex model catalog contains an invalid model ID")
		}
		key := strings.ToLower(model.Slug)
		if seen[key] {
			return fmt.Errorf("Codex model catalog contains duplicate model %q", model.Slug)
		}
		seen[key] = true
		if len(model.DisplayName) > 1024 || len(model.Description) > 65536 || len(model.Visibility) > 64 || len(model.DefaultReasoningLevel) > 64 {
			return fmt.Errorf("Codex model %q has oversized metadata", model.Slug)
		}
		if model.ModelMessages != nil && model.ModelMessages.ContentFilterGuidance != nil && len(*model.ModelMessages.ContentFilterGuidance) > 65536 {
			return fmt.Errorf("Codex model %q has oversized message metadata", model.Slug)
		}
		if model.ContextWindow < 0 || model.MaxContextWindow < 0 || model.AutoCompactTokenLimit < 0 || model.EffectiveContextWindowPercent < 0 || model.EffectiveContextWindowPercent > 100 {
			return fmt.Errorf("Codex model %q has invalid context metadata", model.Slug)
		}
		if model.Priority != nil && (*model.Priority < 0 || *model.Priority > 1_000_000_000) {
			return fmt.Errorf("Codex model %q has invalid catalog priority", model.Slug)
		}
		if model.MinimalClientVersion != "" && (len(model.MinimalClientVersion) > 64 || !numericVersionPattern.MatchString(model.MinimalClientVersion)) {
			return fmt.Errorf("Codex model %q has invalid minimum client version", model.Slug)
		}
		if len(model.SupportedReasoningLevels) > 32 || len(model.ServiceTiers) > 32 || len(model.AdditionalSpeedTiers) > 32 {
			return fmt.Errorf("Codex model %q has oversized capability metadata", model.Slug)
		}
		for _, level := range model.SupportedReasoningLevels {
			if len(level.Effort) > 64 || len(level.Description) > 4096 {
				return fmt.Errorf("Codex model %q has oversized reasoning metadata", model.Slug)
			}
		}
		for _, tier := range model.ServiceTiers {
			if len(tier.ID) > 64 || len(tier.Name) > 1024 {
				return fmt.Errorf("Codex model %q has oversized service tier metadata", model.Slug)
			}
		}
		for _, tier := range model.AdditionalSpeedTiers {
			if len(tier) > 64 {
				return fmt.Errorf("Codex model %q has oversized speed metadata", model.Slug)
			}
		}
	}
	return nil
}

// ValidateTarget validates an exact explicit target. It never substitutes a
// role's default for an unavailable direct model ID.
func ValidateTarget(models []codex.ModelInfo, slug, clientVersion string) (codex.ModelInfo, error) {
	if err := ValidateCatalog(models); err != nil {
		return codex.ModelInfo{}, err
	}
	slug = modelconfig.StripLongContext(slug)
	for _, model := range models {
		if model.Slug != slug {
			continue
		}
		if err := validateTarget(model, codex.ModelsClientVersion(clientVersion)); err != nil {
			return codex.ModelInfo{}, err
		}
		return model, nil
	}
	return codex.ModelInfo{}, fmt.Errorf("Codex model %q is not available in this account's catalog; choose an advertised model or remove the explicit override", slug)
}

func validateTarget(model codex.ModelInfo, clientVersion string) error {
	if !model.SupportedInAPI || model.Visibility != "list" {
		return fmt.Errorf("Codex model %q is not visible and available for API use; choose an advertised model or remove the explicit override", model.Slug)
	}
	if model.MinimalClientVersion != "" && compareNumericVersion(model.MinimalClientVersion, clientVersion) > 0 {
		return fmt.Errorf("Codex model %q requires discovery client %s (current %s); update Claudodex or choose another model", model.Slug, model.MinimalClientVersion, clientVersion)
	}
	if model.ContextWindow <= 0 && model.MaxContextWindow <= 0 {
		return fmt.Errorf("Codex model %q has no usable context window", model.Slug)
	}
	if model.ContextWindow > 0 && model.MaxContextWindow > 0 && model.MaxContextWindow < model.ContextWindow {
		return fmt.Errorf("Codex model %q has inconsistent context windows", model.Slug)
	}
	levels := OrdinaryReasoningLevels(model)
	if len(levels) == 0 {
		return fmt.Errorf("Codex model %q has no supported ordinary reasoning efforts", model.Slug)
	}
	for _, ordinary := range ordinaryEfforts {
		if model.DefaultReasoningLevel == ordinary && !contains(levels, ordinary) {
			return fmt.Errorf("Codex model %q advertises an unsupported default reasoning effort %q", model.Slug, ordinary)
		}
	}
	return nil
}

// Resolve chooses a target independently within every role family. Explicit
// pins keep their exact advertised model, even when they equal a compiled default.
func Resolve(models []codex.ModelInfo, requested modelconfig.Config, pins modelconfig.Pins, clientVersion string) (modelconfig.Config, error) {
	if err := ValidateCatalog(models); err != nil {
		return modelconfig.Config{}, err
	}
	for _, role := range []struct {
		family modelconfig.Family
		target string
	}{{modelconfig.FamilyOpus, requested.Opus}, {modelconfig.FamilySonnet, requested.Sonnet}, {modelconfig.FamilyHaiku, requested.Haiku}, {modelconfig.FamilyFable, requested.Fable}} {
		if pins.Has(role.family) && modelconfig.StripLongContext(role.target) == "" {
			return modelconfig.Config{}, fmt.Errorf("explicit %s override is empty; choose an advertised model or remove the explicit override", role.family)
		}
	}
	requested = requested.Normalize()
	clientVersion = codex.ModelsClientVersion(clientVersion)
	resolved := modelconfig.Config{}
	for _, role := range []struct {
		family modelconfig.Family
		suffix string
		target *string
	}{
		{modelconfig.FamilyOpus, "sol", &resolved.Opus},
		{modelconfig.FamilySonnet, "terra", &resolved.Sonnet},
		{modelconfig.FamilyHaiku, "luna", &resolved.Haiku},
		{modelconfig.FamilyFable, "astra", &resolved.Fable},
	} {
		if pins.Has(role.family) {
			model, err := ValidateTarget(models, requested.Target(role.family), clientVersion)
			if err != nil {
				return modelconfig.Config{}, fmt.Errorf("explicit %s override: %w", role.family, err)
			}
			*role.target = model.Slug
			continue
		}
		var candidates []codex.ModelInfo
		for _, model := range models {
			parts := automaticModelPattern.FindStringSubmatch(model.Slug)
			if len(parts) != 3 || parts[2] != role.suffix || !model.SupportedInAPI || model.Visibility != "list" {
				continue
			}
			if model.MinimalClientVersion != "" && compareNumericVersion(model.MinimalClientVersion, clientVersion) > 0 {
				continue
			}
			candidates = append(candidates, model)
		}
		if len(candidates) == 0 {
			return modelconfig.Config{}, fmt.Errorf("no visible API-compatible %s model for the %s role; update Claudodex or set --claudodex-%s-model to an available explicit model", role.suffix, role.family, role.family)
		}
		sort.Slice(candidates, func(i, j int) bool {
			left := automaticModelPattern.FindStringSubmatch(candidates[i].Slug)[1]
			right := automaticModelPattern.FindStringSubmatch(candidates[j].Slug)[1]
			if cmp := compareNumericVersion(left, right); cmp != 0 {
				return cmp > 0
			}
			lp, rp := priority(candidates[i]), priority(candidates[j])
			if lp != rp {
				return lp < rp
			}
			return candidates[i].Slug < candidates[j].Slug
		})
		selected := candidates[0]
		if err := validateTarget(selected, clientVersion); err != nil {
			return modelconfig.Config{}, err
		}
		*role.target = selected.Slug
	}
	return resolved, nil
}

func priority(model codex.ModelInfo) int {
	if model.Priority == nil {
		return 1_000_000_001
	}
	return *model.Priority
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

// compareNumericVersion compares each numeric component without integer
// conversion, so an oversized version component cannot overflow a machine int.
func compareNumericVersion(left, right string) int {
	l, r := strings.Split(left, "."), strings.Split(right, ".")
	n := len(l)
	if len(r) > n {
		n = len(r)
	}
	for i := 0; i < n; i++ {
		lp, rp := "", ""
		if i < len(l) {
			lp = strings.TrimLeft(l[i], "0")
		}
		if i < len(r) {
			rp = strings.TrimLeft(r[i], "0")
		}
		if len(lp) != len(rp) {
			if len(lp) < len(rp) {
				return -1
			}
			return 1
		}
		if lp < rp {
			return -1
		}
		if lp > rp {
			return 1
		}
	}
	return 0
}
