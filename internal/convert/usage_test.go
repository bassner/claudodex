package convert

import "testing"

func TestCodexUsageToClaudeMapsWindowsAndExtraUsage(t *testing.T) {
	raw := map[string]any{
		"account_id": "must-not-pass-through",
		"email":      "must-not-pass-through",
		"rate_limit": map[string]any{
			"primary_window": map[string]any{
				"used_percent":         float64(12),
				"limit_window_seconds": float64(18000),
				"reset_at":             float64(1770000000),
			},
			"secondary_window": map[string]any{
				"used_percent":         float64(34),
				"limit_window_seconds": float64(604800),
				"reset_at":             float64(1770500000),
			},
		},
		"additional_rate_limits": []any{
			map[string]any{
				"limit_name":      "codex_fable_weekly",
				"metered_feature": "codex_fable",
				"rate_limit": map[string]any{
					"primary_window": map[string]any{
						"used_percent":         float64(40),
						"limit_window_seconds": float64(604800),
						"reset_at":             float64(1770500000),
					},
				},
			},
			map[string]any{
				"limit_name":      "codex_sonnet_weekly",
				"metered_feature": "codex_sonnet",
				"rate_limit": map[string]any{
					"primary_window": map[string]any{
						"used_percent":         float64(50),
						"limit_window_seconds": float64(604800),
						"reset_at":             float64(1770500000),
					},
				},
			},
			map[string]any{
				"limit_name":      "codex_mini_weekly",
				"metered_feature": "codex_haiku",
				"rate_limit": map[string]any{
					"primary_window": map[string]any{
						"used_percent":         float64(25),
						"limit_window_seconds": float64(604800),
						"reset_at":             float64(1770500000),
					},
				},
			},
		},
		"credits": map[string]any{
			"has_credits": true,
			"balance":     "12.34",
		},
		"spend_control": map[string]any{
			"individual_limit": float64(20),
		},
	}
	got := CodexUsageToClaude(raw)
	five := got["five_hour"].(map[string]any)
	if five["utilization"] != float64(12) || five["resets_at"] != "2026-02-02T02:40:00Z" {
		t.Fatalf("five_hour = %#v", five)
	}
	seven := got["seven_day"].(map[string]any)
	if seven["utilization"] != float64(34) {
		t.Fatalf("seven_day = %#v", seven)
	}
	sonnet := got["seven_day_sonnet"].(map[string]any)
	if sonnet["utilization"] != float64(50) {
		t.Fatalf("seven_day_sonnet = %#v", sonnet)
	}
	fable := got["seven_day_fable"].(map[string]any)
	if fable["utilization"] != float64(40) {
		t.Fatalf("seven_day_fable = %#v", fable)
	}
	haiku := got["seven_day_haiku"].(map[string]any)
	if haiku["utilization"] != float64(25) {
		t.Fatalf("seven_day_haiku = %#v", haiku)
	}
	extra := got["extra_usage"].(map[string]any)
	if extra["is_enabled"] != true || extra["monthly_limit"] != float64(20) || extra["used_credits"] != nil {
		t.Fatalf("extra_usage = %#v", extra)
	}
	if _, ok := got["account_id"]; ok {
		t.Fatalf("personal account field passed through: %#v", got)
	}
}
