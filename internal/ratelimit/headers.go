package ratelimit

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Snapshot struct {
	FiveHour *Window
	SevenDay *Window
}

type Window struct {
	UsedPercent float64
	ResetAt     int64
}

func FromCodexHeaders(headers http.Header) *Snapshot {
	primary := parseCodexWindow(headers, "x-codex-primary")
	secondary := parseCodexWindow(headers, "x-codex-secondary")
	if primary == nil && secondary == nil {
		return nil
	}
	snapshot := &Snapshot{}
	for _, window := range []*parsedWindow{primary, secondary} {
		if window == nil {
			continue
		}
		switch {
		case approxMinutes(window.WindowMinutes, 5*time.Hour):
			snapshot.FiveHour = &Window{UsedPercent: window.UsedPercent, ResetAt: window.ResetAt}
		case approxMinutes(window.WindowMinutes, 7*24*time.Hour):
			snapshot.SevenDay = &Window{UsedPercent: window.UsedPercent, ResetAt: window.ResetAt}
		}
	}
	if snapshot.FiveHour == nil && snapshot.SevenDay == nil {
		return nil
	}
	return snapshot
}

func ApplyAnthropicHeaders(headers http.Header, snapshot *Snapshot, forceRejected bool) {
	if snapshot == nil && !forceRejected {
		return
	}
	status := "allowed"
	if forceRejected || windowRejected(snapshot.FiveHour) || windowRejected(snapshot.SevenDay) {
		status = "rejected"
	} else if windowWarning(snapshot.FiveHour) || windowWarning(snapshot.SevenDay) {
		status = "allowed_warning"
	}
	headers.Set("anthropic-ratelimit-unified-status", status)
	headers.Set("anthropic-ratelimit-unified-fallback", "available")
	if snapshot == nil {
		return
	}
	if snapshot.FiveHour != nil {
		headers.Set("anthropic-ratelimit-unified-5h-utilization", fractionString(snapshot.FiveHour.UsedPercent))
		if snapshot.FiveHour.ResetAt > 0 {
			headers.Set("anthropic-ratelimit-unified-5h-reset", strconv.FormatInt(snapshot.FiveHour.ResetAt, 10))
		}
	}
	if snapshot.SevenDay != nil {
		headers.Set("anthropic-ratelimit-unified-7d-utilization", fractionString(snapshot.SevenDay.UsedPercent))
		if snapshot.SevenDay.ResetAt > 0 {
			headers.Set("anthropic-ratelimit-unified-7d-reset", strconv.FormatInt(snapshot.SevenDay.ResetAt, 10))
		}
	}
	claim, reset := representativeClaim(snapshot)
	if claim != "" {
		headers.Set("anthropic-ratelimit-unified-representative-claim", claim)
	}
	if reset > 0 {
		headers.Set("anthropic-ratelimit-unified-reset", strconv.FormatInt(reset, 10))
	}
}

func SnapshotFromRetryAfter(headers http.Header, now time.Time) *Snapshot {
	value := strings.TrimSpace(headers.Get("retry-after"))
	if value == "" {
		return nil
	}
	var resetAt int64
	if seconds, err := strconv.Atoi(value); err == nil {
		resetAt = now.Add(time.Duration(seconds) * time.Second).Unix()
	} else if parsed, err := http.ParseTime(value); err == nil {
		resetAt = parsed.Unix()
	}
	if resetAt <= 0 {
		return nil
	}
	return &Snapshot{FiveHour: &Window{UsedPercent: 100, ResetAt: resetAt}}
}

type parsedWindow struct {
	UsedPercent   float64
	WindowMinutes int
	ResetAt       int64
}

func parseCodexWindow(headers http.Header, prefix string) *parsedWindow {
	usedRaw := headers.Get(prefix + "-used-percent")
	if strings.TrimSpace(usedRaw) == "" {
		return nil
	}
	usedPercent, err := strconv.ParseFloat(usedRaw, 64)
	if err != nil || math.IsNaN(usedPercent) || math.IsInf(usedPercent, 0) {
		return nil
	}
	windowMinutes, _ := strconv.Atoi(strings.TrimSpace(headers.Get(prefix + "-window-minutes")))
	resetAt, _ := strconv.ParseInt(strings.TrimSpace(headers.Get(prefix+"-reset-at")), 10, 64)
	return &parsedWindow{UsedPercent: usedPercent, WindowMinutes: windowMinutes, ResetAt: resetAt}
}

func approxMinutes(minutes int, target time.Duration) bool {
	if minutes <= 0 {
		return false
	}
	targetMinutes := target.Minutes()
	return math.Abs(float64(minutes)-targetMinutes) <= targetMinutes*0.05
}

func windowRejected(window *Window) bool {
	return window != nil && window.UsedPercent >= 100
}

func windowWarning(window *Window) bool {
	return window != nil && window.UsedPercent >= 90
}

func fractionString(percent float64) string {
	fraction := percent / 100
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	return strconv.FormatFloat(fraction, 'f', -1, 64)
}

func representativeClaim(snapshot *Snapshot) (string, int64) {
	if snapshot == nil {
		return "", 0
	}
	type candidate struct {
		claim  string
		window *Window
	}
	candidates := []candidate{{claim: "five_hour", window: snapshot.FiveHour}, {claim: "seven_day", window: snapshot.SevenDay}}
	var best candidate
	for _, candidate := range candidates {
		if candidate.window == nil {
			continue
		}
		if best.window == nil || candidate.window.UsedPercent > best.window.UsedPercent {
			best = candidate
		}
	}
	if best.window == nil {
		return "", 0
	}
	return best.claim, best.window.ResetAt
}

func DebugString(snapshot *Snapshot) string {
	if snapshot == nil {
		return "none"
	}
	return fmt.Sprintf("5h=%v 7d=%v", snapshot.FiveHour, snapshot.SevenDay)
}
