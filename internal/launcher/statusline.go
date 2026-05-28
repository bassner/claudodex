package launcher

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const (
	claudodexFlagSettingsPrefix     = "claudodex-settings-"
	claudodexStatuslineSourcePrefix = "claudodex-statusline-source-"
	claudodexStatuslineSourceName   = "claudodex-statusline-source.json"
	claudodexStatuslineSourceEnv    = "CLAUDODEX_STATUSLINE_SOURCE"
	claudodexStatuslineWrappedEnv   = "CLAUDODEX_STATUSLINE_WRAPPED"
	claudodexStatuslineCommandName  = "__statusline"
)

const StatusLineCommandName = claudodexStatuslineCommandName

type statusLineSource struct {
	UserSettingsPath    string         `json:"user_settings_path,omitempty"`
	ProjectSettingsPath string         `json:"project_settings_path,omitempty"`
	LocalSettingsPath   string         `json:"local_settings_path,omitempty"`
	SettingSources      []string       `json:"setting_sources,omitempty"`
	FlagStatusLine      map[string]any `json:"flag_status_line,omitempty"`
}

func IsStatusLineWrapperCommand(args []string) bool {
	return len(args) > 0 && args[0] == StatusLineCommandName
}

func PrepareStatusLineFlagSettings(sidecarDir string, args []string) ([]string, error) {
	flagSettings, argsWithoutSettings, sawFlagSettings, err := extractFlagSettings(args)
	if err != nil {
		return nil, err
	}
	userSettingsPath, err := userClaudeSettingsPath()
	if err != nil {
		return nil, err
	}
	projectSettingsPath, localSettingsPath := projectClaudeSettingsPaths()

	flagStatusLine := mapValue(flagSettings["statusLine"])
	settingSources := enabledSettingSources(args)
	activeSettings := configuredSettings(statusLineSource{
		UserSettingsPath:    userSettingsPath,
		ProjectSettingsPath: projectSettingsPath,
		LocalSettingsPath:   localSettingsPath,
		SettingSources:      settingSources,
		FlagStatusLine:      flagStatusLine,
	}, flagSettings)
	activeStatusLine := mapValue(activeSettings["statusLine"])
	needsDefaultEffort := !hasEffortArg(args) && !hasSetting(activeSettings, "effortLevel")
	needsGeneratedSettings := hasStatusLineCommand(activeStatusLine) || sawFlagSettings

	settingsPath, sourcePath := statusLineSessionPaths(sidecarDir)
	settings := cloneJSONMap(flagSettings)
	if hasStatusLineCommand(activeStatusLine) {
		source := statusLineSource{
			UserSettingsPath:    userSettingsPath,
			ProjectSettingsPath: projectSettingsPath,
			LocalSettingsPath:   localSettingsPath,
			SettingSources:      append([]string(nil), settingSources...),
			FlagStatusLine:      cloneJSONMap(flagStatusLine),
		}
		if err := writeJSONFile(sourcePath, source, 0o600); err != nil {
			return nil, err
		}
		settings["statusLine"] = wrappedStatusLine(activeStatusLine, sourcePath)
	}
	if !needsGeneratedSettings {
		return withDefaultEffortArg(append([]string(nil), args...), needsDefaultEffort), nil
	}
	if err := writeJSONFile(settingsPath, settings, 0o600); err != nil {
		return nil, err
	}
	return withDefaultEffortArg(append(argsWithoutSettings, "--settings", settingsPath), needsDefaultEffort), nil
}

func statusLineSessionPaths(sidecarDir string) (string, string) {
	suffix := strconv.Itoa(os.Getpid())
	return filepath.Join(sidecarDir, claudodexFlagSettingsPrefix+suffix+".json"), filepath.Join(sidecarDir, claudodexStatuslineSourcePrefix+suffix+".json")
}

func wrappedStatusLine(activeStatusLine map[string]any, sourcePath string) map[string]any {
	padding := statusLinePadding(activeStatusLine)
	statusLine := map[string]any{
		"type":    "command",
		"command": shellQuote(currentExecutable()) + " " + claudodexStatuslineCommandName + " " + shellQuote(sourcePath),
	}
	if padding != nil {
		statusLine["padding"] = *padding
	}
	return statusLine
}

func currentExecutable() string {
	exe, err := os.Executable()
	if err == nil && exe != "" {
		return exe
	}
	return "claudodex"
}

func extractFlagSettings(args []string) (map[string]any, []string, bool, error) {
	settings := map[string]any{}
	out := make([]string, 0, len(args))
	sawSettings := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--settings":
			if i+1 >= len(args) {
				return nil, nil, false, fmt.Errorf("missing value for --settings")
			}
			parsed, err := readSettingsArg(args[i+1])
			if err != nil {
				return nil, nil, false, err
			}
			if !sawSettings {
				settings = parsed
				sawSettings = true
			}
			i++
		case strings.HasPrefix(arg, "--settings="):
			parsed, err := readSettingsArg(strings.TrimPrefix(arg, "--settings="))
			if err != nil {
				return nil, nil, false, err
			}
			if !sawSettings {
				settings = parsed
				sawSettings = true
			}
		default:
			out = append(out, arg)
		}
	}
	return settings, out, sawSettings, nil
}

func hasEffortArg(args []string) bool {
	for _, arg := range args {
		if arg == "--effort" || strings.HasPrefix(arg, "--effort=") {
			return true
		}
	}
	return false
}

func withDefaultEffortArg(args []string, enabled bool) []string {
	if !enabled {
		return args
	}
	return append([]string{"--effort", "max"}, args...)
}

func mergeStatusLine(base, override map[string]any) map[string]any {
	if base == nil && override == nil {
		return nil
	}
	if base == nil {
		return cloneJSONMap(override)
	}
	if override == nil {
		return cloneJSONMap(base)
	}
	return overlayJSONMap(base, override)
}

func overlayJSONMap(base, override map[string]any) map[string]any {
	out := cloneJSONMap(base)
	for key, value := range override {
		baseMap := mapValue(out[key])
		overrideMap := mapValue(value)
		if baseMap != nil && overrideMap != nil {
			out[key] = overlayJSONMap(baseMap, overrideMap)
			continue
		}
		out[key] = cloneJSONValue(value)
	}
	return out
}

func readSettingsArg(value string) (map[string]any, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("empty --settings value")
	}
	if strings.HasPrefix(value, "{") {
		var settings map[string]any
		if err := json.Unmarshal([]byte(value), &settings); err != nil {
			return nil, fmt.Errorf("parse --settings JSON: %w", err)
		}
		if settings == nil {
			settings = map[string]any{}
		}
		return settings, nil
	}
	return readJSONMap(value)
}

func userClaudeSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

func projectClaudeSettingsPaths() (string, string) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", ""
	}
	return filepath.Join(cwd, ".claude", "settings.json"), filepath.Join(cwd, ".claude", "settings.local.json")
}

func enabledSettingSources(args []string) []string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--setting-sources" && i+1 < len(args):
			return parseSettingSources(args[i+1])
		case strings.HasPrefix(arg, "--setting-sources="):
			return parseSettingSources(strings.TrimPrefix(arg, "--setting-sources="))
		}
	}
	return []string{"userSettings", "projectSettings", "localSettings"}
}

func parseSettingSources(value string) []string {
	if value == "" {
		return nil
	}
	out := []string{}
	seen := map[string]struct{}{}
	for _, item := range strings.Split(value, ",") {
		var source string
		switch strings.TrimSpace(item) {
		case "user":
			source = "userSettings"
		case "project":
			source = "projectSettings"
		case "local":
			source = "localSettings"
		default:
			continue
		}
		if _, ok := seen[source]; ok {
			continue
		}
		seen[source] = struct{}{}
		out = append(out, source)
	}
	return out
}

func configuredStatusLine(source statusLineSource) map[string]any {
	return mapValue(configuredSettings(source, nil)["statusLine"])
}

func configuredSettings(source statusLineSource, flagSettings map[string]any) map[string]any {
	settings := map[string]any{}
	var statusLine map[string]any
	for _, settingSource := range source.SettingSources {
		switch settingSource {
		case "userSettings":
			settings = overlayJSONMap(settings, readSettingsMap(source.UserSettingsPath))
		case "projectSettings":
			settings = overlayJSONMap(settings, readSettingsMap(source.ProjectSettingsPath))
		case "localSettings":
			settings = overlayJSONMap(settings, readSettingsMap(source.LocalSettingsPath))
		}
	}
	if flagSettings != nil {
		settings = overlayJSONMap(settings, flagSettings)
	} else if source.FlagStatusLine != nil {
		statusLine = mapValue(settings["statusLine"])
		settings["statusLine"] = mergeStatusLine(statusLine, source.FlagStatusLine)
	}
	return settings
}

func mapValue(value any) map[string]any {
	if value == nil {
		return nil
	}
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return nil
}

func hasStatusLineCommand(statusLine map[string]any) bool {
	if statusLine == nil {
		return false
	}
	if statusLine["type"] != "command" {
		return false
	}
	command, _ := statusLine["command"].(string)
	return strings.TrimSpace(command) != ""
}

func hasSetting(settings map[string]any, key string) bool {
	_, ok := settings[key]
	return ok
}

func statusLinePadding(statusLine map[string]any) *int {
	if statusLine == nil {
		return nil
	}
	switch value := statusLine["padding"].(type) {
	case int:
		return &value
	case float64:
		padding := int(value)
		return &padding
	case json.Number:
		if parsed, err := strconv.Atoi(value.String()); err == nil {
			return &parsed
		}
	}
	return nil
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func RunStatusLineWrapper(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if stdin == nil {
		stdin = os.Stdin
	}
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	input, err := io.ReadAll(stdin)
	if err != nil {
		fmt.Fprintf(stderr, "read statusline input: %v\n", err)
		return 1
	}
	source, err := readStatusLineSource(statusLineSourcePath(args))
	if err != nil {
		fmt.Fprintf(stderr, "read statusline source: %v\n", err)
		return 1
	}
	statusLine := configuredStatusLine(source)
	if !hasStatusLineCommand(statusLine) {
		return 0
	}
	command := strings.TrimSpace(statusLine["command"].(string))
	if strings.Contains(command, claudodexStatuslineCommandName) {
		return 0
	}
	patched := patchStatusLineInput(input, statuslineContextWindow())
	return runStatusLineCommand(command, patched, stdout, stderr)
}

func statusLineSourcePath(args []string) string {
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		return args[0]
	}
	return os.Getenv(claudodexStatuslineSourceEnv)
}

func readStatusLineSource(path string) (statusLineSource, error) {
	if path == "" {
		return statusLineSource{}, errors.New("missing " + claudodexStatuslineSourceEnv)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return statusLineSource{}, err
	}
	var source statusLineSource
	if err := json.Unmarshal(data, &source); err != nil {
		return statusLineSource{}, err
	}
	return source, nil
}

func readSettingsMap(path string) map[string]any {
	if path == "" {
		return map[string]any{}
	}
	settings, err := readJSONMap(path)
	if err != nil {
		return map[string]any{}
	}
	return settings
}

func statuslineContextWindow() int64 {
	for _, key := range []string{"CLAUDODEX_CONTEXT_WINDOW", "CLAUDE_CODE_AUTO_COMPACT_WINDOW", "CLAUDE_CODE_MAX_CONTEXT_TOKENS"} {
		if parsed, err := strconv.ParseInt(strings.TrimSpace(os.Getenv(key)), 10, 64); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 0
}

func patchStatusLineInput(input []byte, contextWindow int64) []byte {
	needsContextPatch := contextWindow > 0 && bytes.Contains(input, []byte(`"context_window"`))
	needsModelPatch := bytes.Contains(input, []byte(`[1m]`))
	if !needsContextPatch && !needsModelPatch {
		return input
	}
	var root map[string]any
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	if err := decoder.Decode(&root); err != nil {
		return input
	}
	if needsContextPatch {
		if contextWindowMap := mapValue(root["context_window"]); contextWindowMap != nil {
			contextWindowMap["context_window_size"] = contextWindow
			if usage := mapValue(contextWindowMap["current_usage"]); usage != nil {
				used := jsonNumber(usage["input_tokens"]) + jsonNumber(usage["cache_read_input_tokens"]) + jsonNumber(usage["cache_creation_input_tokens"])
				if used > 0 {
					usedPercentage := math.Round(float64(used) * 100 / float64(contextWindow))
					contextWindowMap["used_percentage"] = usedPercentage
					contextWindowMap["remaining_percentage"] = 100 - usedPercentage
				}
			}
		}
	}
	if needsModelPatch {
		root = stripLongContextSuffixes(root).(map[string]any)
	}
	data, err := json.Marshal(root)
	if err != nil {
		return input
	}
	return data
}

func stripLongContextSuffixes(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			typed[key] = stripLongContextSuffixes(item)
		}
		return typed
	case []any:
		for i, item := range typed {
			typed[i] = stripLongContextSuffixes(item)
		}
		return typed
	case string:
		return modelconfig.StripLongContext(typed)
	default:
		return typed
	}
}

func jsonNumber(value any) int64 {
	switch v := value.(type) {
	case json.Number:
		parsed, _ := v.Int64()
		return parsed
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		parsed, _ := strconv.ParseInt(v, 10, 64)
		return parsed
	default:
		return 0
	}
}

func runStatusLineCommand(command string, input []byte, stdout, stderr io.Writer) int {
	cmd := statusLineShellCommand(command)
	cmd.Stdin = bytes.NewReader(input)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(), claudodexStatuslineWrappedEnv+"=1")
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode()
		}
		fmt.Fprintf(stderr, "run statusline command: %v\n", err)
		return 1
	}
	return 0
}

func statusLineShellCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd.exe", "/C", command)
	}
	return exec.Command("/bin/sh", "-c", command)
}
