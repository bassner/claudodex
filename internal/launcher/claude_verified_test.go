package launcher

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

func registeredClaudeFixture(t *testing.T, version string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), version)
	data := []byte("#!/bin/sh\n# ORIGINAL\necho '" + version + " (Claude Code)'\n")
	if err := os.WriteFile(path, data, 0o700); err != nil {
		t.Fatal(err)
	}
	claudeUIPatches = append([]claudeUIPatchSpec{{
		Version: version, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, SHA256: sha256Hex(data),
		Apply: func(data []byte, _, _ string, _ modelconfig.Config) bool {
			index := bytes.Index(data, []byte("ORIGINAL"))
			if index < 0 {
				return false
			}
			copy(data[index:], []byte("PATCHED!"))
			return true
		},
	}}, claudeUIPatches...)
	return path
}

func withClaudeFixtures(t *testing.T) *int {
	t.Helper()
	oldPatches, oldSigner := claudeUIPatches, signPatchedClaude
	t.Cleanup(func() { claudeUIPatches = oldPatches; signPatchedClaude = oldSigner })
	t.Setenv("CLAUDODEX_DISABLE_CLAUDE_PATCH", "")
	signs := new(int)
	signPatchedClaude = func(_ context.Context, path string) error {
		*signs++
		// Real codesign changes bytes too. A valid cached output must match its
		// recorded signed hash, rather than the fresh transformation's hash.
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
		if err != nil {
			return err
		}
		_, err = f.WriteString("\n# signed\n")
		closeErr := f.Close()
		if err != nil {
			return err
		}
		return closeErr
	}
	return signs
}

func TestStrictClaudeSelectCachesOriginalAfterPatchAndFallsBack(t *testing.T) {
	signs := withClaudeFixtures(t)
	home := t.TempDir()
	supported := registeredClaudeFixture(t, "2.1.999")
	selected, err := selectClaudeExecutable(context.Background(), home, supported, "test", modelconfig.Default(), nil)
	if err != nil {
		t.Fatal(err)
	}
	selected.Release()
	original, _ := os.ReadFile(supported)
	snapshot, _ := os.ReadFile(selected.SourcePath)
	if !bytes.Equal(original, snapshot) {
		t.Fatal("retained original differs from original source")
	}
	patched, _ := os.ReadFile(selected.Executable)
	if !bytes.Contains(patched, []byte("PATCHED!")) {
		t.Fatal("selected executable is not transformed")
	}
	unsupported := filepath.Join(t.TempDir(), "3.0.0")
	if err := os.WriteFile(unsupported, []byte("unsupported"), 0o700); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	fallback, err := selectClaudeExecutable(context.Background(), home, unsupported, "test", modelconfig.Default(), &stderr)
	if err != nil {
		t.Fatal(err)
	}
	defer fallback.Release()
	if fallback.SourcePath != selected.SourcePath || fallback.Executable != selected.Executable || fallback.Version != "2.1.999" {
		t.Fatalf("wrong fallback: %#v", fallback)
	}
	if !strings.Contains(stderr.String(), "using verified local Claude Code 2.1.999") {
		t.Fatalf("missing fallback notice: %s", stderr.String())
	}
	if *signs != 1 {
		t.Fatalf("cached signed output was unnecessarily rebuilt: signs=%d", *signs)
	}
}

func TestStrictClaudeNoFallbackFailsBeforeAuthAndSettings(t *testing.T) {
	withClaudeFixtures(t)
	home := t.TempDir()
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte("#!/bin/sh\necho '9.9.9 (Claude Code)'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	err := (ProcessLauncher{}).Launch(context.Background(), nil, Config{Home: home})
	if err == nil || !strings.Contains(err.Error(), "claude install 2.1.314") || !strings.Contains(err.Error(), "CLAUDODEX_DISABLE_CLAUDE_PATCH=1") {
		t.Fatalf("missing actionable strict failure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claudodex", "claude-config")); !os.IsNotExist(err) {
		t.Fatalf("settings were prepared before verification: %v", err)
	}
}

func TestClaudeAdministrativePassthroughKeepsOriginalEnvironment(t *testing.T) {
	home := t.TempDir()
	bin := t.TempDir()
	script := "#!/bin/sh\nprintf '%s' \"$DISABLE_TELEMETRY|${CLAUDE_CONFIG_DIR-unset}|${CLAUDODEX_CHILD-unset}|$HTTP_PROXY|$HTTPS_PROXY|$1|$2\"\n"
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("DISABLE_TELEMETRY", "original")
	t.Setenv("CLAUDE_CONFIG_DIR", "original-config")
	t.Setenv("CLAUDODEX_CHILD", "original-child")
	t.Setenv("HTTP_PROXY", "http://original-http")
	t.Setenv("HTTPS_PROXY", "http://original-https")
	var stdout bytes.Buffer
	if err := (ProcessLauncher{}).Launch(context.Background(), []string{"auth", "status"}, Config{Home: home, Stdout: &stdout}); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "1|original-config|original-child|http://original-http|http://original-https|auth|status" {
		t.Fatalf("passthrough env changed: %s", stdout.String())
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Fatal("administrative pass-through wrote compatibility state")
	}
}

func TestStrictClaudeNativeRetainedSourcesNewestFirst(t *testing.T) {
	withClaudeFixtures(t)
	home := t.TempDir()
	dir := filepath.Join(home, ".local", "share", "claude", "versions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"2.1.998", "2.1.999"} {
		fixture := registeredClaudeFixture(t, version)
		data, _ := os.ReadFile(fixture)
		if err := os.WriteFile(filepath.Join(dir, version), data, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	installed := filepath.Join(dir, "3.0.0")
	if err := os.WriteFile(installed, []byte("unsupported"), 0o700); err != nil {
		t.Fatal(err)
	}
	selected, err := selectClaudeExecutable(context.Background(), home, installed, "test", modelconfig.Default(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer selected.Release()
	if selected.Version != "2.1.999" {
		t.Fatalf("selected version = %s", selected.Version)
	}
	data, _ := os.ReadFile(installed)
	if string(data) != "unsupported" {
		t.Fatal("system Claude changed")
	}
}

func TestStrictClaudePatchAndSourceCorruption(t *testing.T) {
	signs := withClaudeFixtures(t)
	home := t.TempDir()
	fixture := registeredClaudeFixture(t, "2.1.999")
	selected, err := selectClaudeExecutable(context.Background(), home, fixture, "test", modelconfig.Default(), nil)
	if err != nil {
		t.Fatal(err)
	}
	selected.Release()
	for _, corruption := range []string{"bytes", "permissions", "record", "symlink"} {
		t.Run(corruption, func(t *testing.T) {
			switch corruption {
			case "bytes":
				if err := os.WriteFile(selected.Executable, []byte("corrupt"), 0o700); err != nil {
					t.Fatal(err)
				}
			case "permissions":
				if err := os.Chmod(selected.Executable, 0o755); err != nil {
					t.Fatal(err)
				}
			case "record":
				if err := os.WriteFile(filepath.Join(filepath.Dir(selected.Executable), "record.json"), []byte(`{}`), 0o600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Remove(selected.Executable); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(fixture, selected.Executable); err != nil {
					t.Fatal(err)
				}
			}
			before := *signs
			rebuilt, err := selectClaudeExecutable(context.Background(), home, fixture, "test", modelconfig.Default(), nil)
			if err != nil {
				t.Fatal(err)
			}
			rebuilt.Release()
			if *signs != before+1 {
				t.Fatal("corrupt patched cache was trusted")
			}
			info, _ := os.Lstat(selected.Executable)
			if !info.Mode().IsRegular() || info.Mode().Perm() != 0o700 {
				t.Fatal("cache was not rebuilt as a private regular executable")
			}
		})
	}
	if err := os.WriteFile(selected.SourcePath, []byte("corrupt source"), 0o700); err != nil {
		t.Fatal(err)
	}
	unsupported := filepath.Join(t.TempDir(), "3.0.0")
	if err := os.WriteFile(unsupported, []byte("unsupported"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := selectClaudeExecutable(context.Background(), home, unsupported, "test", modelconfig.Default(), nil); err == nil {
		t.Fatal("corrupt retained original was trusted")
	}
}

func TestStrictClaudeFailedTransformationDoesNotRetainSource(t *testing.T) {
	withClaudeFixtures(t)
	path := registeredClaudeFixture(t, "2.1.999")
	claudeUIPatches[0].Apply = func([]byte, string, string, modelconfig.Config) bool { return false }
	home := t.TempDir()
	if _, err := selectClaudeExecutable(context.Background(), home, path, "test", modelconfig.Default(), nil); err == nil {
		t.Fatal("failed patch selected")
	}
	root, _ := claudeCacheRoot(home)
	entries, _ := os.ReadDir(filepath.Join(root, "sources"))
	if len(entries) != 0 {
		t.Fatal("original retained before successful transformation")
	}
}

func TestClaudeSourceRetentionProtectsLiveLeaseAndLegacyFiles(t *testing.T) {
	withClaudeFixtures(t)
	home := t.TempDir()
	var selections []claudeSelection
	for _, version := range []string{"2.1.997", "2.1.998", "2.1.999"} {
		fixture := registeredClaudeFixture(t, version)
		selected, err := selectClaudeExecutable(context.Background(), home, fixture, "test", modelconfig.Default(), nil)
		if err != nil {
			t.Fatal(err)
		}
		selections = append(selections, selected)
		if version != "2.1.997" {
			selected.Release()
		}
	}
	if _, err := os.Stat(selections[0].SourcePath); err != nil {
		t.Fatal("live lease did not protect old source")
	}
	if _, err := os.Stat(selections[0].Executable); err != nil {
		t.Fatal("live lease did not protect old patch")
	}
	root, _ := claudeCacheRoot(home)
	legacy := filepath.Join(root, "legacy-unrecorded")
	if err := os.Mkdir(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	selections[0].Release()
	retainClaudeSources(context.Background(), root, "test", modelconfig.Default())
	if _, err := os.Stat(selections[0].SourcePath); !os.IsNotExist(err) {
		t.Fatal("old unused source was not removed")
	}
	if _, err := os.Stat(selections[0].Executable); !os.IsNotExist(err) {
		t.Fatal("old unused patch was not removed")
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Fatal("unrecorded legacy cache removed")
	}
}

func TestClaudeSourceRetentionUnknownLeaseDisablesDeletion(t *testing.T) {
	withClaudeFixtures(t)
	home := t.TempDir()
	old := registeredClaudeFixture(t, "2.1.997")
	selected, err := selectClaudeExecutable(context.Background(), home, old, "test", modelconfig.Default(), nil)
	if err != nil {
		t.Fatal(err)
	}
	selected.Release()
	root, _ := claudeCacheRoot(home)
	if err := os.WriteFile(filepath.Join(root, "leases", "unknown.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"2.1.998", "2.1.999"} {
		fixture := registeredClaudeFixture(t, version)
		s, err := selectClaudeExecutable(context.Background(), home, fixture, "test", modelconfig.Default(), nil)
		if err != nil {
			t.Fatal(err)
		}
		s.Release()
	}
	if _, err := os.Stat(selected.SourcePath); err != nil {
		t.Fatal("unknown lease did not disable source deletion")
	}
}

func TestClaudePatchRetentionRemovesObsoleteVersionAndModelsAfterLeaseEnds(t *testing.T) {
	for _, change := range []string{"version", "models", "schema"} {
		t.Run(change, func(t *testing.T) {
			withClaudeFixtures(t)
			home := t.TempDir()
			fixture := registeredClaudeFixture(t, "2.1.999")
			old, err := selectClaudeExecutable(context.Background(), home, fixture, "old", modelconfig.Default(), nil)
			if err != nil {
				t.Fatal(err)
			}
			version := "old"
			models := modelconfig.Default()
			switch change {
			case "version":
				version = "current"
			case "models":
				models.Opus = "custom-opus"
			case "schema":
				var record claudePatchRecord
				path := filepath.Join(filepath.Dir(old.Executable), "record.json")
				if err := readPrivateJSON(path, &record); err != nil {
					t.Fatal(err)
				}
				record.Schema = "old-schema"
				if err := atomicPrivateJSON(path, record); err != nil {
					t.Fatal(err)
				}
			}
			root, _ := claudeCacheRoot(home)
			retainClaudeSources(context.Background(), root, version, models)
			if _, err := os.Stat(old.Executable); err != nil {
				t.Fatal("live lease did not protect obsolete patch")
			}
			old.Release()
			retainClaudeSources(context.Background(), root, version, models)
			if _, err := os.Stat(old.Executable); !os.IsNotExist(err) {
				t.Fatal("obsolete patch for a retained source was not removed")
			}
			if _, err := os.Stat(old.SourcePath); err != nil {
				t.Fatal("retained source was removed with its obsolete patch")
			}
		})
	}
}

func TestClaudeRetentionRemovesDeadLeaseWithoutStartTime(t *testing.T) {
	withClaudeFixtures(t)
	home := t.TempDir()
	fixture := registeredClaudeFixture(t, "2.1.999")
	selected, err := selectClaudeExecutable(context.Background(), home, fixture, "old", modelconfig.Default(), nil)
	if err != nil {
		t.Fatal(err)
	}
	selected.Release()
	cmd := exec.Command("/bin/sh", "-c", "exit 0")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	root, _ := claudeCacheRoot(home)
	path := filepath.Join(root, "leases", "dead-no-start.json")
	if err := atomicPrivateJSON(path, claudeLease{PID: cmd.Process.Pid}); err != nil {
		t.Fatal(err)
	}
	retainClaudeSources(context.Background(), root, "current", modelconfig.Default())
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("dead incomplete lease was not removed")
	}
	if _, err := os.Stat(selected.Executable); !os.IsNotExist(err) {
		t.Fatal("dead incomplete lease still blocked obsolete-patch cleanup")
	}
}

func TestClaudeRetentionLiveLeaseWithoutStartTimeRemainsConservative(t *testing.T) {
	withClaudeFixtures(t)
	home := t.TempDir()
	fixture := registeredClaudeFixture(t, "2.1.999")
	selected, err := selectClaudeExecutable(context.Background(), home, fixture, "old", modelconfig.Default(), nil)
	if err != nil {
		t.Fatal(err)
	}
	selected.Release()
	root, _ := claudeCacheRoot(home)
	path := filepath.Join(root, "leases", "live-no-start.json")
	if err := atomicPrivateJSON(path, claudeLease{PID: os.Getpid()}); err != nil {
		t.Fatal(err)
	}
	retainClaudeSources(context.Background(), root, "current", modelconfig.Default())
	if _, err := os.Stat(selected.Executable); err != nil {
		t.Fatal("uncertain live lease did not block deletion")
	}
}

func TestPinStatusLineExecutableSurvivesOriginalRemovalAndRepairsCorruption(t *testing.T) {
	home := t.TempDir()
	original := filepath.Join(t.TempDir(), "claudodex")
	data := []byte("#!/bin/sh\necho pinned\n")
	if err := os.WriteFile(original, data, 0o755); err != nil {
		t.Fatal(err)
	}
	pinned, err := PinStatusLineExecutable(home, original)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pinned, []byte("corrupt"), 0o700); err != nil {
		t.Fatal(err)
	}
	repaired, err := PinStatusLineExecutable(home, original)
	if err != nil || repaired != pinned {
		t.Fatalf("pin repair = %q, %v", repaired, err)
	}
	if err := os.Remove(original); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(pinned)
	if err != nil || !bytes.Equal(actual, data) {
		t.Fatalf("pinned helper did not survive original removal: %v", err)
	}
}

func TestSelectedFallbackSourceControlsVersionSpecificSettings(t *testing.T) {
	withClaudeFixtures(t)
	home := t.TempDir()
	fixture := registeredClaudeFixture(t, "2.1.314")
	selected, err := selectClaudeExecutable(context.Background(), home, fixture, "test", modelconfig.Default(), nil)
	if err != nil {
		t.Fatal(err)
	}
	selected.Release()
	unsupported := filepath.Join(t.TempDir(), "3.0.0")
	if err := os.WriteFile(unsupported, []byte("unsupported"), 0o700); err != nil {
		t.Fatal(err)
	}
	fallback, err := selectClaudeExecutable(context.Background(), home, unsupported, "test", modelconfig.Default(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer fallback.Release()
	if !claudeFastModeSettingsFallbackRequired(context.Background(), fallback.SourcePath) || !claudeThreeTierPickerArgsRequired(context.Background(), fallback.SourcePath) || !claudeSourcePatchedPickerArgsRequired(context.Background(), fallback.SourcePath) {
		t.Fatal("fallback source did not retain supported settings decisions")
	}
	var record claudePatchRecord
	data, _ := os.ReadFile(filepath.Join(filepath.Dir(fallback.Executable), "record.json"))
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	if record.SourceSHA256 != sha256Hex(mustReadTestFile(t, fixture)) {
		t.Fatal("patch record is not bound to selected source")
	}
}

func mustReadTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
