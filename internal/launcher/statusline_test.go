package launcher

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPrepareStatusLineFlagSettingsPreservesSettingsFlagWithoutStatusLine(t *testing.T) {
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)
	t.Chdir(t.TempDir())
	sidecarDir := t.TempDir()
	args := []string{"--settings", `{"theme":"dark","effortLevel":"medium"}`, "--model", "opus"}

	got, err := PrepareStatusLineFlagSettings(sidecarDir, args)
	if err != nil {
		t.Fatal(err)
	}
	settings := mustReadJSONMap(t, flagSettingsArg(t, got))
	if settings["theme"] != "dark" || settings["effortLevel"] != "medium" {
		t.Fatalf("flag settings not preserved: %#v", settings)
	}
	for _, pattern := range []string{claudodexStatuslineSourcePrefix + "*.json"} {
		matches, err := filepath.Glob(filepath.Join(sidecarDir, pattern))
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) != 0 {
			t.Fatalf("generated statusline files after no-op prepare: %#v", matches)
		}
	}
}

func TestPrepareStatusLineFlagSettingsDoesNotShadowConfiguredModel(t *testing.T) {
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)
	t.Chdir(t.TempDir())
	realClaudeDir := filepath.Join(userHome, ".claude")
	if err := os.MkdirAll(realClaudeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realClaudeDir, "settings.json"), []byte(`{"model":"sonnet"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := PrepareStatusLineFlagSettings(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"--effort", "max"}) {
		t.Fatalf("args = %#v, want only default effort", got)
	}
	realSettings := mustReadJSONMap(t, filepath.Join(realClaudeDir, "settings.json"))
	if realSettings["model"] != "sonnet" {
		t.Fatalf("real settings modified: %#v", realSettings)
	}
}

func TestPrepareStatusLineFlagSettingsAddsDefaultMaxEffortWhenUnset(t *testing.T) {
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)
	t.Chdir(t.TempDir())
	sidecarDir := t.TempDir()
	args := []string{"--settings", `{"theme":"dark"}`, "--model", "opus"}

	got, err := PrepareStatusLineFlagSettings(sidecarDir, args)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 4 || got[0] != "--effort" || got[1] != "max" || got[len(got)-2] != "--settings" {
		t.Fatalf("args = %#v, want max effort and generated settings", got)
	}
}

func TestPrepareStatusLineFlagSettingsWrapsAndPreservesUserSettingsFlag(t *testing.T) {
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)
	projectDir := t.TempDir()
	t.Chdir(projectDir)
	realClaudeDir := filepath.Join(userHome, ".claude")
	if err := os.MkdirAll(realClaudeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	userSettingsPath := filepath.Join(realClaudeDir, "settings.json")
	if err := os.WriteFile(userSettingsPath, []byte(`{
  "theme": "light",
  "statusLine": {"type": "command", "command": "user-status", "padding": 1}
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	flagSettingsPath := filepath.Join(t.TempDir(), "flag-settings.json")
	if err := os.WriteFile(flagSettingsPath, []byte(`{
  "theme": "dark",
  "env": {"FOO": "bar"},
  "statusLine": {"type": "command", "command": "flag-status", "padding": 2}
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	sidecarDir := t.TempDir()
	args := []string{"--model", "opus", "--settings", flagSettingsPath, "--verbose"}
	got, err := PrepareStatusLineFlagSettings(sidecarDir, args)
	if err != nil {
		t.Fatal(err)
	}
	if contains(got, flagSettingsPath) {
		t.Fatalf("original --settings path leaked into child args: %#v", got)
	}
	generatedSettingsPath := flagSettingsArg(t, got)
	if !strings.HasPrefix(filepath.Base(generatedSettingsPath), claudodexFlagSettingsPrefix) {
		t.Fatalf("generated settings path = %q", generatedSettingsPath)
	}
	_, generatedSourcePath := statusLineSessionPaths(sidecarDir)

	settings := mustReadJSONMap(t, generatedSettingsPath)
	if settings["theme"] != "dark" {
		t.Fatalf("theme = %#v, want flag setting", settings["theme"])
	}
	env := settings["env"].(map[string]any)
	if env["FOO"] != "bar" {
		t.Fatalf("env not preserved: %#v", env)
	}
	statusLine := settings["statusLine"].(map[string]any)
	command, _ := statusLine["command"].(string)
	if statusLine["type"] != "command" || !strings.Contains(command, StatusLineCommandName) || !strings.Contains(command, generatedSourcePath) {
		t.Fatalf("wrapped statusLine = %#v", statusLine)
	}
	if statusLine["padding"] != float64(2) {
		t.Fatalf("padding = %#v, want 2", statusLine["padding"])
	}

	source, err := readStatusLineSource(generatedSourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if source.UserSettingsPath != userSettingsPath {
		t.Fatalf("user settings path = %q, want %q", source.UserSettingsPath, userSettingsPath)
	}
	if source.ProjectSettingsPath != filepath.Join(projectDir, ".claude", "settings.json") {
		t.Fatalf("project settings path = %q", source.ProjectSettingsPath)
	}
	if !reflect.DeepEqual(source.SettingSources, []string{"userSettings", "projectSettings", "localSettings"}) {
		t.Fatalf("setting sources = %#v", source.SettingSources)
	}
	if source.FlagStatusLine["command"] != "flag-status" {
		t.Fatalf("flag statusLine source = %#v", source.FlagStatusLine)
	}
	realUserSettings := mustReadJSONMap(t, userSettingsPath)
	if realUserSettings["theme"] != "light" {
		t.Fatalf("real user settings were modified: %#v", realUserSettings)
	}
}

func TestPrepareStatusLineFlagSettingsMergesPartialFlagStatusLine(t *testing.T) {
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)
	t.Chdir(t.TempDir())
	realClaudeDir := filepath.Join(userHome, ".claude")
	if err := os.MkdirAll(realClaudeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realClaudeDir, "settings.json"), []byte(`{
  "statusLine": {"type": "command", "command": "user-status", "padding": 1}
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	sidecarDir := t.TempDir()
	got, err := PrepareStatusLineFlagSettings(sidecarDir, []string{"--settings", `{"statusLine":{"padding":3}}`})
	if err != nil {
		t.Fatal(err)
	}
	statusLine := mustReadJSONMap(t, flagSettingsArg(t, got))["statusLine"].(map[string]any)
	if statusLine["padding"] != float64(3) {
		t.Fatalf("padding = %#v, want flag override", statusLine["padding"])
	}
}

func TestPrepareStatusLineFlagSettingsRespectsSettingSources(t *testing.T) {
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)
	t.Chdir(t.TempDir())
	realClaudeDir := filepath.Join(userHome, ".claude")
	if err := os.MkdirAll(realClaudeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realClaudeDir, "settings.json"), []byte(`{
  "statusLine": {"type": "command", "command": "user-status"}
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	sidecarDir := t.TempDir()
	args := []string{"--setting-sources=", "--settings", `{"theme":"dark"}`}
	got, err := PrepareStatusLineFlagSettings(sidecarDir, args)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 4 || got[0] != "--effort" || got[1] != "max" || got[len(got)-2] != "--settings" {
		t.Fatalf("args = %#v, want max effort and generated settings", got)
	}
	settings := mustReadJSONMap(t, flagSettingsArg(t, got))
	if settings["theme"] != "dark" {
		t.Fatalf("theme = %#v, want flag setting", settings["theme"])
	}
	matches, err := filepath.Glob(filepath.Join(sidecarDir, claudodexStatuslineSourcePrefix+"*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("statusline source generated despite disabled sources: %#v", matches)
	}
}

func TestExtractFlagSettingsUsesFirstSettingsValue(t *testing.T) {
	settings, args, sawSettings, err := extractFlagSettings([]string{
		"--settings", `{"theme":"first"}`,
		"--verbose",
		"--settings", `{"theme":"second"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !sawSettings {
		t.Fatal("sawSettings = false")
	}
	if settings["theme"] != "first" {
		t.Fatalf("theme = %#v, want first --settings value", settings["theme"])
	}
	if !reflect.DeepEqual(args, []string{"--verbose"}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestRunStatusLineWrapperPatchesInputForOriginalCommand(t *testing.T) {
	if os.PathSeparator != '/' {
		t.Skip("shell command test is POSIX-only")
	}
	scriptPath := filepath.Join(t.TempDir(), "statusline-reader.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\ncat\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	sidecarDir := t.TempDir()
	sourcePath := filepath.Join(sidecarDir, claudodexStatuslineSourceName)
	if err := writeJSONFile(sourcePath, statusLineSource{
		FlagStatusLine: map[string]any{
			"type":    "command",
			"command": shellQuote(scriptPath),
		},
	}, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(claudodexStatuslineSourceEnv, sourcePath)
	t.Setenv("CLAUDODEX_CONTEXT_WINDOW", "272000")

	input := `{"context_window":{"context_window_size":1000000,"current_usage":{"input_tokens":46000,"cache_read_input_tokens":700,"cache_creation_input_tokens":0},"used_percentage":5,"remaining_percentage":95}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunStatusLineWrapper([]string{sourcePath}, strings.NewReader(input), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}

	var patched map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &patched); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, stdout.String())
	}
	contextWindow := patched["context_window"].(map[string]any)
	if contextWindow["context_window_size"] != float64(272000) {
		t.Fatalf("context_window_size = %#v", contextWindow["context_window_size"])
	}
	if contextWindow["used_percentage"] != float64(17) || contextWindow["remaining_percentage"] != float64(83) {
		t.Fatalf("percentages = used %#v remaining %#v", contextWindow["used_percentage"], contextWindow["remaining_percentage"])
	}
}

func TestPatchStatusLineInputStripsLongContextModelSuffixes(t *testing.T) {
	input := []byte(`{"model":{"id":"gpt-5.4[1m][1m]"},"workspace":["gpt-5.5[1m][1m]"],"context_window":{"context_window_size":1000000}}`)
	var patched map[string]any
	if err := json.Unmarshal(patchStatusLineInput(input, 272000), &patched); err != nil {
		t.Fatal(err)
	}
	model := patched["model"].(map[string]any)
	if model["id"] != "gpt-5.4" {
		t.Fatalf("model id = %#v", model["id"])
	}
	workspace := patched["workspace"].([]any)
	if workspace[0] != "gpt-5.5" {
		t.Fatalf("workspace model = %#v", workspace[0])
	}
}

func flagSettingsArg(t *testing.T, args []string) string {
	t.Helper()
	for i := 0; i < len(args); i++ {
		if args[i] == "--settings" && i+1 < len(args) {
			return args[i+1]
		}
	}
	t.Fatalf("missing generated --settings in %#v", args)
	return ""
}
