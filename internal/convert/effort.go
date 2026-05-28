package convert

import (
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

type ReasoningEffort string

const (
	EffortLow    ReasoningEffort = "low"
	EffortMedium ReasoningEffort = "medium"
	EffortHigh   ReasoningEffort = "high"
	EffortXHigh  ReasoningEffort = "xhigh"
)

func MapReasoningEffort(model string, outputEffort string, budgetTokens int) ReasoningEffort {
	return MapReasoningEffortWithConfig(model, outputEffort, budgetTokens, modelconfig.Default())
}

func MapReasoningEffortWithConfig(model string, outputEffort string, budgetTokens int, models modelconfig.Config) ReasoningEffort {
	models = models.Normalize()
	if effort, ok := normalizeEffort(outputEffort); ok {
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
	case "low":
		return EffortLow, true
	case "medium":
		return EffortMedium, true
	case "high":
		return EffortHigh, true
	case "xhigh", "max", "ultracode":
		return EffortXHigh, true
	case "auto":
		return "", false
	default:
		return "", false
	}
}

func supportsXHigh(model string, models modelconfig.Config) bool {
	return models.SupportsXHigh(model)
}
