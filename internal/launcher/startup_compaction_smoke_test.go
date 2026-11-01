package launcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bassner/claudodex/internal/codex"
)

func TestInstalledClaudeStartupCompactionChild(t *testing.T) {
	if os.Getenv("CLAUDODEX_STARTUP_TEST_CHILD") != "1" {
		t.Skip("private terminal child only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	err := (ProcessLauncher{}).Launch(ctx, []string{"--model", "gpt-6.1-sol", "--dangerously-skip-permissions", "--system-prompt", "Local synthetic acceptance fixture.", "Reply ready"}, Config{
		Version: "0.5.3", Home: os.Getenv("CLAUDODEX_STARTUP_TEST_HOME"), CodexBaseURL: os.Getenv("CLAUDODEX_CODEX_BASE_URL"), Interactive: true,
	})
	if err != nil && ctx.Err() == nil {
		t.Fatal(err)
	}
}

func TestInstalledClaudeStartupAndModelCompaction(t *testing.T) {
	nativeCatalogRequire(t)
	tmux, err := exec.LookPath("tmux")
	if err != nil {
		t.Fatal(err)
	}
	home, userHome, work := t.TempDir(), t.TempDir(), t.TempDir()
	work, err = filepath.EvalSymlinks(work)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := nativeCatalogArtifacts(t)
	nativeCatalogAuth(t, home)
	if err := os.MkdirAll(filepath.Join(userHome, ".claude"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(filepath.Join(userHome, ".claude.json"), map[string]any{"hasCompletedOnboarding": true, "theme": "dark", "projects": map[string]any{work: map[string]any{"hasTrustDialogAccepted": true}}}, 0o600); err != nil {
		t.Fatal(err)
	}
	catalog := nativeCatalogFixture()
	for index := range catalog {
		catalog[index].ContextWindow = 300000
		catalog[index].MaxContextWindow = 872000
		if catalog[index].Slug == "gpt-6.1-sol" {
			catalog[index].ContextWindow = 500000
		}
	}
	metadataStarted, allowMetadata := make(chan struct{}), make(chan struct{})
	var modelRequests atomic.Int32
	var started, allowed sync.Once
	allow := func() { allowed.Do(func() { close(allowMetadata) }) }
	defer allow()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/codex/models" {
			started.Do(func() { close(metadataStarted) })
			select {
			case <-allowMetadata:
			case <-r.Context().Done():
				return
			}
			_ = json.NewEncoder(w).Encode(codex.ModelsResponse{Models: catalog})
			return
		}
		if r.URL.Path != "/codex/responses" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusUpgradeRequired)
			return
		}
		modelRequests.Add(1)
		nativeCatalogText(w, "startup-handoff-ready")
	}))
	defer upstream.Close()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	socketDir, err := os.MkdirTemp("/tmp", "clx-start-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(socketDir)
	socket := filepath.Join(socketDir, "tmux.sock")
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	command := "cd " + shellQuote(work) + " && env"
	for key, value := range map[string]string{
		"HOME": userHome, "CLAUDE_CONFIG_DIR": "", "CLAUDE_SECURESTORAGE_CONFIG_DIR": "", "CLAUDODEX_DISABLE_REAL_CLAUDE_BRIDGE_AUTH": "1",
		"CLAUDODEX_STARTUP_TEST_CHILD": "1", "CLAUDODEX_STARTUP_TEST_HOME": home, "CLAUDODEX_CODEX_BASE_URL": upstream.URL,
		"CLAUDODEX_DISABLE_CODEX_WEBSOCKET": "1", "CLAUDODEX_DISABLE_CLAUDE_PATCH": "", "TERM": "xterm-256color",
		"CLAUDODEX_PROXY_LOG": filepath.Join(artifacts, "http.log"),
	} {
		command += " " + key + "=" + shellQuote(value)
	}
	command += " " + shellQuote(executable) + " -test.run='^TestInstalledClaudeStartupCompactionChild$' -test.v"
	if output, err := exec.CommandContext(ctx, tmux, "-S", socket, "new-session", "-d", "-s", "startup", "-x", "120", "-y", "36", command).CombinedOutput(); err != nil {
		t.Fatalf("start private terminal: %v: %s", err, output)
	}
	defer func() {
		_ = exec.Command(tmux, "-S", socket, "send-keys", "-t", "startup:0.0", "Escape").Run()
		time.Sleep(200 * time.Millisecond)
		_ = exec.Command(tmux, "-S", socket, "send-keys", "-t", "startup:0.0", "C-d").Run()
		deadline := time.Now().Add(8 * time.Second)
		for time.Now().Before(deadline) {
			if exec.Command(tmux, "-S", socket, "has-session", "-t", "startup").Run() != nil {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		_ = exec.Command(tmux, "-S", socket, "kill-server").Run()
	}()
	capture := func() string {
		output, _ := exec.CommandContext(ctx, tmux, "-S", socket, "capture-pane", "-p", "-t", "startup:0.0").CombinedOutput()
		return string(output)
	}
	defer func() { _ = os.WriteFile(filepath.Join(artifacts, "terminal-final.txt"), []byte(capture()), 0o600) }()
	send := func(keys ...string) {
		args := append([]string{"-S", socket, "send-keys", "-t", "startup:0.0"}, keys...)
		if output, err := exec.CommandContext(ctx, tmux, args...).CombinedOutput(); err != nil {
			t.Fatalf("send terminal keys: %v: %s", err, output)
		}
	}
	acceptDialogs := func(screen string) {
		if strings.Contains(screen, "Yes, I accept") || strings.Contains(screen, "Yes, I trust this folder") {
			send("Down", "Enter")
		}
		if strings.Contains(screen, "Switch model?") && strings.Contains(screen, "1. Yes, switch") {
			send("Enter")
		}
	}
	wait := func(name, text string) string {
		t.Helper()
		for ctx.Err() == nil {
			screen := capture()
			if strings.Contains(screen, text) {
				_ = os.WriteFile(filepath.Join(artifacts, name+".txt"), []byte(screen), 0o600)
				return screen
			}
			acceptDialogs(screen)
			time.Sleep(200 * time.Millisecond)
		}
		t.Fatalf("waiting for %s (%s):\n%s", name, text, capture())
		return ""
	}
	waitModel := func(model string) {
		t.Helper()
		for ctx.Err() == nil {
			screen := capture()
			pos := strings.LastIndex(screen, "❯ /model "+model)
			if pos >= 0 && strings.Contains(screen[pos:], "Set model to") {
				return
			}
			acceptDialogs(screen)
			time.Sleep(200 * time.Millisecond)
		}
		t.Fatalf("model switch did not finish: %s", capture())
	}
	select {
	case <-metadataStarted:
	case <-ctx.Done():
		t.Fatalf("startup did not reach metadata: %s", capture())
	}
	screen := wait("startup-banner", "Loading the Codex model catalog")
	if !strings.Contains(screen, "v0.5.3") || !strings.Contains(screen, "[INFO]") || !strings.Contains(screen, "▗▄▄▖") {
		t.Fatalf("startup banner missing: %s", screen)
	}
	allow()
	screen = wait("harness-handoff", "startup-handoff-ready")
	if strings.Contains(screen, "Loading the Codex model catalog") || strings.Contains(screen, "[INFO]") || strings.Contains(screen, "▗▄▄▖") {
		t.Fatalf("startup screen remained after handoff: %s", screen)
	}
	for _, check := range []struct{ model, window string }{{"gpt-6.1-sol", "475k"}, {"gpt-6-luna", "285k"}, {"opus", "285k"}, {"gpt-6.1-sol[1m]", "828.4k"}} {
		send("/model "+check.model, "Enter")
		waitModel(check.model)
		send("/autocompact", "Enter")
		screen = wait("window-"+check.model, "Select auto-compact window")
		current := ""
		for _, line := range strings.Split(screen, "\n") {
			if strings.Contains(line, "Current setting for ") {
				current = line
			}
		}
		if !strings.Contains(current, check.window+" tokens") {
			t.Fatalf("%s did not use its own %s compaction window:\n%s", check.model, check.window, screen)
		}
		send("Escape")
		for strings.Contains(capture(), "Select auto-compact window") && ctx.Err() == nil {
			time.Sleep(100 * time.Millisecond)
		}
	}
	send("/autocompact 200k", "Enter")
	wait("user-lower-window", "200k")
	send("/autocompact", "Enter")
	screen = wait("user-lower-window-dialog", "Select auto-compact window")
	if !strings.Contains(screen, "200k tokens (from settings)") {
		t.Fatalf("lower user window did not win: %s", screen)
	}
	send("Escape")
	before := modelRequests.Load()
	t.Setenv("HOME", userHome)
	var stdout, stderr bytes.Buffer
	err = (ProcessLauncher{}).Launch(ctx, []string{"-p", "unused", "--model", "gpt-9-missing"}, Config{
		Version: "0.5.3", Home: home, CodexBaseURL: upstream.URL, Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr,
	})
	if err == nil || modelRequests.Load() != before {
		t.Fatalf("model missing from catalog reached upstream: %v, %s", err, stdout.String())
	}
	requestLog, err := os.ReadFile(filepath.Join(artifacts, "http.log"))
	if err != nil || bytes.Contains(requestLog, []byte("/api/claude_cli/bootstrap")) {
		t.Fatalf("cache-only macOS compaction gate unexpectedly needed bootstrap: %v", err)
	}
	t.Logf("real terminal acceptance captures: %s", artifacts)
	fmt.Fprintln(os.Stderr, "startup and per-model compaction acceptance passed")
}
