package launcher

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bassner/claudodex/internal/modelcatalog"
	"github.com/bassner/claudodex/internal/modelconfig"
)

func TestCompactionCacheIsSeededAndIsolatedFromRealClaude(t *testing.T) {
	userHome, home := t.TempDir(), t.TempDir()
	t.Setenv("HOME", userHome)
	canonical := filepath.Join(userHome, ".claude.json")
	if err := writeJSONFile(canonical, map[string]any{"autoCompactWindowsCache": map[string]any{"real-claude-model": 123456}}, 0o600); err != nil {
		t.Fatal(err)
	}
	sidecar, err := PrepareClaudeConfigSidecar(home, modelconfig.Default())
	if err != nil {
		t.Fatal(err)
	}
	models := nativeCatalogFixture()
	globalPath := filepath.Join(sidecar, claudeGlobalConfigName)
	stale := mustReadJSONMap(t, globalPath)
	stale["clientDataCache"] = map[string]any{"rowan_thicket": map[string]any{"gpt-6.1-sol": 100000}}
	stale["clientDataCacheSlots"] = map[string]any{"test-slot": map[string]any{"data": map[string]any{"rowan_thicket": map[string]any{"gpt-6.1-sol": 100000}, "keep": true}}}
	if err := writeJSONFile(globalPath, stale, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteClaudeModelCapabilitiesCache(sidecar, models, modelconfig.Default()); err != nil {
		t.Fatal(err)
	}
	mirror, err := StartClaudeConfigMirror(context.Background(), sidecar, modelconfig.Default())
	if err != nil {
		t.Fatal(err)
	}
	if err := mirror.Close(); err != nil {
		t.Fatal(err)
	}
	real := mustReadJSONMap(t, canonical)["autoCompactWindowsCache"].(map[string]any)
	if len(real) != 1 || real["real-claude-model"] != float64(123456) {
		t.Fatalf("Codex limits escaped into real Claude config: %#v", real)
	}
	windows, _ := modelcatalog.AutoCompactWindows(models, modelconfig.Default())
	config := mustReadJSONMap(t, filepath.Join(sidecar, claudeGlobalConfigName))
	if mapValue(config["clientDataCache"])["rowan_thicket"] != nil || mapValue(mapValue(mapValue(config["clientDataCacheSlots"])["test-slot"])["data"])["rowan_thicket"] != nil {
		t.Fatal("stale client-data compaction window still overrides the live cache")
	}
	seeded := config["autoCompactWindowsCache"].(map[string]any)
	if len(seeded) != len(windows) || seeded[modelconfig.DefaultOpusModel] != float64(windows[modelconfig.DefaultOpusModel]) {
		t.Fatalf("mirror overwrote the seeded compaction windows: %#v", seeded)
	}
}

func TestNativeModelCompactionRequiresVerifiedVersion(t *testing.T) {
	for version, want := range map[string]bool{"2.1.314": true, "2.1.316": true, "2.1.296": false, "": false, "2.1.317": true, "2.1.318": true, "2.1.319": true} {
		if supportsNativeModelCompaction(version) != want {
			t.Fatalf("native compaction support for %s = %t", version, !want)
		}
	}
}
