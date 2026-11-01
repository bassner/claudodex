package proxy

import (
	"reflect"
	"testing"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelcatalog"
	"github.com/bassner/claudodex/internal/modelconfig"
)

func TestBootstrapPublishesTheSameNativeCompactionWindows(t *testing.T) {
	cfg := modelconfig.Default()
	var models []codex.ModelInfo
	for _, id := range cfg.RequiredModels() {
		models = append(models, codex.ModelInfo{Slug: id, ContextWindow: 500000, MaxContextWindow: 872000})
	}
	windows, _ := modelcatalog.AutoCompactWindows(models, cfg)
	if got := bootstrapResponse(cfg, models)["auto_compact_windows"]; !reflect.DeepEqual(got, windows) {
		t.Fatalf("bootstrap and cache compaction windows differ: %#v", got)
	}
}
