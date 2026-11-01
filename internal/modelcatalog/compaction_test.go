package modelcatalog

import (
	"testing"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
)

func TestAutoCompactWindowsKeepRoutesIndependent(t *testing.T) {
	cfg := modelconfig.Default()
	var models []codex.ModelInfo
	for _, id := range cfg.RequiredModels() {
		models = append(models, codex.ModelInfo{Slug: id, ContextWindow: 300000, MaxContextWindow: 872000})
	}
	for index := range models {
		if models[index].Slug == cfg.Opus {
			models[index].ContextWindow = 500000
		}
	}
	windows, complete := AutoCompactWindows(models, cfg)
	if !complete {
		t.Fatal("complete catalog did not enable native windows")
	}
	for id, want := range map[string]int64{
		cfg.Opus: 475000, modelconfig.WithLongContext(cfg.Opus): 828400,
		cfg.Haiku: 285000, "opus": 475000, "claude-opus-4-6": 475000, "claude-opus-4-6[1m]": 475000,
	} {
		if windows[id] != want {
			t.Fatalf("%s window = %d, want %d", id, windows[id], want)
		}
	}
	models[0].AutoCompactTokenLimit = 200000
	windows, _ = AutoCompactWindows(models, cfg)
	if windows[models[0].Slug] != 200000 || windows[modelconfig.WithLongContext(models[0].Slug)] != 200000 {
		t.Fatal("explicit backend compaction cap did not constrain both routes")
	}
}

func TestAutoCompactWindowsRespectNativeSchemaBounds(t *testing.T) {
	cfg := modelconfig.Default()
	var models []codex.ModelInfo
	for _, id := range cfg.RequiredModels() {
		models = append(models, codex.ModelInfo{Slug: id, ContextWindow: 2000000, MaxContextWindow: 3000000})
	}
	windows, complete := AutoCompactWindows(models, cfg)
	if !complete || windows[cfg.Opus] != 1000000 {
		t.Fatal("large live windows did not use the safe native maximum")
	}
	models[0].AutoCompactTokenLimit = 60000
	windows, complete = AutoCompactWindows(models, cfg)
	if complete {
		t.Fatal("unrepresentable small window removed the conservative fallback")
	}
	for _, value := range windows {
		if value < 100000 || value > 1000000 {
			t.Fatal("invalid window would reject the native bootstrap payload")
		}
	}
}
