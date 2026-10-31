package convert

import (
	"fmt"
	"strings"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelcatalog"
	"github.com/bassner/claudodex/internal/modelconfig"
)

type ReasoningEffort string

const (
	EffortNone    ReasoningEffort = "none"
	EffortMinimal ReasoningEffort = "minimal"
	EffortLow     ReasoningEffort = "low"
	EffortMedium  ReasoningEffort = "medium"
	EffortHigh    ReasoningEffort = "high"
	EffortXHigh   ReasoningEffort = "xhigh"
	EffortMax     ReasoningEffort = "max"
)

// MapReasoningEffortForModel derives ordinary reasoning behavior exclusively
// from the actual advertised target. It is also used for Claude UI defaults.
func MapReasoningEffortForModel(model codex.ModelInfo, outputEffort string, budgetTokens int) (ReasoningEffort, error) {
	levels := modelcatalog.OrdinaryReasoningLevels(model)
	if len(levels) == 0 {
		return "", fmt.Errorf("Codex model %q has no supported ordinary reasoning efforts", model.Slug)
	}
	requested, explicit := normalizeEffort(outputEffort)
	if !explicit && budgetTokens > 0 && !strings.EqualFold(strings.TrimSpace(outputEffort), "auto") {
		switch {
		case budgetTokens < 4000:
			requested = EffortLow
		case budgetTokens < 16000:
			requested = EffortMedium
		case budgetTokens < 32000:
			requested = EffortHigh
		default:
			requested = EffortXHigh
		}
		explicit = true
	}
	if !explicit {
		for _, level := range levels {
			if level == model.DefaultReasoningLevel {
				return ReasoningEffort(level), nil
			}
		}
		// Unknown, omitted, and workflow defaults never select ultra. Medium is
		// the conservative fallback, clamped to an actually supported level.
		requested = EffortMedium
	}
	order := map[ReasoningEffort]int{EffortNone: 0, EffortMinimal: 1, EffortLow: 2, EffortMedium: 3, EffortHigh: 4, EffortXHigh: 5, EffortMax: 6}
	selected := ReasoningEffort(levels[0])
	for _, level := range levels {
		if order[ReasoningEffort(level)] <= order[requested] {
			selected = ReasoningEffort(level)
		}
	}
	return selected, nil
}

func MapReasoningEffort(model string, outputEffort string, budgetTokens int) ReasoningEffort {
	return MapReasoningEffortWithConfig(model, outputEffort, budgetTokens, modelconfig.Default())
}

func MapReasoningEffortWithConfig(model string, outputEffort string, budgetTokens int, models modelconfig.Config) ReasoningEffort {
	models = models.Normalize()
	if effort, ok := normalizeEffort(outputEffort); ok {
		if effort == EffortMax {
			if supportsMax(model) {
				return EffortMax
			}
			effort = EffortXHigh
		}
		if effort == EffortXHigh && !supportsXHigh(model, models) {
			return EffortHigh
		}
		return effort
	}
	if budgetTokens > 0 {
		switch {
		case budgetTokens < 4000:
			return EffortLow
		case budgetTokens < 16000:
			return EffortMedium
		case budgetTokens < 32000:
			return EffortHigh
		default:
			if supportsXHigh(model, models) {
				return EffortXHigh
			}
			return EffortHigh
		}
	}
	normalizedModel := strings.ToLower(modelconfig.StripLongContext(model))
	if models.IsLowEffortDefault(model) ||
		strings.Contains(normalizedModel, "mini") ||
		strings.Contains(normalizedModel, "haiku") ||
		strings.Contains(normalizedModel, "small-fast") ||
		normalizedModel == "small" {
		return EffortLow
	}
	return EffortMedium
}

func normalizeEffort(value string) (ReasoningEffort, bool) {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "-", "")) {
	case "none":
		return EffortNone, true
	case "minimal":
		return EffortMinimal, true
	case "low":
		return EffortLow, true
	case "medium":
		return EffortMedium, true
	case "high":
		return EffortHigh, true
	case "xhigh":
		return EffortXHigh, true
	case "max":
		return EffortMax, true
	case "ultracode":
		return EffortMax, true
	case "auto":
		return "", false
	default:
		return "", false
	}
}

func supportsXHigh(model string, models modelconfig.Config) bool {
	return models.SupportsXHigh(model)
}

func supportsMax(model string) bool {
	switch strings.ToLower(modelconfig.StripLongContext(model)) {
	case modelconfig.DefaultOpusModel, modelconfig.DefaultSonnetModel, modelconfig.DefaultHaikuModel, modelconfig.DefaultFableModel, "gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna":
		return true
	default:
		return false
	}
}
