package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bassner/claudodex/internal/auth"
	"github.com/bassner/claudodex/internal/launcher"
	"github.com/bassner/claudodex/internal/modelconfig"
)

type fakeLauncher struct {
	args []string
	cfg  launcher.Config
}

func (f *fakeLauncher) Launch(_ context.Context, args []string, cfg launcher.Config) error {
	f.args = append([]string(nil), args...)
	f.cfg = cfg
	return nil
}

func TestVersionCommand(t *testing.T) {
	var out bytes.Buffer
	code := Run(context.Background(), Config{Home: t.TempDir(), Version: "1.2.3", Stdout: &out}, []string{"clx:version"})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if out.String() != "claudodex 1.2.3\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestPromptClxPassesThrough(t *testing.T) {
	launcher := &fakeLauncher{}
	home := t.TempDir()
	code := Run(context.Background(), Config{Launcher: launcher, Home: home}, []string{"-p", "clx:doctor"})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if len(launcher.args) != 2 || launcher.args[0] != "-p" || launcher.args[1] != "clx:doctor" {
		t.Fatalf("args = %#v", launcher.args)
	}
	if launcher.cfg.Home != home {
		t.Fatalf("launcher home = %q, want %q", launcher.cfg.Home, home)
	}
}

func TestForcePassThroughDropsSeparator(t *testing.T) {
	launcher := &fakeLauncher{}
	code := Run(context.Background(), Config{Home: t.TempDir(), Launcher: launcher}, []string{"--", "clx:doctor"})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if len(launcher.args) != 1 || launcher.args[0] != "clx:doctor" {
		t.Fatalf("args = %#v", launcher.args)
	}
}

func TestStartupModelFlagsPassConfiguredTargetsToLauncher(t *testing.T) {
	launcher := &fakeLauncher{}
	code := Run(context.Background(), Config{Home: t.TempDir(), Launcher: launcher}, []string{
		"--claudodex-opus-model", "gpt-opus-next",
		"--claudodex-models=sonnet=gpt-sonnet-next,haiku=gpt-haiku-next",
		"--model", "sonnet",
	})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if launcher.cfg.Models != (modelconfig.Config{Opus: "gpt-opus-next", Sonnet: "gpt-sonnet-next", Haiku: "gpt-haiku-next", Fable: modelconfig.DefaultFableModel}) {
		t.Fatalf("models = %#v", launcher.cfg.Models)
	}
	wantPins := modelconfig.PinOpus | modelconfig.PinSonnet | modelconfig.PinHaiku
	if launcher.cfg.ModelPins != wantPins {
		t.Fatalf("model pins = %04b, want %04b", launcher.cfg.ModelPins, wantPins)
	}
	if strings.Join(launcher.args, " ") != "--model sonnet" {
		t.Fatalf("args = %#v", launcher.args)
	}
}

func TestStartupModelFlagEqualToCompiledDefaultIsPinned(t *testing.T) {
	launcher := &fakeLauncher{}
	code := Run(context.Background(), Config{Home: t.TempDir(), Launcher: launcher}, []string{
		"--claudodex-opus-model", modelconfig.DefaultOpusModel,
	})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if launcher.cfg.Models.Opus != modelconfig.DefaultOpusModel {
		t.Fatalf("opus model = %q, want %q", launcher.cfg.Models.Opus, modelconfig.DefaultOpusModel)
	}
	if launcher.cfg.ModelPins != modelconfig.PinOpus {
		t.Fatalf("model pins = %04b, want %04b", launcher.cfg.ModelPins, modelconfig.PinOpus)
	}
}

func TestCallerModelConfigOverridesArePinned(t *testing.T) {
	launcher := &fakeLauncher{}
	callerModels := modelconfig.Config{Sonnet: "gpt-6.2-terra"}
	code := Run(context.Background(), Config{Home: t.TempDir(), Launcher: launcher, Models: callerModels}, []string{"--model", "sonnet"})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if launcher.cfg.Models.Sonnet != callerModels.Sonnet {
		t.Fatalf("sonnet model = %q, want caller override %q", launcher.cfg.Models.Sonnet, callerModels.Sonnet)
	}
	if launcher.cfg.ModelPins != modelconfig.PinSonnet {
		t.Fatalf("model pins = %04b, want %04b", launcher.cfg.ModelPins, modelconfig.PinSonnet)
	}
}

func TestServeRequiresAuth(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(context.Background(), Config{Home: t.TempDir(), Stderr: &stderr}, []string{"clx:serve"})
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "login required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestServeFetchesDynamicModelMetadata(t *testing.T) {
	home := t.TempDir()
	saveAppAuth(t, home)
	var modelRequests atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/codex/models" {
			t.Fatalf("path = %q, want /codex/models", r.URL.Path)
		}
		modelRequests.Add(1)
		if got := r.Header.Get("authorization"); got != "Bearer access-1" {
			t.Fatalf("authorization = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{
				appCatalogEntry("gpt-6.1-sol", 111000),
				appCatalogEntry("gpt-6-astra", 111000),
				appCatalogEntry("gpt-6.1-terra", 222000),
				appCatalogEntry("gpt-6-luna", 333000),
			},
		})
	}))
	defer upstream.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(ctx, Config{
		Home:         home,
		Stdout:       cancelAfterWrite{w: &stdout, cancel: cancel},
		Stderr:       &stderr,
		CodexBaseURL: upstream.URL,
		HTTPClient:   upstream.Client(),
	}, []string{"clx:serve"})
	if code != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
	if modelRequests.Load() != 1 {
		t.Fatalf("model metadata requests = %d, want 1", modelRequests.Load())
	}
	if !strings.HasPrefix(stdout.String(), "http://127.0.0.1:") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestUsageCommandFetchesCodexUsage(t *testing.T) {
	home := t.TempDir()
	saveAppAuth(t, home)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wham/usage" {
			t.Fatalf("path = %q, want /wham/usage", r.URL.Path)
		}
		if got := r.Header.Get("authorization"); got != "Bearer access-1" {
			t.Fatalf("authorization = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"rate_limit": map[string]any{
				"primary_window": map[string]any{
					"limit_window_seconds": 18000,
					"used_percent":         13,
					"reset_at":             1770000000,
				},
				"secondary_window": map[string]any{
					"limit_window_seconds": 604800,
					"used_percent":         22,
					"reset_at":             1770100000,
				},
			},
			"additional_rate_limits": []any{
				map[string]any{
					"limit_name":      "codex_fable_weekly",
					"metered_feature": "codex_fable",
					"rate_limit": map[string]any{
						"primary_window": map[string]any{
							"limit_window_seconds": 604800,
							"used_percent":         44,
						},
					},
				},
			},
		})
	}))
	defer upstream.Close()

	var stdout bytes.Buffer
	code := Run(context.Background(), Config{Home: home, Stdout: &stdout, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client()}, []string{"clx:usage"})
	if code != 0 {
		t.Fatalf("code = %d, stdout = %q", code, stdout.String())
	}
	for _, want := range []string{
		"Codex usage",
		"five_hour: 13%",
		"seven_day: 22%",
		"service_tier: standard",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	for _, unwanted := range []string{"opus", "sonnet", "fable", "haiku"} {
		if strings.Contains(stdout.String(), unwanted) {
			t.Fatalf("stdout should not expose Claude family %q:\n%s", unwanted, stdout.String())
		}
	}
}

func TestUsageCommandHandlesMissingFiveHourWindow(t *testing.T) {
	home := t.TempDir()
	saveAppAuth(t, home)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wham/usage" {
			t.Fatalf("path = %q, want /wham/usage", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"rate_limit": map[string]any{
				"secondary_window": map[string]any{
					"limit_window_seconds": 604800,
					"used_percent":         22,
				},
			},
		})
	}))
	defer upstream.Close()

	var stdout bytes.Buffer
	code := Run(context.Background(), Config{Home: home, Stdout: &stdout, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client()}, []string{"clx:usage"})
	if code != 0 {
		t.Fatalf("code = %d, stdout = %q", code, stdout.String())
	}
	for _, want := range []string{"five_hour: unavailable", "seven_day: 22%"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestUsageCommandRefreshesAndRetriesOnceOn401(t *testing.T) {
	home := t.TempDir()
	saveAppAuth(t, home)
	var usageAttempts int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wham/usage":
			usageAttempts++
			if usageAttempts == 1 {
				http.Error(w, `{"error":{"message":"expired"}}`, http.StatusUnauthorized)
				return
			}
			if got := r.Header.Get("authorization"); got != "Bearer fresh-access" {
				t.Fatalf("retry authorization = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"rate_limit": map[string]any{
					"primary_window": map[string]any{
						"limit_window_seconds": 18000,
						"used_percent":         7,
						"reset_at":             1770000000,
					},
				},
			})
		case "/oauth/token":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token":  "fresh-access",
				"refresh_token": "refresh-2",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	var stdout bytes.Buffer
	code := Run(context.Background(), Config{
		Home:          home,
		Stdout:        &stdout,
		CodexBaseURL:  upstream.URL,
		TokenEndpoint: upstream.URL + "/oauth/token",
		HTTPClient:    upstream.Client(),
	}, []string{"clx:usage"})
	if code != 0 {
		t.Fatalf("code = %d, stdout = %q", code, stdout.String())
	}
	if usageAttempts != 2 {
		t.Fatalf("usage attempts = %d, want 2", usageAttempts)
	}
	if !strings.Contains(stdout.String(), "five_hour: 7%") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestUsageCommandRequiresAuth(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(context.Background(), Config{Home: t.TempDir(), Stderr: &stderr}, []string{"clx:usage"})
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "not logged in") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func saveAppAuth(t *testing.T, home string) {
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

func appCatalogEntry(slug string, contextWindow int64) map[string]any {
	return map[string]any{
		"slug":                       slug,
		"context_window":             contextWindow,
		"max_context_window":         contextWindow,
		"supported_in_api":           true,
		"visibility":                 "list",
		"default_reasoning_level":    "medium",
		"supported_reasoning_levels": []map[string]string{{"effort": "low"}, {"effort": "medium"}, {"effort": "high"}, {"effort": "xhigh"}, {"effort": "max"}},
	}
}

type cancelAfterWrite struct {
	w      *bytes.Buffer
	cancel context.CancelFunc
}

func (w cancelAfterWrite) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	w.cancel()
	return n, err
}

func TestFableStartupOverrides(t *testing.T) {
	for _, args := range [][]string{{"--claudodex-fable-model", "astra-custom"}, {"--claudodex-models=fable=astra-custom"}} {
		passed, cfg, err := extractStartupModelConfig(append(args, "--model", "fable"), modelconfig.Config{})
		if err != nil || cfg.Fable != "astra-custom" || strings.Join(passed, " ") != "--model fable" {
			t.Fatalf("Fable override: args=%v cfg=%#v err=%v", passed, cfg, err)
		}
	}
}
