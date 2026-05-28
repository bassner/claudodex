package launcher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/bassner/claudodex/internal/auth"
)

func TestRunChildKillsIgnoredSigtermOnContextCancel(t *testing.T) {
	oldTimeout := childTerminateTimeout
	childTerminateTimeout = 50 * time.Millisecond
	defer func() { childTerminateTimeout = oldTimeout }()

	ctx, cancel := context.WithCancel(context.Background())
	stdout := cancelOnReady{cancel: cancel}
	err := runChild(ctx, os.Args[0], []string{"-test.run=TestHelperProcess"}, append(os.Environ(),
		"CLAUDODEX_HELPER_PROCESS=1",
		"CLAUDODEX_HELPER_MODE=ignore-term",
	), nil, stdout, io.Discard, true)

	var exitErr ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("runChild error = %v, want ExitError", err)
	}
	if exitErr.Code != 137 {
		t.Fatalf("exit code = %d, want 137", exitErr.Code)
	}
}

func TestProcessLauncherRunsClaudeWithProxyEnvAndArgs(t *testing.T) {
	home := t.TempDir()
	saveLauncherAuth(t, home)

	binDir := t.TempDir()
	capturePath := filepath.Join(t.TempDir(), "capture.txt")
	fakeClaude := filepath.Join(binDir, "claude")
	script := `#!/bin/sh
{
	  echo "args:$#:$1:$2"
	  printf 'args_all:'
	  for arg do printf '[%s]' "$arg"; done
	  echo
	  echo "base:$ANTHROPIC_BASE_URL"
  echo "api_base:$CLAUDE_CODE_API_BASE_URL"
  echo "auth_token:$ANTHROPIC_AUTH_TOKEN"
  echo "api_key:$ANTHROPIC_API_KEY"
  echo "claude_config:$CLAUDE_CONFIG_DIR"
  echo "secure_storage_config:$CLAUDE_SECURESTORAGE_CONFIG_DIR"
  echo "provider_managed:$CLAUDE_CODE_PROVIDER_MANAGED_BY_HOST"
  echo "user_type:$USER_TYPE"
  echo "local_oauth:$USE_LOCAL_OAUTH"
  echo "local_oauth_base:$CLAUDE_LOCAL_OAUTH_API_BASE"
  echo "oauth_token:$CLAUDE_CODE_OAUTH_TOKEN"
  echo "oauth_scopes:$CLAUDE_CODE_OAUTH_SCOPES"
  echo "subscription:$CLAUDE_CODE_SUBSCRIPTION_TYPE"
  echo "https_proxy:$HTTPS_PROXY"
  echo "ca:$NODE_EXTRA_CA_CERTS"
  echo "opus:$ANTHROPIC_DEFAULT_OPUS_MODEL"
  echo "sonnet:$ANTHROPIC_DEFAULT_SONNET_MODEL"
	  echo "haiku:$ANTHROPIC_DEFAULT_HAIKU_MODEL"
		  echo "small_fast:$ANTHROPIC_SMALL_FAST_MODEL"
			  echo "max_context:$CLAUDE_CODE_DISABLE_1M_CONTEXT"
			  echo "claudodex_context_window:$CLAUDODEX_CONTEXT_WINDOW"
			  echo "claudodex_statusline_source:$CLAUDODEX_STATUSLINE_SOURCE"
			  echo "auto_compact_window:$CLAUDE_CODE_AUTO_COMPACT_WINDOW"
		  echo "context_tokens:$CLAUDE_CODE_MAX_CONTEXT_TOKENS"
		  echo "model_capabilities:$(test -f "$CLAUDE_CONFIG_DIR/cache/model-capabilities.json" && echo yes || echo no)"
		  echo "fc_overrides:$CLAUDE_INTERNAL_FC_OVERRIDES"
		  echo "custom_model_option:$ANTHROPIC_CUSTOM_MODEL_OPTION"
		  echo "custom_model_option_name:$ANTHROPIC_CUSTOM_MODEL_OPTION_NAME"
		  echo "nonessential:$CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"
  echo "telemetry:$DISABLE_TELEMETRY"
  echo "dnt:$DO_NOT_TRACK"
  echo "growthbook:$DISABLE_GROWTHBOOK"
  echo "path:$PATH"
} > "$CLAUDODEX_CAPTURE"
`
	if err := os.WriteFile(fakeClaude, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "leak")
	t.Setenv("ANTHROPIC_API_KEY", "leak")
	t.Setenv("DISABLE_TELEMETRY", "0")
	t.Setenv("DISABLE_GROWTHBOOK", "0")
	t.Setenv("CLAUDODEX_CAPTURE", capturePath)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/codex/models" {
			t.Fatalf("model metadata path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("client_version"); got != "1.2.3" {
			t.Fatalf("client_version = %q", got)
		}
		_, _ = io.WriteString(w, `{"models":[{"slug":"gpt-5.5","context_window":272000},{"slug":"gpt-5.4","context_window":300000},{"slug":"gpt-5.4-mini","context_window":400000}]}`)
	}))
	defer upstream.Close()

	err := (ProcessLauncher{}).Launch(context.Background(), []string{"--model", "claude-sonnet-4-6"}, Config{
		Version:      "1.2.3",
		Home:         home,
		CodexBaseURL: upstream.URL,
		HTTPClient:   upstream.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatal(err)
	}
	capture := string(data)
	for _, want := range []string{
		"args:4:--effort:max",
		"args_all:[--effort][max][--model][gpt-5.4[1m]]",
		"auth_token:",
		"api_key:",
		"claude_config:" + filepath.Join(home, ".claudodex", "claude-config"),
		"secure_storage_config:" + filepath.Join(home, ".claudodex", "claude-config"),
		"provider_managed:1",
		"user_type:ant",
		"local_oauth:1",
		"oauth_token:",
		"oauth_scopes:",
		"subscription:",
		"opus:gpt-5.5",
		"sonnet:gpt-5.4[1m]",
		"haiku:gpt-5.4-mini[1m]",
		"small_fast:gpt-5.4-mini[1m]",
		"max_context:",
		"claudodex_context_window:272000",
		"claudodex_statusline_source:" + filepath.Join(home, ".claudodex", "claude-config", claudodexStatuslineSourceName),
		"auto_compact_window:272000",
		"context_tokens:272000",
		"model_capabilities:yes",
		`"alias":"opus"`,
		`"defaultModel":"gpt-5.5"`,
		`"defaultModelEffortLevel":"max"`,
		`"model":"gpt-5.5"`,
		`"defaultEffortLevel":"max"`,
		`"contextWindow":272000`,
		`"alias":"claude-sonnet-4-6"`,
		`"contextWindow":300000`,
		`"alias":"haiku"`,
		`"contextWindow":400000`,
		"custom_model_option:gpt-5.4[1m]",
		"custom_model_option_name:gpt-5.4",
		"nonessential:1",
		"telemetry:1",
		"dnt:1",
		"growthbook:1",
	} {
		if !strings.Contains(capture, want) {
			t.Fatalf("capture missing %q:\n%s", want, capture)
		}
	}
	if !strings.Contains(capture, "base:"+firstPartyAnthropicBaseURL) {
		t.Fatalf("capture missing first-party Anthropic base URL:\n%s", capture)
	}
	if !strings.Contains(capture, "api_base:"+firstPartyAnthropicBaseURL) {
		t.Fatalf("capture missing first-party Claude Code API base URL:\n%s", capture)
	}
	if !strings.Contains(capture, "local_oauth_base:http://127.0.0.1:") {
		t.Fatalf("capture missing local OAuth base URL:\n%s", capture)
	}
	if !strings.Contains(capture, "https_proxy:http://127.0.0.1:") {
		t.Fatalf("capture missing HTTPS proxy:\n%s", capture)
	}
	if !strings.Contains(capture, "ca:/") {
		t.Fatalf("capture missing local CA path:\n%s", capture)
	}
	if strings.Contains(capture, "auth_token:leak") || strings.Contains(capture, "api_key:leak") {
		t.Fatalf("external Anthropic auth leaked into Claude env:\n%s", capture)
	}
	capabilities := mustReadJSONMap(t, filepath.Join(home, ".claudodex", "claude-config", "cache", claudeModelCapabilitiesFileName))
	models := capabilities["models"].([]any)
	if len(models) == 0 || models[0].(map[string]any)["id"] != "claude-sonnet-4-6" {
		t.Fatalf("model capabilities not sorted longest-first: %#v", capabilities)
	}
	globalConfig := mustReadJSONMap(t, filepath.Join(home, ".claudodex", "claude-config", claudeGlobalConfigName))
	if got := globalConfig["clientDataCache"].(map[string]any)["kelp_forest_sonnet"]; got != "300000" {
		t.Fatalf("kelp_forest_sonnet = %#v, want 300000", got)
	}
	if runtime.GOOS == "darwin" {
		wantPrefix := "path:" + filepath.Join(home, ".claudodex", "claude-config", claudodexShimDirName) + string(os.PathListSeparator)
		if !strings.Contains(capture, wantPrefix) {
			t.Fatalf("capture missing PATH shim prefix %q:\n%s", wantPrefix, capture)
		}
	}
}

type cancelOnReady struct {
	cancel func()
}

func (w cancelOnReady) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte("ready")) {
		w.cancel()
	}
	return len(p), nil
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("CLAUDODEX_HELPER_PROCESS") != "1" {
		return
	}
	switch os.Getenv("CLAUDODEX_HELPER_MODE") {
	case "ignore-term":
		signalIgnore(syscall.SIGTERM)
		fmt.Println("ready")
		for {
			time.Sleep(time.Hour)
		}
	default:
		os.Exit(2)
	}
}

var signalIgnore = signalIgnoreFunc

func signalIgnoreFunc(sig os.Signal) {
	signal.Ignore(sig)
}

func saveLauncherAuth(t *testing.T, home string) {
	t.Helper()
	if err := auth.NewStore(home).Save(auth.File{
		AuthMode: "chatgpt",
		Issuer:   auth.Issuer,
		ClientID: auth.ClientID,
		Tokens: auth.Tokens{
			AccessToken:  "access-1",
			RefreshToken: "refresh-1",
			AccountID:    "acc_123",
		},
	}); err != nil {
		t.Fatal(err)
	}
}
