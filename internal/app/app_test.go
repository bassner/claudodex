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
	code := Run(context.Background(), Config{Version: "1.2.3", Stdout: &out}, []string{"clx:version"})
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
	code := Run(context.Background(), Config{Launcher: launcher}, []string{"--", "clx:doctor"})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if len(launcher.args) != 1 || launcher.args[0] != "clx:doctor" {
		t.Fatalf("args = %#v", launcher.args)
	}
}

func TestStartupModelFlagsPassConfiguredTargetsToLauncher(t *testing.T) {
	launcher := &fakeLauncher{}
	code := Run(context.Background(), Config{Launcher: launcher}, []string{
		"--claudodex-opus-model", "gpt-opus-next",
		"--claudodex-models=sonnet=gpt-sonnet-next,haiku=gpt-haiku-next",
		"--model", "sonnet",
	})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if launcher.cfg.Models != (modelconfig.Config{Opus: "gpt-opus-next", Sonnet: "gpt-sonnet-next", Haiku: "gpt-haiku-next"}) {
		t.Fatalf("models = %#v", launcher.cfg.Models)
	}
	if strings.Join(launcher.args, " ") != "--model sonnet" {
		t.Fatalf("args = %#v", launcher.args)
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
				{"slug": "gpt-5.5", "context_window": 111000},
				{"slug": "gpt-5.4", "context_window": 222000},
				{"slug": "gpt-5.4-mini", "context_window": 333000},
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

type cancelAfterWrite struct {
	w      *bytes.Buffer
	cancel context.CancelFunc
}

func (w cancelAfterWrite) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	w.cancel()
	return n, err
}
