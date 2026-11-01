package modelcatalog

import (
	"strings"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
)

// AutoCompactWindows supplies Claude's native server-window cache. The boolean
// is false if a selectable route cannot be represented by its native schema.
func AutoCompactWindows(models []codex.ModelInfo, configured modelconfig.Config) (map[string]int64, bool) {
	configured = configured.Normalize()
	windows := make(map[string]int64)
	complete := true
	put := func(id string, value int64) {
		if value < 100000 {
			complete = false
			return
		}
		if value > 1000000 {
			value = 1000000
		}
		id = strings.ToLower(id)
		if old, ok := windows[id]; !ok || value < old {
			windows[id] = value
		}
	}
	required := make(map[string]bool)
	for _, id := range configured.RequiredModels() {
		required[strings.ToLower(id)] = true
	}
	for _, model := range models {
		if !required[strings.ToLower(model.Slug)] && (!model.SupportedInAPI || model.Visibility != "list" || len(OrdinaryReasoningLevels(model)) == 0) {
			continue
		}
		ordinary := model.ContextWindow
		if ordinary <= 0 {
			ordinary = model.MaxContextWindow
		}
		long := model.MaxContextWindow
		if long <= 0 {
			long = ordinary
		}
		percent := model.EffectiveContextWindowPercent
		if percent == 0 {
			percent = 95
		}
		limit := func(window int64) int64 {
			if percent <= 0 || percent > 100 || window <= 0 || model.AutoCompactTokenLimit < 0 {
				return 0
			}
			value := (window/100)*percent + (window%100)*percent/100
			if model.AutoCompactTokenLimit > 0 && model.AutoCompactTokenLimit < value {
				value = model.AutoCompactTokenLimit
			}
			return value
		}
		put(model.Slug, limit(ordinary))
		put(modelconfig.WithLongContext(model.Slug), limit(long))
	}
	aliases := append(modelconfig.ClaudeAliasSpecs(configured), modelconfig.FamilyAliasSpecs()...)
	for _, alias := range aliases {
		target := strings.ToLower(configured.Target(alias.Family))
		value, ok := windows[modelconfig.StripLongContext(target)]
		if !ok {
			complete = false
			continue
		}
		// Native canonicalization drops [1m] on Claude-named aliases. Keep
		// their shared key conservative; exact GPT routes remain distinct.
		put(alias.ID, value)
		put(modelconfig.WithLongContext(alias.ID), value)
	}
	for id := range required {
		if _, ok := windows[id]; !ok {
			complete = false
		}
	}
	return windows, complete
}
