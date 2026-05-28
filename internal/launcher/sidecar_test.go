package launcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
)

func TestPrepareClaudeConfigSidecarLinksUserStateAndWritesLocalOAuth(t *testing.T) {
	claudodexHome := t.TempDir()
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)

	realClaudeDir := filepath.Join(userHome, ".claude")
	if err := os.MkdirAll(realClaudeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(realClaudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	globalConfigPath := filepath.Join(userHome, ".claude.json")
	if err := os.WriteFile(globalConfigPath, []byte(`{
  "hasCompletedOnboarding": true,
  "theme": "light",
  "projects": {"/repo": {"hasTrustDialogAccepted": true}},
  "oauthAccount": {"emailAddress": "real@example.com"},
  "primaryApiKey": "real-key",
  "customApiKeyResponses": {"approved": ["sk-ant"], "rejected": []}
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	sidecarDir, err := PrepareClaudeConfigSidecar(claudodexHome, modelconfig.Default())
	if err != nil {
		t.Fatal(err)
	}

	if got, err := os.Readlink(filepath.Join(sidecarDir, "settings.json")); err != nil || got != settingsPath {
		t.Fatalf("settings symlink = %q, %v; want %q", got, err, settingsPath)
	}
	if _, err := os.Readlink(filepath.Join(sidecarDir, ".claude-local-oauth.json")); err == nil {
		t.Fatalf("local OAuth config should be owned by sidecar, not symlinked to %q", globalConfigPath)
	}
	configData, err := os.ReadFile(filepath.Join(sidecarDir, ".claude-local-oauth.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatal(err)
	}
	if config["theme"] != "light" || config["oauthAccount"] != nil || config["primaryApiKey"] != nil || config["customApiKeyResponses"] != nil {
		t.Fatalf("local OAuth config = %#v", config)
	}
	projects, ok := config["projects"].(map[string]any)
	if !ok || projects["/repo"] == nil {
		t.Fatalf("projects missing from local OAuth config: %#v", config)
	}

	data, err := os.ReadFile(filepath.Join(sidecarDir, ".credentials.json"))
	if err != nil {
		t.Fatal(err)
	}
	var credentials claudeCredentials
	if err := json.Unmarshal(data, &credentials); err != nil {
		t.Fatal(err)
	}
	if credentials.ClaudeAIOAuth.AccessToken != "claudodex-local-oauth" {
		t.Fatalf("access token = %q", credentials.ClaudeAIOAuth.AccessToken)
	}
	if !contains(credentials.ClaudeAIOAuth.Scopes, "user:profile") || !contains(credentials.ClaudeAIOAuth.Scopes, "user:inference") {
		t.Fatalf("scopes = %#v", credentials.ClaudeAIOAuth.Scopes)
	}
	info, err := os.Stat(filepath.Join(sidecarDir, ".credentials.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("credentials mode = %o, want 600", info.Mode().Perm())
	}
	shellPath := filepath.Join(sidecarDir, claudodexShimDirName, "zsh")
	shellInfo, err := os.Stat(shellPath)
	if err != nil {
		t.Fatal(err)
	}
	if shellInfo.Mode().Perm() != 0o700 {
		t.Fatalf("shell shim mode = %o, want 700", shellInfo.Mode().Perm())
	}
	if runtime.GOOS == "darwin" {
		shimPath := filepath.Join(sidecarDir, claudodexShimDirName, "security")
		shimInfo, err := os.Stat(shimPath)
		if err != nil {
			t.Fatal(err)
		}
		if shimInfo.Mode().Perm() != 0o700 {
			t.Fatalf("security shim mode = %o, want 700", shimInfo.Mode().Perm())
		}
		shim, err := os.ReadFile(shimPath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(shim), "Claude Code") || !strings.Contains(string(shim), "/usr/bin/security") {
			t.Fatalf("security shim missing expected filters:\n%s", shim)
		}
	}
}

func TestClaudeSecurityShimBlocksClaudeCodeCredentialCommands(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-only shim")
	}
	sidecarDir := t.TempDir()
	if err := writeClaudeShims(sidecarDir); err != nil {
		t.Fatal(err)
	}
	shimPath := filepath.Join(sidecarDir, claudodexShimDirName, "security")

	cmd := exec.Command(shimPath, "find-generic-password", "-a", "pat", "-w", "-s", "Claude Code-credentials-deadbeef")
	if err := cmd.Run(); err == nil {
		t.Fatal("find-generic-password for Claude Code service succeeded, want blocked")
	} else if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 44 {
		t.Fatalf("find-generic-password exit = %v, want 44", err)
	}

	cmd = exec.Command(shimPath, "-i")
	cmd.Stdin = strings.NewReader(`add-generic-password -U -a "pat" -s "Claude Code-credentials-deadbeef" -X "00"` + "\n")
	if err := cmd.Run(); err == nil {
		t.Fatal("stdin add-generic-password for Claude Code service succeeded, want blocked")
	} else if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 44 {
		t.Fatalf("stdin add-generic-password exit = %v, want 44", err)
	}

	cmd = exec.Command(shimPath, "add-generic-password", "-U", "-a", "pat", "-s", "not-claude", "-X", "00")
	if err := cmd.Run(); err == nil {
		t.Fatal("add-generic-password succeeded, want blocked")
	} else if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 44 {
		t.Fatalf("add-generic-password exit = %v, want 44", err)
	}
}

func TestClaudeShellShimRestoresOriginalToolEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell shim is POSIX-only")
	}
	sidecarDir := t.TempDir()
	if err := writeClaudeShims(sidecarDir); err != nil {
		t.Fatal(err)
	}
	shimPath := filepath.Join(sidecarDir, claudodexShimDirName, "sh")
	cmd := exec.Command(shimPath, "-c", `printf '%s\n' "$SHELL|$HTTP_PROXY|$HTTPS_PROXY|${NO_PROXY-unset}|${NODE_EXTRA_CA_CERTS-unset}|${CLAUDODEX_REAL_SHELL-unset}|${CLAUDODEX_ORIGINAL_HTTPS_PROXY-unset}"`)
	cmd.Env = []string{
		"CLAUDODEX_REAL_SHELL=/bin/sh",
		"SHELL=/tmp/claudodex-shim/sh",
		"HTTP_PROXY=http://claudodex-internal",
		"HTTPS_PROXY=http://claudodex-internal",
		"NO_PROXY=127.0.0.1,localhost",
		"NODE_EXTRA_CA_CERTS=/tmp/claudodex-ca.pem",
		"CLAUDODEX_ORIGINAL_SHELL=/opt/homebrew/bin/fish",
		"CLAUDODEX_ORIGINAL_HTTP_PROXY=http://user-http",
		"CLAUDODEX_ORIGINAL_HTTPS_PROXY=http://user-https",
		"CLAUDODEX_ORIGINAL_NO_PROXY=.example.com",
	}
	output, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(output))
	want := "/opt/homebrew/bin/fish|http://user-http|http://user-https|.example.com|unset|unset|unset"
	if got != want {
		t.Fatalf("restored env = %q, want %q", got, want)
	}
}

func TestWriteClaudeModelCapabilitiesCacheUsesPrivateSidecarCache(t *testing.T) {
	sidecarDir := t.TempDir()
	realCacheDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(realCacheDir, claudeModelCapabilitiesFileName), []byte(`{"models":[],"timestamp":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realCacheDir, filepath.Join(sidecarDir, "cache")); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(filepath.Join(sidecarDir, claudeGlobalConfigName), map[string]any{
		"oauthAccount": map[string]any{
			"displayName":       "Claudodex",
			"display_name":      "Claudodex",
			"emailAddress":      "fake@example.com",
			"organizationName":  "Claudodex",
			"organization_name": "Claudodex",
		},
	}, 0o600); err != nil {
		t.Fatal(err)
	}

	err := WriteClaudeModelCapabilitiesCache(sidecarDir, []codex.ModelInfo{
		{Slug: "gpt-5.5", ContextWindow: 272000},
		{Slug: "gpt-5.4", ContextWindow: 300000},
		{Slug: "gpt-5.4-mini", ContextWindow: 400000},
	}, modelconfig.Default())
	if err != nil {
		t.Fatal(err)
	}

	cacheDir := filepath.Join(sidecarDir, "cache")
	if _, err := os.Readlink(cacheDir); err == nil {
		t.Fatal("sidecar cache is still a symlink to the real Claude cache")
	}
	info, err := os.Stat(cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() || info.Mode().Perm() != 0o700 {
		t.Fatalf("cache dir mode = %o, isDir=%v; want 700 dir", info.Mode().Perm(), info.IsDir())
	}
	cachePath := filepath.Join(cacheDir, claudeModelCapabilitiesFileName)
	cache := mustReadJSONMap(t, cachePath)
	models := cache["models"].([]any)
	if len(models) != 10 {
		t.Fatalf("models length = %d, want 10: %#v", len(models), models)
	}
	foundSonnet := false
	for _, item := range models {
		model := item.(map[string]any)
		if strings.Contains(model["id"].(string), "[1m]") {
			t.Fatalf("long-context runtime suffix leaked into capabilities cache: %#v", models)
		}
		if model["id"] == "claude-sonnet-4-6" {
			foundSonnet = true
			if model["max_input_tokens"] != float64(300000) || model["max_tokens"] != float64(128000) {
				t.Fatalf("sonnet capability = %#v", model)
			}
		}
	}
	if !foundSonnet {
		t.Fatalf("claude-sonnet-4-6 capability missing: %#v", models)
	}
	realCache := mustReadJSONMap(t, filepath.Join(realCacheDir, claudeModelCapabilitiesFileName))
	if len(realCache["models"].([]any)) != 0 {
		t.Fatalf("real Claude cache was modified: %#v", realCache)
	}
	globalConfig := mustReadJSONMap(t, filepath.Join(sidecarDir, claudeGlobalConfigName))
	clientData := globalConfig["clientDataCache"].(map[string]any)
	if clientData["kelp_forest_sonnet"] != "300000" {
		t.Fatalf("kelp_forest_sonnet = %#v, want 300000", clientData["kelp_forest_sonnet"])
	}
	options := globalConfig["additionalModelOptionsCache"].([]any)
	if len(options) != 3 {
		t.Fatalf("additional model options length = %d, want 3: %#v", len(options), options)
	}
	first := options[0].(map[string]any)
	if first["value"] != "gpt-5.5[1m]" || first["label"] != "gpt-5.5" {
		t.Fatalf("first additional model option = %#v", first)
	}
	oauthAccount := globalConfig["oauthAccount"].(map[string]any)
	if oauthAccount["displayName"] != "" || oauthAccount["display_name"] != "" {
		t.Fatalf("sidecar oauth display names were not suppressed: %#v", oauthAccount)
	}
	if oauthAccount["organizationName"] != "" || oauthAccount["organization_name"] != "" {
		t.Fatalf("sidecar oauth organization names were not suppressed: %#v", oauthAccount)
	}
	if oauthAccount["emailAddress"] != "fake@example.com" {
		t.Fatalf("sidecar oauth account metadata was not preserved: %#v", oauthAccount)
	}
}

func TestPrepareClaudeConfigSidecarConcurrentLaunches(t *testing.T) {
	claudodexHome := t.TempDir()
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)

	realClaudeDir := filepath.Join(userHome, ".claude")
	if err := os.MkdirAll(realClaudeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(realClaudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userHome, ".claude.json"), []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	errs := make(chan error, 8)
	for i := 0; i < cap(errs); i++ {
		go func() {
			_, err := PrepareClaudeConfigSidecar(claudodexHome, modelconfig.Default())
			errs <- err
		}()
	}
	for i := 0; i < cap(errs); i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}

	sidecarDir := filepath.Join(claudodexHome, ".claudodex", claudeSidecarDirName)
	if got, err := os.Readlink(filepath.Join(sidecarDir, "settings.json")); err != nil || got != settingsPath {
		t.Fatalf("settings symlink = %q, %v; want %q", got, err, settingsPath)
	}
}

func TestWithClaudeConfigLocksWaitsForTransientLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	lockPath := path + ".lock"
	if err := os.MkdirAll(lockPath, 0o700); err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	go func() {
		defer close(release)
		time.Sleep(50 * time.Millisecond)
		_ = os.RemoveAll(lockPath)
	}()

	called := false
	if err := withClaudeConfigLocks([]string{path}, time.Second, func() error {
		called = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	<-release
	if !called {
		t.Fatal("lock callback was not called")
	}
}

func TestNormalizeClaudeSettingsModelMapsCodexRuntimeModels(t *testing.T) {
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	if err := writeJSONFile(settingsPath, map[string]any{
		"model":       "gpt-5.4[1m][1m]",
		"effortLevel": "xhigh",
	}, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := normalizeClaudeSettingsModel(settingsPath, time.Second, modelconfig.Default()); err != nil {
		t.Fatal(err)
	}

	settings := mustReadJSONMap(t, settingsPath)
	if settings["model"] != "sonnet" {
		t.Fatalf("model = %#v, want sonnet", settings["model"])
	}
	if settings["effortLevel"] != "xhigh" {
		t.Fatalf("effortLevel was changed: %#v", settings)
	}
}

func TestNormalizeClaudeSettingsModelUsesConfiguredTargets(t *testing.T) {
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	if err := writeJSONFile(settingsPath, map[string]any{
		"model": "gpt-sonnet-next[1m]",
	}, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := normalizeClaudeSettingsModel(settingsPath, time.Second, modelconfig.Config{Sonnet: "gpt-sonnet-next"}); err != nil {
		t.Fatal(err)
	}

	settings := mustReadJSONMap(t, settingsPath)
	if settings["model"] != "sonnet" {
		t.Fatalf("model = %#v, want sonnet", settings["model"])
	}
}

func TestNormalizeSharedClaudeSettingsDoesNotCreateProjectClaudeDir(t *testing.T) {
	userHome := t.TempDir()
	projectDir := t.TempDir()
	t.Chdir(projectDir)
	if err := normalizeSharedClaudeSettings(userHome, time.Second, modelconfig.Default()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".claude")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("project .claude dir was created or stat failed: %v", err)
	}
}

func TestPrepareClaudeConfigSidecarReconcilesExistingSidecar(t *testing.T) {
	claudodexHome := t.TempDir()
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)

	realClaudeDir := filepath.Join(userHome, ".claude")
	if err := os.MkdirAll(realClaudeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	realPath := filepath.Join(userHome, ".claude.json")
	if err := writeJSONFile(realPath, map[string]any{
		"theme":        "dark",
		"oauthAccount": map[string]any{"emailAddress": "real@example.com"},
	}, 0o600); err != nil {
		t.Fatal(err)
	}

	sidecarDir := filepath.Join(claudodexHome, ".claudodex", claudeSidecarDirName)
	if err := os.MkdirAll(sidecarDir, 0o700); err != nil {
		t.Fatal(err)
	}
	sidecarPath := filepath.Join(sidecarDir, claudeLocalOAuthConfigName)
	if err := writeJSONFile(sidecarPath, map[string]any{
		"theme":        "light",
		"oauthAccount": map[string]any{"emailAddress": "fake@example.com"},
	}, 0o600); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Unix(1000, 0)
	newTime := time.Unix(2000, 0)
	if err := os.Chtimes(realPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(sidecarPath, newTime, newTime); err != nil {
		t.Fatal(err)
	}

	gotDir, err := PrepareClaudeConfigSidecar(claudodexHome, modelconfig.Default())
	if err != nil {
		t.Fatal(err)
	}
	if gotDir != sidecarDir {
		t.Fatalf("sidecar dir = %q, want %q", gotDir, sidecarDir)
	}

	realAfter := mustReadJSONMap(t, realPath)
	if realAfter["theme"] != "light" {
		t.Fatalf("real theme = %q, want light", realAfter["theme"])
	}
	if got := realAfter["oauthAccount"].(map[string]any)["emailAddress"]; got != "real@example.com" {
		t.Fatalf("real oauth account = %q", got)
	}

	sidecarAfter := mustReadJSONMap(t, sidecarPath)
	if sidecarAfter["theme"] != "light" {
		t.Fatalf("sidecar theme = %q, want light", sidecarAfter["theme"])
	}
	if got := sidecarAfter["oauthAccount"].(map[string]any)["emailAddress"]; got != "fake@example.com" {
		t.Fatalf("sidecar oauth account = %q", got)
	}
}

func TestReconcileClaudeGlobalConfigMirrorsSafeDeltasBothWays(t *testing.T) {
	userHome := t.TempDir()
	sidecarDir := t.TempDir()
	realPath := filepath.Join(userHome, ".claude.json")
	sidecarPath := filepath.Join(sidecarDir, ".claude.json")
	sidecarLocalPath := filepath.Join(sidecarDir, ".claude-local-oauth.json")

	realInitial := map[string]any{
		"theme": "dark",
		"projects": map[string]any{
			"/a": map[string]any{"hasTrustDialogAccepted": true},
		},
		"oauthAccount":  map[string]any{"emailAddress": "real@example.com"},
		"primaryApiKey": "real-key",
	}
	if err := writeJSONFile(realPath, realInitial, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(sidecarPath, sanitizeGlobalConfig(realInitial), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(sidecarLocalPath, sanitizeGlobalConfig(realInitial), 0o600); err != nil {
		t.Fatal(err)
	}
	baseline := sanitizeGlobalConfig(realInitial)

	realChanged := map[string]any{
		"theme":   "dark",
		"verbose": true,
		"projects": map[string]any{
			"/a": map[string]any{"hasTrustDialogAccepted": true},
			"/c": map[string]any{"hasTrustDialogAccepted": true},
		},
		"oauthAccount":  map[string]any{"emailAddress": "real@example.com"},
		"primaryApiKey": "real-key",
	}
	sidecarChanged := map[string]any{
		"theme": "light",
		"projects": map[string]any{
			"/a": map[string]any{"hasTrustDialogAccepted": true},
			"/b": map[string]any{"hasTrustDialogAccepted": true},
		},
		"oauthAccount":  map[string]any{"emailAddress": "fake@example.com"},
		"primaryApiKey": "fake-key",
	}
	if err := writeJSONFile(realPath, realChanged, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(sidecarPath, sidecarChanged, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := reconcileClaudeGlobalConfig(sidecarDir, userHome, &baseline, time.Second); err != nil {
		t.Fatal(err)
	}

	realAfter := mustReadJSONMap(t, realPath)
	if realAfter["theme"] != "light" || realAfter["verbose"] != true {
		t.Fatalf("real config did not merge scalar deltas: %#v", realAfter)
	}
	if got := realAfter["oauthAccount"].(map[string]any)["emailAddress"]; got != "real@example.com" {
		t.Fatalf("real oauth account = %q", got)
	}
	if realAfter["primaryApiKey"] != "real-key" {
		t.Fatalf("real primaryApiKey = %q", realAfter["primaryApiKey"])
	}
	assertProjectKeys(t, realAfter, "/a", "/b", "/c")

	sidecarAfter := mustReadJSONMap(t, sidecarPath)
	if sidecarAfter["theme"] != "light" || sidecarAfter["verbose"] != true {
		t.Fatalf("sidecar config did not merge scalar deltas: %#v", sidecarAfter)
	}
	if got := sidecarAfter["oauthAccount"].(map[string]any)["emailAddress"]; got != "fake@example.com" {
		t.Fatalf("sidecar oauth account = %q", got)
	}
	if sidecarAfter["primaryApiKey"] != "fake-key" {
		t.Fatalf("sidecar primaryApiKey = %q", sidecarAfter["primaryApiKey"])
	}
	assertProjectKeys(t, sidecarAfter, "/a", "/b", "/c")
	assertProjectKeys(t, baseline, "/a", "/b", "/c")
}

func TestReconcileClaudeGlobalConfigCombinesBothSidecarFiles(t *testing.T) {
	userHome := t.TempDir()
	sidecarDir := t.TempDir()
	realPath := filepath.Join(userHome, ".claude.json")
	sidecarPath := filepath.Join(sidecarDir, ".claude.json")
	sidecarLocalPath := filepath.Join(sidecarDir, ".claude-local-oauth.json")

	baseline := map[string]any{
		"projects": map[string]any{
			"/a": map[string]any{"hasTrustDialogAccepted": true},
		},
	}
	if err := writeJSONFile(realPath, baseline, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(sidecarPath, map[string]any{
		"projects": map[string]any{
			"/a": map[string]any{"hasTrustDialogAccepted": true},
			"/b": map[string]any{"hasTrustDialogAccepted": true},
		},
	}, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(sidecarLocalPath, map[string]any{
		"projects": map[string]any{
			"/a": map[string]any{"hasTrustDialogAccepted": true},
			"/c": map[string]any{"hasTrustDialogAccepted": true},
		},
	}, 0o600); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Unix(1000, 0)
	midTime := time.Unix(2000, 0)
	newTime := time.Unix(3000, 0)
	for path, at := range map[string]time.Time{
		realPath:         oldTime,
		sidecarPath:      midTime,
		sidecarLocalPath: newTime,
	} {
		if err := os.Chtimes(path, at, at); err != nil {
			t.Fatal(err)
		}
	}

	if err := reconcileClaudeGlobalConfig(sidecarDir, userHome, &baseline, time.Second); err != nil {
		t.Fatal(err)
	}

	assertProjectKeys(t, mustReadJSONMap(t, realPath), "/a", "/b", "/c")
	assertProjectKeys(t, mustReadJSONMap(t, sidecarPath), "/a", "/b", "/c")
	assertProjectKeys(t, mustReadJSONMap(t, sidecarLocalPath), "/a", "/b", "/c")
	assertProjectKeys(t, baseline, "/a", "/b", "/c")
}

func TestClaudeConfigMirrorSyncsWhileRunning(t *testing.T) {
	oldInterval := claudeConfigMirrorInterval
	claudeConfigMirrorInterval = 20 * time.Millisecond
	defer func() { claudeConfigMirrorInterval = oldInterval }()

	userHome := t.TempDir()
	sidecarDir := t.TempDir()
	t.Setenv("HOME", userHome)

	realPath := filepath.Join(userHome, ".claude.json")
	sidecarPath := filepath.Join(sidecarDir, ".claude.json")
	sidecarLocalPath := filepath.Join(sidecarDir, ".claude-local-oauth.json")
	initial := map[string]any{
		"theme": "dark",
		"projects": map[string]any{
			"/initial": map[string]any{"hasTrustDialogAccepted": true},
		},
		"oauthAccount": map[string]any{"emailAddress": "real@example.com"},
	}
	if err := writeJSONFile(realPath, initial, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(sidecarPath, sanitizeGlobalConfig(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(sidecarLocalPath, sanitizeGlobalConfig(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	mirror, err := StartClaudeConfigMirror(context.Background(), sidecarDir, modelconfig.Default())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := mirror.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	if err := writeJSONFile(sidecarLocalPath, map[string]any{
		"theme": "light",
		"projects": map[string]any{
			"/initial": map[string]any{"hasTrustDialogAccepted": true},
			"/sidecar": map[string]any{"hasTrustDialogAccepted": true},
		},
		"oauthAccount": map[string]any{"emailAddress": "fake@example.com"},
	}, 0o600); err != nil {
		t.Fatal(err)
	}

	waitForConfig(t, func() bool {
		real := mustReadJSONMap(t, realPath)
		return real["theme"] == "light" && hasProjectKey(real, "/sidecar")
	})

	if err := writeJSONFile(realPath, map[string]any{
		"theme":   "light",
		"verbose": true,
		"projects": map[string]any{
			"/initial": map[string]any{"hasTrustDialogAccepted": true},
			"/real":    map[string]any{"hasTrustDialogAccepted": true},
			"/sidecar": map[string]any{"hasTrustDialogAccepted": true},
		},
		"oauthAccount": map[string]any{"emailAddress": "real@example.com"},
	}, 0o600); err != nil {
		t.Fatal(err)
	}

	waitForConfig(t, func() bool {
		sidecar := mustReadJSONMap(t, sidecarLocalPath)
		return sidecar["verbose"] == true && hasProjectKey(sidecar, "/real")
	})
}

func TestClaudeConfigMirrorCatchesUpAfterClaudeLockContention(t *testing.T) {
	oldInterval := claudeConfigMirrorInterval
	claudeConfigMirrorInterval = 20 * time.Millisecond
	defer func() { claudeConfigMirrorInterval = oldInterval }()

	userHome := t.TempDir()
	sidecarDir := t.TempDir()
	t.Setenv("HOME", userHome)

	realPath := filepath.Join(userHome, ".claude.json")
	sidecarPath := filepath.Join(sidecarDir, ".claude.json")
	sidecarLocalPath := filepath.Join(sidecarDir, ".claude-local-oauth.json")
	initial := map[string]any{"theme": "dark"}
	if err := writeJSONFile(realPath, initial, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(sidecarPath, initial, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(sidecarLocalPath, initial, 0o600); err != nil {
		t.Fatal(err)
	}

	mirror, err := StartClaudeConfigMirror(context.Background(), sidecarDir, modelconfig.Default())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := mirror.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	lockPath := sidecarLocalPath + ".lock"
	if err := os.Mkdir(lockPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(sidecarLocalPath, map[string]any{"theme": "light"}, 0o600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)
	if got := mustReadJSONMap(t, realPath)["theme"]; got != "dark" {
		t.Fatalf("real theme updated while sidecar lock was held: %q", got)
	}

	if err := os.Remove(lockPath); err != nil {
		t.Fatal(err)
	}
	waitForConfig(t, func() bool {
		return mustReadJSONMap(t, realPath)["theme"] == "light"
	})
}

func TestSyncTranscriptModelDefaultWritesSharedClaudeSetting(t *testing.T) {
	userHome := t.TempDir()
	sidecarDir := t.TempDir()
	settingsPath := filepath.Join(userHome, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(settingsPath, map[string]any{
		"model":       "sonnet",
		"effortLevel": "xhigh",
	}, 0o600); err != nil {
		t.Fatal(err)
	}

	sessionDir := filepath.Join(sidecarDir, "projects", "-tmp-project")
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	transcriptPath := filepath.Join(sessionDir, "session.jsonl")
	line := fmt.Sprintf(`{"timestamp":%q,"message":{"content":"<local-command-stdout>Set model to \u001b[1mgpt-5.4-mini[1m]\u001b[22m and saved as your default for new sessions</local-command-stdout>"}}`+"\n", now.Format(time.RFC3339Nano))
	if err := os.WriteFile(transcriptPath, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}

	cursor, err := syncTranscriptModelDefault(sidecarDir, userHome, time.Second, modelconfig.Default(), now.Add(-time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if !cursor.Equal(now) {
		t.Fatalf("cursor = %s, want %s", cursor, now)
	}

	settings := mustReadJSONMap(t, settingsPath)
	if settings["model"] != "haiku" {
		t.Fatalf("model = %#v, want haiku", settings["model"])
	}
	if settings["effortLevel"] != "xhigh" {
		t.Fatalf("effortLevel was changed: %#v", settings)
	}
}

func TestSyncTranscriptModelDefaultIgnoresSessionOnlyAndOldLines(t *testing.T) {
	userHome := t.TempDir()
	sidecarDir := t.TempDir()
	settingsPath := filepath.Join(userHome, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(settingsPath, map[string]any{"model": "sonnet"}, 0o600); err != nil {
		t.Fatal(err)
	}
	sessionDir := filepath.Join(sidecarDir, "projects", "-tmp-project")
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	since := now.Add(-time.Second)
	lines := strings.Join([]string{
		fmt.Sprintf(`{"timestamp":%q,"message":{"content":"Set model to gpt-5.5[1m] and saved as your default for new sessions"}}`, now.Add(-time.Hour).Format(time.RFC3339Nano)),
		fmt.Sprintf(`{"timestamp":%q,"message":{"content":"Set model to gpt-5.4-mini[1m] for this session only"}}`, now.Format(time.RFC3339Nano)),
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(sessionDir, "session.jsonl"), []byte(lines), 0o600); err != nil {
		t.Fatal(err)
	}

	cursor, err := syncTranscriptModelDefault(sidecarDir, userHome, time.Second, modelconfig.Default(), since)
	if err != nil {
		t.Fatal(err)
	}
	if !cursor.Equal(since) {
		t.Fatalf("cursor = %s, want %s", cursor, since)
	}
	settings := mustReadJSONMap(t, settingsPath)
	if settings["model"] != "sonnet" {
		t.Fatalf("model = %#v, want sonnet", settings["model"])
	}
}

func TestSyncTranscriptModelDefaultCursorPreventsReplayingOldSelection(t *testing.T) {
	userHome := t.TempDir()
	sidecarDir := t.TempDir()
	settingsPath := filepath.Join(userHome, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(settingsPath, map[string]any{"model": "sonnet"}, 0o600); err != nil {
		t.Fatal(err)
	}
	sessionDir := filepath.Join(sidecarDir, "projects", "-tmp-project")
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	line := fmt.Sprintf(`{"timestamp":%q,"message":{"content":"Set model to gpt-5.4-mini[1m] and saved as your default for new sessions"}}`+"\n", now.Format(time.RFC3339Nano))
	if err := os.WriteFile(filepath.Join(sessionDir, "session.jsonl"), []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}

	cursor, err := syncTranscriptModelDefault(sidecarDir, userHome, time.Second, modelconfig.Default(), now.Add(-time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(settingsPath, map[string]any{"model": "opus"}, 0o600); err != nil {
		t.Fatal(err)
	}
	cursor, err = syncTranscriptModelDefault(sidecarDir, userHome, time.Second, modelconfig.Default(), cursor)
	if err != nil {
		t.Fatal(err)
	}
	settings := mustReadJSONMap(t, settingsPath)
	if settings["model"] != "opus" {
		t.Fatalf("old transcript selection replayed over newer settings: %#v", settings)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func mustReadJSONMap(t *testing.T, path string) map[string]any {
	t.Helper()
	config, err := readJSONMap(path)
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func waitForConfig(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition was not met before deadline")
}

func hasProjectKey(config map[string]any, want string) bool {
	projects, ok := config["projects"].(map[string]any)
	return ok && projects[want] != nil
}

func assertProjectKeys(t *testing.T, config map[string]any, wants ...string) {
	t.Helper()
	projects, ok := config["projects"].(map[string]any)
	if !ok {
		t.Fatalf("projects missing from config: %#v", config)
	}
	for _, want := range wants {
		if projects[want] == nil {
			t.Fatalf("project %q missing from %#v", want, projects)
		}
	}
}
