package launcher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/bassner/claudodex/internal/codex"
)

func TestClaudeRelativeMarkdownFileLinksChild(t *testing.T) {
	if os.Getenv("CLAUDODEX_LINK_TEST_CHILD") != "1" {
		t.Skip("private terminal child only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	err := (ProcessLauncher{}).Launch(ctx, []string{"--model", "opus", "--dangerously-skip-permissions", "--system-prompt", "Local synthetic Markdown-link acceptance fixture.", "Show the local file links."}, Config{
		Version: "smoke", Home: os.Getenv("CLAUDODEX_LINK_TEST_HOME"), CodexBaseURL: os.Getenv("CLAUDODEX_LINK_TEST_BACKEND"), Interactive: true,
	})
	if err != nil && ctx.Err() == nil {
		t.Fatal(err)
	}
}

func TestInstalledClaudeRelativeMarkdownFileLinks(t *testing.T) {
	version := nativeCatalogRequire(t)
	tmux, err := exec.LookPath("tmux")
	if err != nil {
		t.Fatal(err)
	}
	home, userHome, fixture := t.TempDir(), t.TempDir(), t.TempDir()
	fixture, err = filepath.EvalSymlinks(fixture)
	if err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(fixture, "repo")
	files := []string{
		"repo/.local/student-review-2026-11-03/correction-plan-v2.md",
		"repo/.local/student-review-2026-11-03/plan-consultation-result.md",
		"repo/docs/Grüße 日本語.md", "repo/docs/hello world.md", "outside.md",
	}
	for _, name := range files {
		path := filepath.Join(fixture, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("# Local link fixture\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	artifacts := nativeCatalogArtifacts(t)
	nativeCatalogAuth(t, home)
	if err := os.MkdirAll(filepath.Join(userHome, ".claude"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(filepath.Join(userHome, ".claude.json"), map[string]any{"hasCompletedOnboarding": true, "theme": "dark", "projects": map[string]any{work: map[string]any{"hasTrustDialogAccepted": true}}}, 0o600); err != nil {
		t.Fatal(err)
	}
	const markdown = "[Correction plan](.local/student-review-2026-11-03/correction-plan-v2.md)\n\n" +
		"[Consultation result](.local/student-review-2026-11-03/plan-consultation-result.md)\n\n" +
		"[Spaced Unicode](<docs/Grüße 日本語.md>)\n\n" +
		"[Encoded file](docs/hello%20world.md?view=1#section)\n\n" +
		"[Parent file](../outside.md)\n\n" +
		"[Web](https://example.org/docs?q=1#part)\n\n" +
		"[Blocked network file](file:///net/server/path.md)\n\nlocal-links-ready"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/codex/models":
			_ = json.NewEncoder(w).Encode(codex.ModelsResponse{Models: nativeCatalogFixture()})
		case "/codex/responses":
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusUpgradeRequired)
				return
			}
			nativeCatalogText(w, markdown)
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	socketDir, err := os.MkdirTemp("/tmp", "clx-links-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(socketDir)
	socket := filepath.Join(socketDir, "tmux.sock")
	rawPath, ready := filepath.Join(artifacts, "terminal.raw"), filepath.Join(socketDir, "ready")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	// Wait for pipe-pane before launching Claude so every OSC8 byte is captured.
	command := "while [ ! -f " + shellQuote(ready) + " ]; do sleep 0.05; done; cd " + shellQuote(work) + " && exec env"
	for key, value := range map[string]string{
		"HOME": userHome, "CLAUDE_CONFIG_DIR": "", "CLAUDE_SECURESTORAGE_CONFIG_DIR": "", "CLAUDODEX_DISABLE_REAL_CLAUDE_BRIDGE_AUTH": "1",
		"CLAUDODEX_LINK_TEST_CHILD": "1", "CLAUDODEX_LINK_TEST_HOME": home, "CLAUDODEX_LINK_TEST_BACKEND": upstream.URL, "CLAUDODEX_CODEX_BASE_URL": upstream.URL,
		"CLAUDODEX_DISABLE_CODEX_WEBSOCKET": "1", "CLAUDODEX_DISABLE_CLAUDE_PATCH": "", "TERM": "xterm-256color", "FORCE_HYPERLINK": "1",
	} {
		command += " " + key + "=" + shellQuote(value)
	}
	command += " " + shellQuote(executable) + " -test.run='^TestClaudeRelativeMarkdownFileLinksChild$' -test.v"
	if output, err := exec.CommandContext(ctx, tmux, "-S", socket, "new-session", "-d", "-s", "links", "-x", "140", "-y", "45", "/bin/sh -c "+shellQuote(command)).CombinedOutput(); err != nil {
		t.Fatalf("start private terminal: %v: %s", err, output)
	}
	defer func() {
		_ = exec.Command(tmux, "-S", socket, "send-keys", "-t", "links:0.0", "C-d").Run()
		deadline := time.Now().Add(8 * time.Second)
		for time.Now().Before(deadline) {
			if exec.Command(tmux, "-S", socket, "has-session", "-t", "links").Run() != nil {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		_ = exec.Command(tmux, "-S", socket, "kill-server").Run()
	}()
	if output, err := exec.CommandContext(ctx, tmux, "-S", socket, "pipe-pane", "-o", "-t", "links:0.0", "cat > "+shellQuote(rawPath)).CombinedOutput(); err != nil {
		t.Fatalf("capture raw terminal: %v: %s", err, output)
	}
	if err := os.WriteFile(ready, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	capture := func() string {
		output, _ := exec.CommandContext(ctx, tmux, "-S", socket, "capture-pane", "-p", "-S", "-1000", "-t", "links:0.0").CombinedOutput()
		return string(output)
	}
	var screen string
	for ctx.Err() == nil {
		screen = capture()
		if strings.Contains(screen, "local-links-ready") {
			break
		}
		if strings.Contains(screen, "Yes, I accept") || strings.Contains(screen, "Yes, I trust this folder") {
			_ = exec.CommandContext(ctx, tmux, "-S", socket, "send-keys", "-t", "links:0.0", "Down", "Enter").Run()
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err := os.WriteFile(filepath.Join(artifacts, "terminal.txt"), []byte(screen), 0o600); err != nil {
		t.Fatal(err)
	}
	if ctx.Err() != nil {
		t.Fatalf("local links did not reach terminal: %s", screen)
	}
	// Stop the pipe to flush its output before inspecting the real hyperlink targets.
	if output, err := exec.CommandContext(ctx, tmux, "-S", socket, "pipe-pane", "-t", "links:0.0").CombinedOutput(); err != nil {
		t.Fatalf("finish raw terminal capture: %v: %s", err, output)
	}
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatal(err)
	}
	links := regexp.MustCompile("\x1b\\]8;[^;]*;([^\x07\x1b]*)(?:\x07|\x1b\\\\)").FindAllSubmatch(raw, -1)
	targets := map[string]bool{}
	for _, link := range links {
		if len(link[1]) != 0 {
			targets[string(link[1])] = true
		}
	}
	for index, name := range files {
		path := filepath.Join(fixture, name)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("hyperlink target missing: %v", err)
		}
		expected := (&url.URL{Scheme: "file", Path: path}).String()
		if index == 3 {
			expected += "?view=1#section"
		}
		if !targets[expected] {
			t.Errorf("missing absolute OSC8 file target %q; targets=%v", expected, targets)
		}
	}
	if !targets["https://example.org/docs?q=1#part"] {
		t.Error("web hyperlink was changed")
	}
	for target := range targets {
		if strings.HasPrefix(target, ".local/") || strings.HasPrefix(target, "docs/") || strings.HasPrefix(target, "../") || strings.Contains(target, "/net/server/path.md") {
			t.Errorf("unsafe or unresolved OSC8 target: %q", target)
		}
	}
	result := map[string]any{"claude_version": version, "paid_model_requests": 0, "target_files_exist": true, "osc8_targets": targets, "passed": !t.Failed()}
	if err := writeJSONFile(filepath.Join(artifacts, "result.json"), result, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("real terminal local-link acceptance artifacts: %s", artifacts)
}
