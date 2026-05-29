package launcher

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestInstalledClaudePrintSmokeWithFakeCodexUpstream(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skipf("claude binary not available: %v", err)
	}

	home := t.TempDir()
	saveLauncherAuth(t, home)
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)

	var upstreamRequests atomic.Int32
	var captured map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/codex/models" {
			_, _ = io.WriteString(w, `{"models":[{"slug":"gpt-5.5","context_window":272000},{"slug":"gpt-5.4","context_window":300000},{"slug":"gpt-5.4-mini","context_window":400000}]}`)
			return
		}
		if r.URL.Path != "/codex/responses" {
			t.Fatalf("unexpected upstream path %s", r.URL.Path)
		}
		upstreamRequests.Add(1)
		if got := r.Header.Get("authorization"); got != "Bearer access-1" {
			t.Fatalf("authorization = %q", got)
		}
		if got := r.Header.Get("chatgpt-account-id"); got != "acc_123" {
			t.Fatalf("chatgpt-account-id = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: response.created`,
			`data: {"type":"response.created","response":{"id":"resp_smoke"}}`,
			``,
			`event: response.output_item.added`,
			`data: {"type":"response.output_item.added","item":{"type":"message","id":"item_smoke"}}`,
			``,
			`event: response.output_text.delta`,
			`data: {"type":"response.output_text.delta","delta":"ok"}`,
			``,
			`event: response.output_item.done`,
			`data: {"type":"response.output_item.done","item":{"type":"message","id":"item_smoke","content":[{"type":"output_text","text":"ok"}]}}`,
			``,
			`event: response.completed`,
			`data: {"type":"response.completed","response":{"stop_reason":"stop","usage":{"input_tokens":2,"output_tokens":1}}}`,
			``,
			``,
		}, "\n")))
	}))
	defer upstream.Close()
	t.Setenv("CLAUDODEX_CODEX_BASE_URL", upstream.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := (ProcessLauncher{}).Launch(ctx, []string{
		"-p", "say ok",
		"--model", "claude-sonnet-4-6",
		"--dangerously-skip-permissions",
		"--max-turns", "1",
	}, Config{
		Version:      "smoke",
		Stdin:        strings.NewReader(""),
		Stdout:       &stdout,
		Stderr:       &stderr,
		Home:         home,
		CodexBaseURL: upstream.URL,
	})
	if err != nil {
		t.Fatalf("launch failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if upstreamRequests.Load() == 0 {
		t.Fatalf("fake Codex upstream was not called\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok") {
		t.Fatalf("stdout did not include model output\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if captured["model"] != "gpt-5.4" {
		t.Fatalf("upstream model = %#v, want gpt-5.4; request=%#v", captured["model"], captured)
	}
	instructions, _ := captured["instructions"].(string)
	if !strings.Contains(instructions, "the follow-up after tool results must not greet again or restart the conversation") {
		t.Fatalf("installed Claude request is missing Claudodex same-turn greeting guard; instructions=%q request=%#v", instructions, captured)
	}
	if !strings.Contains(instructions, "perform that opening at most once per user-visible turn") {
		t.Fatalf("installed Claude request is missing Claudodex setup continuation guard; instructions=%q request=%#v", instructions, captured)
	}
	if !strings.Contains(instructions, "resolve symlinks first and operate on the real target path") {
		t.Fatalf("installed Claude request is missing Claudodex sidecar path guidance; instructions=%q request=%#v", instructions, captured)
	}
}

func TestInstalledClaudeFastModeSmokeWithFakeCodexUpstream(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skipf("claude binary not available: %v", err)
	}

	home := t.TempDir()
	saveLauncherAuth(t, home)
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)
	settingsPath := filepath.Join(userHome, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(settingsPath, map[string]any{"fastMode": true}, 0o600); err != nil {
		t.Fatal(err)
	}

	var upstreamRequests atomic.Int32
	var captured map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/codex/models" {
			_, _ = io.WriteString(w, `{"models":[{"slug":"gpt-5.5","context_window":272000},{"slug":"gpt-5.4","context_window":300000},{"slug":"gpt-5.4-mini","context_window":400000}]}`)
			return
		}
		if r.URL.Path != "/codex/responses" {
			t.Fatalf("unexpected upstream path %s", r.URL.Path)
		}
		upstreamRequests.Add(1)
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: response.created`,
			`data: {"type":"response.created","response":{"id":"resp_fast_smoke"}}`,
			``,
			`event: response.output_item.added`,
			`data: {"type":"response.output_item.added","item":{"type":"message","id":"item_fast_smoke"}}`,
			``,
			`event: response.output_text.delta`,
			`data: {"type":"response.output_text.delta","delta":"ok"}`,
			``,
			`event: response.output_item.done`,
			`data: {"type":"response.output_item.done","item":{"type":"message","id":"item_fast_smoke","content":[{"type":"output_text","text":"ok"}]}}`,
			``,
			`event: response.completed`,
			`data: {"type":"response.completed","response":{"stop_reason":"stop","usage":{"input_tokens":2,"output_tokens":1}}}`,
			``,
			``,
		}, "\n")))
	}))
	defer upstream.Close()
	t.Setenv("CLAUDODEX_CODEX_BASE_URL", upstream.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := (ProcessLauncher{}).Launch(ctx, []string{
		"-p", "say ok",
		"--model", "opus",
		"--settings", `{"fastMode":true}`,
		"--dangerously-skip-permissions",
		"--max-turns", "1",
	}, Config{
		Version:      "smoke",
		Stdin:        strings.NewReader(""),
		Stdout:       &stdout,
		Stderr:       &stderr,
		Home:         home,
		CodexBaseURL: upstream.URL,
	})
	if err != nil {
		t.Fatalf("launch failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if upstreamRequests.Load() == 0 {
		t.Fatalf("fake Codex upstream was not called\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if captured["model"] != "gpt-5.5" {
		t.Fatalf("upstream model = %#v, want gpt-5.5; request=%#v", captured["model"], captured)
	}
	if captured["service_tier"] != "priority" {
		t.Fatalf("service_tier = %#v, want priority; request=%#v\nstdout:\n%s\nstderr:\n%s", captured["service_tier"], captured, stdout.String(), stderr.String())
	}
}

func TestInstalledClaudeSmokeWithUIPatchDisabled(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skipf("claude binary not available: %v", err)
	}

	t.Setenv("CLAUDODEX_DISABLE_CLAUDE_PATCH", "1")

	home := t.TempDir()
	saveLauncherAuth(t, home)
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)

	var upstreamRequests atomic.Int32
	var captured map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/codex/models" {
			_, _ = io.WriteString(w, `{"models":[{"slug":"gpt-5.5","context_window":272000},{"slug":"gpt-5.4","context_window":300000},{"slug":"gpt-5.4-mini","context_window":400000}]}`)
			return
		}
		if r.URL.Path != "/codex/responses" {
			t.Fatalf("unexpected upstream path %s", r.URL.Path)
		}
		upstreamRequests.Add(1)
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: response.created`,
			`data: {"type":"response.created","response":{"id":"resp_unpatched_smoke"}}`,
			``,
			`event: response.output_item.added`,
			`data: {"type":"response.output_item.added","item":{"type":"message","id":"item_unpatched_smoke"}}`,
			``,
			`event: response.output_text.delta`,
			`data: {"type":"response.output_text.delta","delta":"ok"}`,
			``,
			`event: response.output_item.done`,
			`data: {"type":"response.output_item.done","item":{"type":"message","id":"item_unpatched_smoke","content":[{"type":"output_text","text":"ok"}]}}`,
			``,
			`event: response.completed`,
			`data: {"type":"response.completed","response":{"stop_reason":"stop","usage":{"input_tokens":2,"output_tokens":1}}}`,
			``,
			``,
		}, "\n")))
	}))
	defer upstream.Close()
	t.Setenv("CLAUDODEX_CODEX_BASE_URL", upstream.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := (ProcessLauncher{}).Launch(ctx, []string{
		"-p", "say ok",
		"--model", "claude-sonnet-4-6",
		"--dangerously-skip-permissions",
		"--max-turns", "1",
	}, Config{
		Version:      "smoke",
		Stdin:        strings.NewReader(""),
		Stdout:       &stdout,
		Stderr:       &stderr,
		Home:         home,
		CodexBaseURL: upstream.URL,
	})
	if err != nil {
		t.Fatalf("launch failed with UI patch disabled: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if upstreamRequests.Load() == 0 {
		t.Fatalf("fake Codex upstream was not called\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok") {
		t.Fatalf("stdout did not include model output\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if captured["model"] != "gpt-5.4" {
		t.Fatalf("upstream model = %#v, want gpt-5.4; request=%#v", captured["model"], captured)
	}
}
