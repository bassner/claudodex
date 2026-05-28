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
	}, 4321, "/tmp/claudodex-claude", "http://127.0.0.1:9999", "/tmp/ca.pem", []codex.ModelInfo{
		{Slug: "gpt-5.5", ContextWindow: 272000},
		{Slug: "gpt-5.4", ContextWindow: 300000},
		{Slug: "gpt-5.4-mini", ContextWindow: 400000},
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
		"USER_TYPE":                                "ant",
		"USE_LOCAL_OAUTH":                          "1",
		"CLAUDE_LOCAL_OAUTH_API_BASE":              "http://127.0.0.1:4321",
		"HTTP_PROXY":                               "http://user-http",
		"HTTPS_PROXY":                              "http://127.0.0.1:9999",
		"https_proxy":                              "http://127.0.0.1:9999",
		"NO_PROXY":                                 ".example.com,127.0.0.1,localhost",
		"NODE_EXTRA_CA_CERTS":                      "/tmp/ca.pem",
		"CLAUDODEX_REAL_SHELL":                     "/bin/zsh",
		"SHELL":                                    filepath.Join("/tmp/claudodex-claude", claudodexShimDirName, "zsh"),
		"CLAUDODEX_ORIGINAL_SHELL":                 "/bin/zsh",
		"CLAUDODEX_ORIGINAL_HTTP_PROXY":            "http://user-http",
		"CLAUDODEX_ORIGINAL_HTTPS_PROXY":           "http://user-https",
		"CLAUDODEX_ORIGINAL_NO_PROXY":              ".anthropic.com,.example.com",
		"CLAUDODEX_ORIGINAL_NODE_EXTRA_CA_CERTS":   "/tmp/user-ca.pem",
		"ANTHROPIC_DEFAULT_OPUS_MODEL":             "gpt-5.5",
		"ANTHROPIC_DEFAULT_SONNET_MODEL":           "gpt-5.4[1m]",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":            "gpt-5.4-mini[1m]",
		"ANTHROPIC_SMALL_FAST_MODEL":               "gpt-5.4-mini[1m]",
		"CLAUDODEX_CONTEXT_WINDOW":                 "272000",
		"CLAUDODEX_STATUSLINE_SOURCE":              filepath.Join("/tmp/claudodex-claude", claudodexStatuslineSourceName),
		"CLAUDE_CODE_AUTO_COMPACT_WINDOW":          "272000",
		"CLAUDE_CODE_MAX_CONTEXT_TOKENS":           "272000",
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
	for _, key := range []string{"ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN", "CLAUDE_CODE_OAUTH_SCOPES", "CLAUDE_CODE_SUBSCRIPTION_TYPE"} {
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
		`"defaultModel":"gpt-5.5"`,
		`"defaultModelEffortLevel":"max"`,
		`"alias":"opus"`,
		`"model":"gpt-5.5"`,
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
}

func TestBuildClaudeEnvDoesNotInventHTTPProxy(t *testing.T) {
	env := BuildClaudeEnv([]string{"PATH=/bin"}, 4321, "/tmp/claudodex-claude", "http://127.0.0.1:9999", "", nil, modelconfig.Default())
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
	if got["HTTPS_PROXY"] != "http://127.0.0.1:9999" || got["https_proxy"] != "http://127.0.0.1:9999" {
		t.Fatalf("HTTPS proxy not set: %#v", got)
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
	env := BuildClaudeEnv([]string{"PATH=/bin", "SHELL=/opt/homebrew/bin/fish"}, 4321, "/tmp/claudodex-claude", "", "", nil, modelconfig.Default())
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
	env := BuildClaudePrivacyEnv([]string{"ANTHROPIC_BASE_URL=http://old", "DISABLE_GROWTHBOOK=0"})
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
	if got["DISABLE_GROWTHBOOK"] != "1" || got["DO_NOT_TRACK"] != "1" {
		t.Fatalf("privacy flags not forced: %#v", got)
	}
}
