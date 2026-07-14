package launcher

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
)

func TestBuildClaudeEnv(t *testing.T) {
	env := BuildClaudeEnv([]string{
		"PATH=/bin",
		"SHELL=/bin/zsh",
		"DISABLE_TELEMETRY=0",
		"HTTP_PROXY=http://user-http",
		"HTTPS_PROXY=http://user-https",
		"NO_PROXY=.anthropic.com,.example.com",
		"NODE_EXTRA_CA_CERTS=/tmp/user-ca.pem",
		"ANTHROPIC_AUTH_TOKEN=leak",
		"ANTHROPIC_API_KEY=leak",
		"CLAUDE_CODE_OAUTH_TOKEN=leak",
		"CLAUDE_CODE_DISABLE_AGENT_VIEW=0",
		"ENABLE_TOOL_SEARCH=true",
		"CLAUDODEX_PRESERVE_ME=yes",
		"ANTHROPIC_DEFAULT_FABLE_MODEL=leak",
		"ANTHROPIC_DEFAULT_FABLE_MODEL_NAME=leak",
		"ANTHROPIC_DEFAULT_FABLE_MODEL_DESCRIPTION=leak",
	}, 4321, "/tmp/claudodex-claude", "/tmp/claudodex-api.sock", "http://127.0.0.1:9999", "/tmp/ca.pem", []codex.ModelInfo{
		{Slug: "gpt-5.6-sol", ContextWindow: 272000, EffectiveContextWindowPercent: 95},
		{Slug: "gpt-5.6-terra", ContextWindow: 300000, EffectiveContextWindowPercent: 90},
		{Slug: "gpt-5.6-luna", ContextWindow: 400000, EffectiveContextWindowPercent: 80},
	}, modelconfig.Default())
	got := map[string]string{}
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			got[key] = value
		}
	}

	want := map[string]string{
		"ANTHROPIC_BASE_URL":                       firstPartyAnthropicBaseURL,
		"CLAUDE_CODE_API_BASE_URL":                 firstPartyAnthropicBaseURL,
		"CLAUDE_CONFIG_DIR":                        "/tmp/claudodex-claude",
		"CLAUDE_SECURESTORAGE_CONFIG_DIR":          "/tmp/claudodex-claude",
		"CLAUDE_CODE_PROVIDER_MANAGED_BY_HOST":     "1",
		"ENABLE_TOOL_SEARCH":                       "false",
		"CLAUDODEX_PRESERVE_ME":                    "yes",
		"USER_TYPE":                                "ant",
		"USE_LOCAL_OAUTH":                          "1",
		"CLAUDE_LOCAL_OAUTH_API_BASE":              "http://127.0.0.1:4321",
		"ANTHROPIC_UNIX_SOCKET":                    "/tmp/claudodex-api.sock",
		"CLAUDE_CODE_OAUTH_TOKEN":                  localOAuthAccessToken,
		"CLAUDE_CODE_SKIP_FAST_MODE_ORG_CHECK":     "1",
		"CLAUDE_BRIDGE_BASE_URL":                   firstPartyAnthropicBaseURL,
		"CLAUDE_BRIDGE_SESSION_INGRESS_URL":        firstPartyAnthropicBaseURL,
		"NO_PROXY":                                 ".anthropic.com,.example.com",
		"NODE_EXTRA_CA_CERTS":                      "/tmp/ca.pem",
		"CLAUDODEX_REAL_SHELL":                     "/bin/zsh",
		"SHELL":                                    filepath.Join("/tmp/claudodex-claude", claudodexShimDirName, "zsh"),
		"CLAUDODEX_ORIGINAL_SHELL":                 "/bin/zsh",
		"CLAUDODEX_ORIGINAL_HTTP_PROXY":            "http://user-http",
		"CLAUDODEX_ORIGINAL_HTTPS_PROXY":           "http://user-https",
		"CLAUDODEX_ORIGINAL_NO_PROXY":              ".anthropic.com,.example.com",
		"CLAUDODEX_ORIGINAL_NODE_EXTRA_CA_CERTS":   "/tmp/user-ca.pem",
		"ANTHROPIC_DEFAULT_OPUS_MODEL":             "gpt-5.6-sol",
		"ANTHROPIC_DEFAULT_OPUS_MODEL_NAME":        "gpt-5.6-sol",
		"ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION": "Default Codex route",
		"ANTHROPIC_DEFAULT_SONNET_MODEL":           "gpt-5.6-terra",
		"ANTHROPIC_DEFAULT_SONNET_MODEL_NAME":      "gpt-5.6-terra",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":            "gpt-5.6-luna",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME":       "gpt-5.6-luna",
		"ANTHROPIC_SMALL_FAST_MODEL":               "gpt-5.6-luna",
		"CLAUDODEX_CONTEXT_WINDOW":                 "272000",
		"CLAUDODEX_STATUSLINE_SOURCE":              filepath.Join("/tmp/claudodex-claude", claudodexStatuslineSourceName),
		"CLAUDE_CODE_AUTO_COMPACT_WINDOW":          "208000",
		"CLAUDE_CODE_MAX_CONTEXT_TOKENS":           "272000",
		"CLAUDE_CODE_FORCE_FULL_LOGO":              "1",
		"CLAUDE_CODE_DISABLE_AGENT_VIEW":           "1",
		"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
		"DISABLE_TELEMETRY":                        "1",
		"DO_NOT_TRACK":                             "1",
		"DISABLE_GROWTHBOOK":                       "1",
	}
	for key, value := range want {
		if got[key] != value {
			t.Fatalf("%s = %q, want %q", key, got[key], value)
		}
	}
	for _, key := range []string{"ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_API_KEY", "ANTHROPIC_DEFAULT_FABLE_MODEL", "ANTHROPIC_DEFAULT_FABLE_MODEL_NAME", "ANTHROPIC_DEFAULT_FABLE_MODEL_DESCRIPTION", "CLAUDE_CODE_OAUTH_SCOPES", "CLAUDE_CODE_OAUTH_REFRESH_TOKEN", "CLAUDE_CODE_SUBSCRIPTION_TYPE", "CLAUDE_CODE_RATE_LIMIT_TIER", "HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy"} {
		if got[key] != "" {
			t.Fatalf("%s leaked into Claude env: %q", key, got[key])
		}
	}
	if runtime.GOOS != "windows" {
		wantPrefix := filepath.Join("/tmp/claudodex-claude", claudodexShimDirName) + ":"
		if !strings.HasPrefix(got["PATH"], wantPrefix) {
			t.Fatalf("PATH = %q, want prefix %q", got["PATH"], wantPrefix)
		}
	}
	if got["CLAUDE_CODE_DISABLE_1M_CONTEXT"] != "" {
		t.Fatalf("CLAUDE_CODE_DISABLE_1M_CONTEXT = %q, want unset", got["CLAUDE_CODE_DISABLE_1M_CONTEXT"])
	}
	for _, want := range []string{
		`"tengu_ant_model_override"`,
		`"tengu_ccr_bridge":true`,
		`"tengu_bridge_repl_v2":true`,
		`"defaultModel":"gpt-5.6-sol"`,
		`"defaultModelEffortLevel":"max"`,
		`"alias":"opus"`,
		`"model":"gpt-5.6-sol"`,
		`"defaultEffortLevel":"max"`,
		`"contextWindow":272000`,
		`"alias":"claude-sonnet-4-6"`,
		`"contextWindow":300000`,
		`"alias":"haiku"`,
		`"contextWindow":400000`,
	} {
		if !strings.Contains(got["CLAUDE_INTERNAL_FC_OVERRIDES"], want) {
			t.Fatalf("CLAUDE_INTERNAL_FC_OVERRIDES missing %s:\n%s", want, got["CLAUDE_INTERNAL_FC_OVERRIDES"])
		}
	}
	if strings.Contains(got["CLAUDE_INTERNAL_FC_OVERRIDES"], `"model":"gpt-5.6-sol[1m]"`) {
		t.Fatalf("CLAUDE_INTERNAL_FC_OVERRIDES should strip [1m] from ant model backend ids:\n%s", got["CLAUDE_INTERNAL_FC_OVERRIDES"])
	}
}

func TestRequiredModelAutoCompactWindow(t *testing.T) {
	configuredModels := modelconfig.Config{Opus: "custom-opus", Sonnet: "custom-sonnet", Haiku: "custom-haiku"}
	tests := []struct {
		name   string
		models []codex.ModelInfo
		want   int64
		ok     bool
	}{
		{
			name: "minimum uses each models live context and percentage",
			models: []codex.ModelInfo{
				{Slug: "custom-opus", ContextWindow: 300_000, EffectiveContextWindowPercent: 90},
				{Slug: "custom-sonnet", ContextWindow: 250_000, EffectiveContextWindowPercent: 100},
				{Slug: "custom-haiku", ContextWindow: 400_000, EffectiveContextWindowPercent: 50},
			},
			want: 186_000,
			ok:   true,
		},
		{
			name: "changed live context and percentage change threshold",
			models: []codex.ModelInfo{
				{Slug: "custom-opus", ContextWindow: 500_000, EffectiveContextWindowPercent: 80},
				{Slug: "custom-sonnet", ContextWindow: 450_000, EffectiveContextWindowPercent: 70},
				{Slug: "custom-haiku", ContextWindow: 600_000, EffectiveContextWindowPercent: 95},
			},
			want: 315_000,
			ok:   true,
		},
		{
			name: "max context fallback",
			models: []codex.ModelInfo{
				{Slug: "custom-opus", MaxContextWindow: 300_000, EffectiveContextWindowPercent: 95},
				{Slug: "custom-sonnet", MaxContextWindow: 400_000, EffectiveContextWindowPercent: 95},
				{Slug: "custom-haiku", MaxContextWindow: 500_000, EffectiveContextWindowPercent: 95},
			},
			want: 236_000,
			ok:   true,
		},
		{
			name: "catalog auto compact limit is an additional ceiling",
			models: []codex.ModelInfo{
				{Slug: "custom-opus", ContextWindow: 1_000_000, EffectiveContextWindowPercent: 95, AutoCompactTokenLimit: 900_000},
				{Slug: "custom-sonnet", ContextWindow: 1_000_000, EffectiveContextWindowPercent: 95},
				{Slug: "custom-haiku", ContextWindow: 1_000_000, EffectiveContextWindowPercent: 95},
			},
			want: 900_000,
			ok:   true,
		},
		{
			name: "missing required model",
			models: []codex.ModelInfo{
				{Slug: "custom-opus", ContextWindow: 300_000, EffectiveContextWindowPercent: 95},
				{Slug: "custom-sonnet", ContextWindow: 300_000, EffectiveContextWindowPercent: 95},
			},
		},
		{
			name: "omitted effective percentage uses Codex default",
			models: []codex.ModelInfo{
				{Slug: "custom-opus", ContextWindow: 300_000, EffectiveContextWindowPercent: 95},
				{Slug: "custom-sonnet", ContextWindow: 300_000},
				{Slug: "custom-haiku", ContextWindow: 300_000, EffectiveContextWindowPercent: 95},
			},
			want: 236_000,
			ok:   true,
		},
		{
			name: "invalid effective percentage",
			models: []codex.ModelInfo{
				{Slug: "custom-opus", ContextWindow: 300_000, EffectiveContextWindowPercent: 95},
				{Slug: "custom-sonnet", ContextWindow: 300_000, EffectiveContextWindowPercent: 101},
				{Slug: "custom-haiku", ContextWindow: 300_000, EffectiveContextWindowPercent: 95},
			},
		},
		{
			name: "invalid negative effective percentage",
			models: []codex.ModelInfo{
				{Slug: "custom-opus", ContextWindow: 300_000, EffectiveContextWindowPercent: 95},
				{Slug: "custom-sonnet", ContextWindow: 300_000, EffectiveContextWindowPercent: -1},
				{Slug: "custom-haiku", ContextWindow: 300_000, EffectiveContextWindowPercent: 95},
			},
		},
		{
			name: "invalid negative auto compact limit",
			models: []codex.ModelInfo{
				{Slug: "custom-opus", ContextWindow: 300_000, EffectiveContextWindowPercent: 95},
				{Slug: "custom-sonnet", ContextWindow: 300_000, EffectiveContextWindowPercent: 95, AutoCompactTokenLimit: -1},
				{Slug: "custom-haiku", ContextWindow: 300_000, EffectiveContextWindowPercent: 95},
			},
		},
		{
			name: "context too small for advertised default output budget",
			models: []codex.ModelInfo{
				{Slug: "custom-opus", ContextWindow: 64_000, EffectiveContextWindowPercent: 95},
				{Slug: "custom-sonnet", ContextWindow: 300_000, EffectiveContextWindowPercent: 95},
				{Slug: "custom-haiku", ContextWindow: 300_000, EffectiveContextWindowPercent: 95},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := requiredModelAutoCompactWindow(test.models, configuredModels)
			if got != test.want || ok != test.ok {
				t.Fatalf("requiredModelAutoCompactWindow() = (%d, %t), want (%d, %t)", got, ok, test.want, test.ok)
			}
		})
	}
}

func TestBuildClaudeEnvDropsUnsafeInheritedAutoCompactWindow(t *testing.T) {
	env := BuildClaudeEnv([]string{"CLAUDE_CODE_AUTO_COMPACT_WINDOW=999999"}, 4321, "/tmp/claudodex-claude", "", "", "", []codex.ModelInfo{
		{Slug: "gpt-5.6-sol", ContextWindow: 272_000, EffectiveContextWindowPercent: 95},
		{Slug: "gpt-5.6-terra", ContextWindow: 272_000, EffectiveContextWindowPercent: 101},
		{Slug: "gpt-5.6-luna", ContextWindow: 272_000, EffectiveContextWindowPercent: 95},
	}, modelconfig.Default())

	for _, item := range env {
		if strings.HasPrefix(item, "CLAUDE_CODE_AUTO_COMPACT_WINDOW=") {
			t.Fatalf("unsafe inherited auto-compact threshold survived metadata validation: %q", item)
		}
	}
}

func TestBuildClaudeEnvFallbackProxyDoesNotInventHTTPProxy(t *testing.T) {
	env := BuildClaudeEnv([]string{"PATH=/bin"}, 4321, "/tmp/claudodex-claude", "", "http://127.0.0.1:9999", "", nil, modelconfig.Default())
	got := map[string]string{}
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			got[key] = value
		}
	}
	if got["HTTP_PROXY"] != "" || got["http_proxy"] != "" {
		t.Fatalf("HTTP proxy should not be set for Claude process: %#v", got)
	}
	if got["ENABLE_TOOL_SEARCH"] != "false" {
		t.Fatalf("ENABLE_TOOL_SEARCH = %q, want false", got["ENABLE_TOOL_SEARCH"])
	}
	if got["HTTPS_PROXY"] != "http://127.0.0.1:9999" || got["https_proxy"] != "http://127.0.0.1:9999" {
		t.Fatalf("HTTPS proxy not set: %#v", got)
	}
}

func TestBuildClaudeEnvUnixSocketHidesProxyFromClaude(t *testing.T) {
	env := BuildClaudeEnv([]string{
		"PATH=/bin",
		"HTTP_PROXY=http://user-http",
		"https_proxy=http://user-https",
		"ALL_PROXY=socks5://user-all",
	}, 4321, "/tmp/claudodex-claude", "/tmp/claudodex-api.sock", "http://127.0.0.1:9999", "/tmp/ca.pem", nil, modelconfig.Default())
	got := map[string]string{}
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			got[key] = value
		}
	}
	for _, key := range []string{"HTTP_PROXY", "http_proxy", "HTTPS_PROXY", "https_proxy", "ALL_PROXY", "all_proxy"} {
		if got[key] != "" {
			t.Fatalf("%s visible to Claude in unix-socket mode: %#v", key, got)
		}
	}
	if got["ANTHROPIC_UNIX_SOCKET"] != "/tmp/claudodex-api.sock" {
		t.Fatalf("ANTHROPIC_UNIX_SOCKET = %q", got["ANTHROPIC_UNIX_SOCKET"])
	}
	if got["CLAUDE_CODE_OAUTH_TOKEN"] != localOAuthAccessToken {
		t.Fatalf("CLAUDE_CODE_OAUTH_TOKEN = %q", got["CLAUDE_CODE_OAUTH_TOKEN"])
	}
	if got["CLAUDODEX_ORIGINAL_HTTP_PROXY"] != "http://user-http" || got["CLAUDODEX_ORIGINAL_https_proxy"] != "http://user-https" || got["CLAUDODEX_ORIGINAL_ALL_PROXY"] != "socks5://user-all" {
		t.Fatalf("original tool proxy env not preserved: %#v", got)
	}
}

func TestWithFriendlyCustomModelOptionLabelsRuntimeModel(t *testing.T) {
	env := WithFriendlyCustomModelOption([]string{"PATH=/bin"}, "gpt-5.4[1m]")
	got := map[string]string{}
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			got[key] = value
		}
	}
	if got["ANTHROPIC_CUSTOM_MODEL_OPTION"] != "gpt-5.4[1m]" {
		t.Fatalf("custom option = %q", got["ANTHROPIC_CUSTOM_MODEL_OPTION"])
	}
	if got["ANTHROPIC_CUSTOM_MODEL_OPTION_NAME"] != "gpt-5.4" || got["ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION"] != "gpt-5.4" {
		t.Fatalf("custom option labels = %#v", got)
	}
}

func TestBuildClaudeEnvAvoidsFishForToolShell(t *testing.T) {
	env := BuildClaudeEnv([]string{"PATH=/bin", "SHELL=/opt/homebrew/bin/fish"}, 4321, "/tmp/claudodex-claude", "", "", "", nil, modelconfig.Default())
	got := map[string]string{}
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			got[key] = value
		}
	}
	if strings.HasSuffix(got["SHELL"], "/fish") || strings.HasSuffix(got["CLAUDODEX_REAL_SHELL"], "/fish") {
		t.Fatalf("fish shell leaked into Claude tool shell env: %#v", got)
	}
	if got["SHELL"] == "" || got["CLAUDODEX_REAL_SHELL"] == "" {
		t.Fatalf("tool shell env missing: %#v", got)
	}
}

func TestBuildClaudePrivacyEnvDoesNotSetProxy(t *testing.T) {
	env := BuildClaudePrivacyEnv([]string{"ANTHROPIC_BASE_URL=http://old", "CLAUDE_CODE_DISABLE_AGENT_VIEW=0", "DISABLE_GROWTHBOOK=0"})
	got := map[string]string{}
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			got[key] = value
		}
	}

	if got["ANTHROPIC_BASE_URL"] != "http://old" {
		t.Fatalf("ANTHROPIC_BASE_URL = %q", got["ANTHROPIC_BASE_URL"])
	}
	if got["CLAUDE_CODE_FORCE_FULL_LOGO"] != "1" || got["CLAUDE_CODE_DISABLE_AGENT_VIEW"] != "1" || got["DISABLE_GROWTHBOOK"] != "1" || got["DO_NOT_TRACK"] != "1" {
		t.Fatalf("privacy flags not forced: %#v", got)
	}
}
