package launcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
)

const (
	firstPartyAnthropicBaseURL = "https://api.anthropic.com"
	localOAuthAccessToken      = "claudodex-local-oauth"
	localOAuthScopes           = "user:profile user:inference user:sessions:claude_code user:mcp_servers user:file_upload"
	localOAuthSubscriptionType = "max"
)

func BuildClaudeEnv(base []string, proxyPort int, claudeConfigDir string, anthropicUnixSocket string, httpsProxy string, caPath string, codexModels []codex.ModelInfo, modelCfg modelconfig.Config) []string {
	modelCfg = modelCfg.Normalize()
	env := envMap(base)
	captureOriginalToolEnv(env)
	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", proxyPort)
	env["ANTHROPIC_BASE_URL"] = firstPartyAnthropicBaseURL
	env["CLAUDE_CODE_API_BASE_URL"] = firstPartyAnthropicBaseURL
	env["CLAUDE_CONFIG_DIR"] = claudeConfigDir
	env["CLAUDE_SECURESTORAGE_CONFIG_DIR"] = claudeConfigDir
	env["CLAUDE_CODE_PROVIDER_MANAGED_BY_HOST"] = "1"
	env["USER_TYPE"] = "ant"
	env["USE_LOCAL_OAUTH"] = "1"
	env["CLAUDE_LOCAL_OAUTH_API_BASE"] = proxyURL
	if runtime.GOOS != "windows" {
		shimDir := filepath.Join(claudeConfigDir, claudodexShimDirName)
		env["PATH"] = prependPath(env["PATH"], shimDir)
		configureShellShimEnv(env, shimDir)
	}
	delete(env, "ANTHROPIC_AUTH_TOKEN")
	delete(env, "ANTHROPIC_API_KEY")
	delete(env, "CLAUDE_CODE_OAUTH_TOKEN")
	delete(env, "CLAUDE_CODE_OAUTH_SCOPES")
	delete(env, "CLAUDE_CODE_SUBSCRIPTION_TYPE")
	delete(env, "CLAUDE_CODE_RATE_LIMIT_TIER")
	delete(env, "CLAUDE_CODE_OAUTH_TOKEN_FILE_DESCRIPTOR")
	delete(env, "CLAUDE_CODE_API_KEY_FILE_DESCRIPTOR")
	env["CLAUDE_CODE_SKIP_FAST_MODE_ORG_CHECK"] = "1"
	if anthropicUnixSocket != "" {
		env["ANTHROPIC_UNIX_SOCKET"] = anthropicUnixSocket
		env["CLAUDE_CODE_OAUTH_TOKEN"] = localOAuthAccessToken
		clearClaudeVisibleProxyEnv(env)
	} else {
		delete(env, "ANTHROPIC_UNIX_SOCKET")
	}
	if anthropicUnixSocket == "" && httpsProxy != "" {
		env["HTTPS_PROXY"] = httpsProxy
		env["https_proxy"] = httpsProxy
		env["NO_PROXY"] = mergeNoProxy(filterNoProxy(env["NO_PROXY"]), "127.0.0.1", "localhost")
		env["no_proxy"] = mergeNoProxy(filterNoProxy(env["no_proxy"]), "127.0.0.1", "localhost")
	}
	if caPath != "" {
		env["NODE_EXTRA_CA_CERTS"] = caPath
	}
	delete(env, "ANTHROPIC_DEFAULT_FABLE_MODEL")
	delete(env, "ANTHROPIC_DEFAULT_FABLE_MODEL_NAME")
	delete(env, "ANTHROPIC_DEFAULT_FABLE_MODEL_DESCRIPTION")
	env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = modelconfig.StripLongContext(modelCfg.RuntimeModel(string(modelconfig.FamilyOpus)))
	env["ANTHROPIC_DEFAULT_OPUS_MODEL_NAME"] = modelconfig.StripLongContext(modelCfg.Opus)
	env["ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION"] = "Default Codex route"
	env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = modelconfig.StripLongContext(modelCfg.RuntimeModel(string(modelconfig.FamilySonnet)))
	env["ANTHROPIC_DEFAULT_SONNET_MODEL_NAME"] = modelconfig.StripLongContext(modelCfg.Sonnet)
	env["ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION"] = "Everyday Codex coding route"
	env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = modelconfig.StripLongContext(modelCfg.RuntimeModel(string(modelconfig.FamilyHaiku)))
	env["ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME"] = modelconfig.StripLongContext(modelCfg.Haiku)
	env["ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION"] = "Fast Codex coding route"
	env["ANTHROPIC_SMALL_FAST_MODEL"] = modelconfig.StripLongContext(modelCfg.RuntimeModel(string(modelconfig.FamilyHaiku)))
	delete(env, "CLAUDE_CODE_DISABLE_1M_CONTEXT")
	requiredContextWindow := requiredModelContextWindow(codexModels, modelCfg)
	env["CLAUDODEX_CONTEXT_WINDOW"] = strconv.FormatInt(requiredContextWindow, 10)
	env[claudodexStatuslineSourceEnv] = filepath.Join(claudeConfigDir, claudodexStatuslineSourceName)
	env["CLAUDE_CODE_AUTO_COMPACT_WINDOW"] = strconv.FormatInt(requiredContextWindow, 10)
	env["CLAUDE_CODE_MAX_CONTEXT_TOKENS"] = strconv.FormatInt(requiredContextWindow, 10)
	applyModelOverrideEnv(env, codexModels, modelCfg)
	applyRemoteControlBridgeEnv(env)
	applyPrivacyEnv(env)
	return flattenEnv(env)
}

func clearClaudeVisibleProxyEnv(env map[string]string) {
	for _, key := range []string{
		"HTTP_PROXY",
		"http_proxy",
		"HTTPS_PROXY",
		"https_proxy",
		"ALL_PROXY",
		"all_proxy",
	} {
		delete(env, key)
	}
}

func BuildClaudePrivacyEnv(base []string) []string {
	env := envMap(base)
	applyPrivacyEnv(env)
	return flattenEnv(env)
}

func WithFriendlyCustomModelOption(envList []string, runtimeModel string) []string {
	displayModel := modelconfig.StripLongContext(runtimeModel)
	if strings.TrimSpace(runtimeModel) == "" {
		return envList
	}
	env := envMap(envList)
	env["ANTHROPIC_CUSTOM_MODEL_OPTION"] = runtimeModel
	env["ANTHROPIC_CUSTOM_MODEL_OPTION_NAME"] = displayModel
	env["ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION"] = displayModel
	return flattenEnv(env)
}

func applyPrivacyEnv(env map[string]string) {
	env["CLAUDE_CODE_FORCE_FULL_LOGO"] = "1"
	env["CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"] = "1"
	env["DISABLE_TELEMETRY"] = "1"
	env["DO_NOT_TRACK"] = "1"
	env["DISABLE_GROWTHBOOK"] = "1"
}

func applyRemoteControlBridgeEnv(env map[string]string) {
	if strings.TrimSpace(env["CLAUDE_BRIDGE_BASE_URL"]) == "" {
		env["CLAUDE_BRIDGE_BASE_URL"] = firstPartyAnthropicBaseURL
	}
	if strings.TrimSpace(env["CLAUDE_BRIDGE_SESSION_INGRESS_URL"]) == "" {
		env["CLAUDE_BRIDGE_SESSION_INGRESS_URL"] = firstPartyAnthropicBaseURL
	}
	mergeGrowthBookOverride(env, "tengu_ccr_bridge", true)
	mergeGrowthBookOverride(env, "tengu_bridge_repl_v2", true)
}

type antModelOverrideConfig struct {
	DefaultModel            string             `json:"defaultModel"`
	DefaultModelEffortLevel string             `json:"defaultModelEffortLevel,omitempty"`
	AntModels               []antModelOverride `json:"antModels"`
}

type antModelOverride struct {
	Alias               string `json:"alias"`
	Model               string `json:"model"`
	Label               string `json:"label"`
	Description         string `json:"description,omitempty"`
	ContextWindow       int64  `json:"contextWindow,omitempty"`
	DefaultEffortLevel  string `json:"defaultEffortLevel,omitempty"`
	DefaultMaxTokens    int64  `json:"defaultMaxTokens,omitempty"`
	UpperMaxTokensLimit int64  `json:"upperMaxTokensLimit,omitempty"`
	AlwaysOnThinking    bool   `json:"alwaysOnThinking,omitempty"`
}

func applyModelOverrideEnv(env map[string]string, codexModels []codex.ModelInfo, modelCfg modelconfig.Config) {
	modelCfg = modelCfg.Normalize()
	override := antModelOverrideConfig{
		DefaultModel:            modelconfig.StripLongContext(modelCfg.RuntimeModel(string(modelconfig.FamilyOpus))),
		DefaultModelEffortLevel: "max",
	}
	specs := append(modelconfig.FamilyAliasSpecs(), modelconfig.ClaudeAliasSpecs(modelCfg)...)
	specs = append(specs, modelconfig.DirectRuntimeModelSpecs(modelCfg)...)
	for _, spec := range specs {
		target := modelCfg.Target(spec.Family)
		runtimeModel := modelconfig.WithLongContext(modelCfg.RuntimeModel(string(spec.Family)))
		override.AntModels = append(override.AntModels, codexAntModel(spec.ID, spec.DisplayName, runtimeModel, modelContextWindow(codexModels, target)))
	}
	mergeGrowthBookOverride(env, "tengu_ant_model_override", override)
}

func codexAntModel(alias, label, model string, contextWindow int64) antModelOverride {
	return antModelOverride{
		Alias:               alias,
		Model:               modelconfig.StripLongContext(model),
		Label:               label,
		Description:         "Routes to " + modelconfig.StripLongContext(model),
		ContextWindow:       contextWindow,
		DefaultEffortLevel:  "max",
		DefaultMaxTokens:    64_000,
		UpperMaxTokensLimit: 128_000,
	}
}

func modelContextWindow(models []codex.ModelInfo, slug string) int64 {
	if contextWindow, ok := catalogContextWindow(models, slug); ok {
		return contextWindow
	}
	return 0
}

func catalogContextWindow(models []codex.ModelInfo, slug string) (int64, bool) {
	for _, model := range models {
		if !strings.EqualFold(strings.TrimSpace(model.Slug), slug) {
			continue
		}
		if model.ContextWindow > 0 {
			return model.ContextWindow, true
		}
		if model.MaxContextWindow > 0 {
			return model.MaxContextWindow, true
		}
	}
	return 0, false
}

func requiredModelContextWindow(models []codex.ModelInfo, modelCfg modelconfig.Config) int64 {
	var min int64
	for _, slug := range modelCfg.RequiredModels() {
		contextWindow, ok := catalogContextWindow(models, slug)
		if !ok {
			continue
		}
		if min == 0 || contextWindow < min {
			min = contextWindow
		}
	}
	return min
}

func mergeGrowthBookOverride(env map[string]string, key string, value any) {
	overrides := map[string]any{}
	if raw := strings.TrimSpace(env["CLAUDE_INTERNAL_FC_OVERRIDES"]); raw != "" {
		_ = json.Unmarshal([]byte(raw), &overrides)
	}
	overrides[key] = value
	data, err := json.Marshal(overrides)
	if err != nil {
		return
	}
	env["CLAUDE_INTERNAL_FC_OVERRIDES"] = string(data)
}

var toolEnvKeys = []string{
	"SHELL",
	"HTTP_PROXY",
	"http_proxy",
	"HTTPS_PROXY",
	"https_proxy",
	"NO_PROXY",
	"no_proxy",
	"ALL_PROXY",
	"all_proxy",
	"NODE_EXTRA_CA_CERTS",
}

func captureOriginalToolEnv(env map[string]string) {
	for _, key := range toolEnvKeys {
		if value, ok := env[key]; ok {
			env[originalToolEnvKey(key)] = value
		}
	}
}

func originalToolEnvKey(key string) string {
	return "CLAUDODEX_ORIGINAL_" + key
}

func configureShellShimEnv(env map[string]string, shimDir string) {
	realShell := compatibleToolShell(strings.TrimSpace(env["SHELL"]))
	shellName := filepath.Base(realShell)
	if shellName == "" || shellName == "." || shellName == string(filepath.Separator) {
		shellName = "sh"
	}
	env["CLAUDODEX_REAL_SHELL"] = realShell
	env["SHELL"] = filepath.Join(shimDir, shellName)
}

func compatibleToolShell(userShell string) string {
	switch filepath.Base(userShell) {
	case "sh", "bash", "zsh":
		return userShell
	}
	for _, candidate := range []string{"/bin/zsh", "/bin/bash", "/bin/sh"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "/bin/sh"
}

func filterNoProxy(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	var kept []string
	for _, part := range strings.Split(value, ",") {
		item := strings.TrimSpace(part)
		if item == "" || noProxyEntryMatchesAnthropic(item) {
			continue
		}
		kept = append(kept, item)
	}
	return strings.Join(kept, ",")
}

func noProxyEntryMatchesAnthropic(item string) bool {
	normalized := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(item)), ".")
	if host, _, ok := strings.Cut(normalized, ":"); ok {
		normalized = host
	}
	if normalized == "*" {
		return true
	}
	return normalized == "anthropic.com" || normalized == "api.anthropic.com"
}

func mergeNoProxy(value string, extra ...string) string {
	seen := map[string]struct{}{}
	var merged []string
	add := func(item string) {
		item = strings.TrimSpace(item)
		if item == "" {
			return
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		merged = append(merged, item)
	}
	for _, part := range strings.Split(value, ",") {
		add(part)
	}
	for _, item := range extra {
		add(item)
	}
	return strings.Join(merged, ",")
}

func envMap(env []string) map[string]string {
	out := make(map[string]string, len(env)+16)
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if !ok || key == "" {
			continue
		}
		out[key] = value
	}
	return out
}

func flattenEnv(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for key, value := range env {
		out = append(out, key+"="+value)
	}
	return out
}

func prependPath(current string, dir string) string {
	if current == "" {
		return dir
	}
	return dir + string(os.PathListSeparator) + current
}
