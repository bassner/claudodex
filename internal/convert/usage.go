package convert

import (
	"math"
	"strconv"
	"strings"
	"time"
)

func CodexUsageToClaude(raw map[string]any) map[string]any {
	out := map[string]any{
		"five_hour":            nil,
		"seven_day":            nil,
		"seven_day_oauth_apps": nil,
		"seven_day_fable":      nil,
		"seven_day_haiku":      nil,
		"seven_day_opus":       nil,
		"seven_day_sonnet":     nil,
		"service_tier":         "standard",
		"extra_usage":          extraUsage(raw),
	}
	if rateLimit, _ := raw["rate_limit"].(map[string]any); rateLimit != nil {
		for _, key := range []string{"primary_window", "secondary_window"} {
			window, _ := rateLimit[key].(map[string]any)
			switch {
			case approxWindow(window, 5*time.Hour):
				out["five_hour"] = claudeRateLimit(window)
			case approxWindow(window, 7*24*time.Hour):
				out["seven_day"] = claudeRateLimit(window)
			}
		}
	}
	if limits, _ := raw["additional_rate_limits"].([]any); len(limits) > 0 {
		for _, item := range limits {
			limit, _ := item.(map[string]any)
			if limit == nil {
				continue
			}
			family := modelFamilyFromLimit(limit)
			if family == "" {
				continue
			}
			rateLimit, _ := limit["rate_limit"].(map[string]any)
			window := sevenDayWindow(rateLimit)
			if window == nil {
				continue
			}
			out["seven_day_"+family] = claudeRateLimit(window)
		}
	}
	return out
}

func sevenDayWindow(rateLimit map[string]any) map[string]any {
	for _, key := range []string{"primary_window", "secondary_window"} {
		window, _ := rateLimit[key].(map[string]any)
		if approxWindow(window, 7*24*time.Hour) {
			return window
		}
	}
	return nil
}

func approxWindow(window map[string]any, target time.Duration) bool {
	if window == nil {
		return false
	}
	seconds := number(window["limit_window_seconds"])
	if seconds <= 0 {
		return false
	}
	targetSeconds := target.Seconds()
	return math.Abs(seconds-targetSeconds) <= targetSeconds*0.05
}

func claudeRateLimit(window map[string]any) map[string]any {
	var resetsAt any
	if reset := int64(number(window["reset_at"])); reset > 0 {
		resetsAt = time.Unix(reset, 0).UTC().Format(time.RFC3339)
	}
	return map[string]any{
		"utilization": numberOrNil(window["used_percent"]),
		"resets_at":   resetsAt,
	}
}

func extraUsage(raw map[string]any) any {
	credits, hasCreditsObj := raw["credits"].(map[string]any)
	spend, hasSpendObj := raw["spend_control"].(map[string]any)
	hasCredits, _ := credits["has_credits"].(bool)
	if !hasCredits && !hasSpendObj && !hasCreditsObj {
		return nil
	}
	var monthlyLimit any
	if hasSpendObj {
		monthlyLimit = numberOrNil(spend["individual_limit"])
	}
	return map[string]any{
		"is_enabled":    hasCredits || hasSpendObj,
		"monthly_limit": monthlyLimit,
		"used_credits":  nil,
		"utilization":   nil,
	}
}

func modelFamilyFromLimit(limit map[string]any) string {
	value := strings.ToLower(stringFromAny(limit["metered_feature"]) + " " + stringFromAny(limit["limit_name"]))
	switch {
	case strings.Contains(value, "fable"),
		strings.Contains(value, "mythos"):
		return "fable"
	case strings.Contains(value, "opus"):
		return "opus"
	case strings.Contains(value, "sonnet"):
		return "sonnet"
	case strings.Contains(value, "haiku"),
		strings.Contains(value, "mini"):
		return "haiku"
	default:
		return ""
	}
}

func numberOrNil(value any) any {
	n := number(value)
	if n == 0 && value == nil {
		return nil
	}
	return n
}

func number(value any) float64 {
	switch v := value.(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0
		}
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case jsonNumber:
		f, _ := strconv.ParseFloat(string(v), 64)
		return f
	}
	return 0
}

type jsonNumber string

func stringFromAny(value any) string {
	if s, _ := value.(string); s != "" {
		return s
	}
	return ""
}
