package launcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	"github.com/bassner/claudodex/internal/auth"
	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
)

func nativeCatalogFixture() []codex.ModelInfo {
	var models []codex.ModelInfo
	for _, cfg := range []modelconfig.Config{modelconfig.Default(), {Opus: "gpt-8.2-sol", Sonnet: "gpt-8.2-terra", Haiku: "gpt-8-luna", Fable: "gpt-8-astra"}} {
		for _, slug := range cfg.RequiredModels() {
			model := codex.ModelInfo{Slug: slug, DisplayName: slug, SupportedInAPI: true, Visibility: "list", ContextWindow: 372000, MaxContextWindow: 872000,
				DefaultReasoningLevel: "low", SupportedReasoningLevels: []codex.ReasoningEffortPreset{{Effort: "low"}, {Effort: "high"}}}
			if slug == modelconfig.DefaultOpusModel {
				model.DefaultReasoningLevel = "medium"
				model.SupportedReasoningLevels = []codex.ReasoningEffortPreset{{Effort: "medium"}, {Effort: "max"}}
			}
			if slug == "gpt-8.2-sol" {
				model.ServiceTiers = []codex.ModelServiceTier{{ID: "priority", Name: "Fast"}}
				model.AdditionalSpeedTiers = []string{"fast"}
			}
			models = append(models, model)
		}
	}
	return models
}

func nativeCatalogAuth(t *testing.T, home string) {
	t.Helper()
	if err := auth.NewStore(home).Save(auth.File{AuthMode: "chatgpt", Issuer: auth.Issuer, ClientID: auth.ClientID,
		Tokens: auth.Tokens{AccessToken: "access-1", RefreshToken: "refresh-1", AccountID: "synthetic-catalog-account", UserID: "synthetic-catalog-user"}}); err != nil {
		t.Fatal(err)
	}
}

func nativeCatalogRequire(t *testing.T) string {
	t.Helper()
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude fake-backend acceptance")
	}
	installed, err := exec.LookPath("claude")
	if err != nil {
		t.Skipf("claude binary not available: %v", err)
	}
	installed, err = filepath.EvalSymlinks(installed)
	if err != nil {
		t.Fatal(err)
	}
	version, _ := requireInstalledClaudeUIPatch(t, installed)
	return version
}

func nativeCatalogArtifacts(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("CLAUDODEX_CATALOG_ACCEPTANCE_ARTIFACT_DIR")
	if dir == "" {
		return t.TempDir()
	}
	dir = filepath.Join(dir, t.Name())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func nativeCatalogText(w http.ResponseWriter, text string) {
	w.Header().Set("content-type", "text/event-stream")
	_, _ = fmt.Fprintf(w, "event: response.created\ndata: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_catalog\"}}\n\nevent: response.output_item.added\ndata: {\"type\":\"response.output_item.added\",\"item\":{\"type\":\"message\",\"id\":\"item_catalog\"}}\n\nevent: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":%q}\n\nevent: response.output_item.done\ndata: {\"type\":\"response.output_item.done\",\"item\":{\"type\":\"message\",\"id\":\"item_catalog\",\"content\":[{\"type\":\"output_text\",\"text\":%q}]}}\n\nevent: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"stop_reason\":\"stop\",\"usage\":{\"input_tokens\":2,\"output_tokens\":1}}}\n\n", text, text)
}

func TestInstalledClaudeCatalogStartupFakeUpstream(t *testing.T) {
	version := nativeCatalogRequire(t)
	home, userHome, work := t.TempDir(), t.TempDir(), t.TempDir()
	artifacts := nativeCatalogArtifacts(t)
	nativeCatalogAuth(t, home)
	t.Chdir(work)
	t.Setenv("HOME", userHome)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CLAUDE_SECURESTORAGE_CONFIG_DIR", "")
	t.Setenv("CLAUDODEX_DISABLE_REAL_CLAUDE_BRIDGE_AUTH", "1")
	var outage atomic.Bool
	var metadataCalls atomic.Int32
	var mu sync.Mutex
	var requests []map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/codex/models" {
			metadataCalls.Add(1)
			if outage.Load() {
				w.WriteHeader(503)
				return
			}
			_ = json.NewEncoder(w).Encode(codex.ModelsResponse{Models: nativeCatalogFixture()})
			return
		}
		if r.URL.Path != "/codex/responses" {
			http.Error(w, "unexpected fake path", 404)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusUpgradeRequired)
			return
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode fake request: %v", err)
			http.Error(w, "bad request", 400)
			return
		}
		mu.Lock()
		requests = append(requests, request)
		mu.Unlock()
		nativeCatalogText(w, "native-catalog-ok")
	}))
	defer upstream.Close()
	t.Setenv("CLAUDODEX_CODEX_BASE_URL", upstream.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()
	for index, route := range []struct {
		requested, expected, effort string
		cached                      bool
		mode                        string
	}{
		{"opus", "gpt-8.2-sol", "high", false, ""},
		{"sonnet", "gpt-8.2-terra", "high", false, ""},
		{"haiku", "gpt-8-luna", "high", false, ""},
		{"fable", "gpt-8-astra", "high", false, ""},
		{"sonnet", "gpt-8.2-terra", "high", true, ""},
		{modelconfig.DefaultOpusModel, modelconfig.DefaultOpusModel, "max", false, ""},
		{modelconfig.DefaultOpusModel, modelconfig.DefaultOpusModel, "medium", false, "saved-direct"},
		{"opus", "gpt-8.2-sol", "low", false, "omitted-effort"},
	} {
		outage.Store(route.cached)
		settingsPath := filepath.Join(userHome, ".claude", "settings.json")
		if route.mode == "saved-direct" {
			if err := os.MkdirAll(filepath.Dir(settingsPath), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(settingsPath, []byte(`{"model":"gpt-6.1-sol"}`), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		args := []string{"-p", "Reply with native-catalog-ok", "--max-turns", "1", "--dangerously-skip-permissions", "--output-format", "stream-json", "--verbose", "--system-prompt", "Local synthetic catalog acceptance fixture."}
		if route.mode != "saved-direct" {
			args = append(args, "--model", route.requested)
		}
		if route.mode == "" {
			args = append(args, "--effort", "max")
		}
		var stdout, stderr bytes.Buffer
		err := (ProcessLauncher{}).Launch(ctx, args, Config{Version: "smoke", Home: home, CodexBaseURL: upstream.URL, Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr})
		name := fmt.Sprintf("launch-%d", index)
		for suffix, data := range map[string][]byte{"-stdout.jsonl": stdout.Bytes(), "-stderr.txt": stderr.Bytes()} {
			if writeErr := os.WriteFile(filepath.Join(artifacts, name+suffix), data, 0o600); writeErr != nil {
				t.Fatal(writeErr)
			}
		}
		if err != nil {
			t.Fatalf("%s route=%s: %v\n%s\n%s", name, route.requested, err, stdout.String(), stderr.String())
		}
		if !strings.Contains(stdout.String(), "native-catalog-ok") {
			t.Fatalf("real Claude did not return fake output: %s", stdout.String())
		}
		foundVersion := false
		for _, line := range strings.Split(stdout.String(), "\n") {
			var event map[string]any
			if json.Unmarshal([]byte(line), &event) == nil && event["type"] == "system" && event["subtype"] == "init" {
				foundVersion = true
				if event["claude_code_version"] != version {
					t.Fatalf("native selected Claude %v, want installed verified %s", event["claude_code_version"], version)
				}
			}
		}
		if !foundVersion {
			t.Fatalf("native init did not report selected Claude version: %s", stdout.String())
		}
		if strings.Contains(stderr.String(), "unrecognized_model") || strings.Contains(stderr.String(), "unpatched Claude Code") {
			t.Fatalf("real Claude rejected discovered model: %s", stderr.String())
		}
		if strings.Contains(stderr.String(), "using model metadata cached") != route.cached {
			t.Fatalf("cached=%t stderr=%s", route.cached, stderr.String())
		}
		mu.Lock()
		if len(requests) != index+1 {
			count := len(requests)
			mu.Unlock()
			t.Fatalf("%s produced %d total fake generation requests, want %d", name, count, index+1)
		}
		last := requests[len(requests)-1]
		mu.Unlock()
		if last["model"] != route.expected {
			t.Fatalf("native route %s = %v, want %s", route.requested, last["model"], route.expected)
		}
		assertCapturedReasoningEffort(t, last, route.effort)
		if route.mode == "saved-direct" {
			after, err := os.ReadFile(settingsPath)
			if err != nil || string(after) != `{"model":"gpt-6.1-sol"}` {
				t.Fatalf("saved direct model changed: %s, %v", after, err)
			}
		}
	}
	if metadataCalls.Load() != 8 {
		t.Fatalf("metadata calls = %d, want fresh attempt on all eight launches", metadataCalls.Load())
	}
	mu.Lock()
	captured := append([]map[string]any(nil), requests...)
	mu.Unlock()
	result := map[string]any{"passed": true, "claude_version": version, "paid_model_requests": 0, "metadata_requests": metadataCalls.Load(), "future_four_role_routes": true, "old_direct_slug_preserved": true, "saved_direct_slug_preserved": true, "effort_clamped": true, "omitted_effort_uses_catalog_default": true, "cached_startup_after_503": true}
	for name, value := range map[string]any{"result.json": result, "requests.json": captured} {
		data, _ := json.MarshalIndent(value, "", "  ")
		if err := os.WriteFile(filepath.Join(artifacts, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("native catalog acceptance artifacts: %s", artifacts)
}

func TestInstalledClaudeCatalogEffortPreferencesFakeUpstream(t *testing.T) {
	version := nativeCatalogRequire(t)
	home, userHome := t.TempDir(), t.TempDir()
	artifacts := nativeCatalogArtifacts(t)
	nativeCatalogAuth(t, home)
	t.Setenv("HOME", userHome)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CLAUDE_SECURESTORAGE_CONFIG_DIR", "")
	t.Setenv("CLAUDODEX_DISABLE_REAL_CLAUDE_BRIDGE_AUTH", "1")
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "")
	t.Chdir(t.TempDir())
	tests := []struct {
		name, desired, settings, envEffort string
		levels                             []string
	}{
		{"mixed-none", "none", `{}`, "", []string{"none", "high"}},
		{"mixed-minimal", "minimal", `{}`, "", []string{"minimal", "high"}},
		{"saved-per-model", "high", `{"modelSettings":{"gpt-8.2-sol":{"effortLevel":"high"}}}`, "", []string{"low", "high"}},
		{"explicit-environment", "minimal", `{}`, "low", []string{"none", "minimal", "high"}},
	}
	var phase atomic.Int32
	var mu sync.Mutex
	var requests []map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/codex/models" {
			models := nativeCatalogFixture()
			current := tests[phase.Load()]
			for index := range models {
				if models[index].Slug == "gpt-8.2-sol" {
					models[index].DefaultReasoningLevel = "low"
					if current.name == "mixed-none" || current.name == "explicit-environment" {
						models[index].DefaultReasoningLevel = "none"
					}
					if current.name == "mixed-minimal" {
						models[index].DefaultReasoningLevel = "minimal"
					}
					models[index].SupportedReasoningLevels = nil
					for _, level := range current.levels {
						models[index].SupportedReasoningLevels = append(models[index].SupportedReasoningLevels, codex.ReasoningEffortPreset{Effort: level})
					}
				}
			}
			_ = json.NewEncoder(w).Encode(codex.ModelsResponse{Models: models})
			return
		}
		if r.URL.Path != "/codex/responses" || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusUpgradeRequired)
			return
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode fake request: %v", err)
			return
		}
		mu.Lock()
		requests = append(requests, request)
		mu.Unlock()
		nativeCatalogText(w, "native-effort-ok")
	}))
	defer upstream.Close()
	t.Setenv("CLAUDODEX_CODEX_BASE_URL", upstream.URL)
	settingsPath := filepath.Join(userHome, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	for index, current := range tests {
		phase.Store(int32(index))
		t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", current.envEffort)
		if err := os.WriteFile(settingsPath, []byte(current.settings), 0o600); err != nil {
			t.Fatal(err)
		}
		var stdout, stderr bytes.Buffer
		err := (ProcessLauncher{}).Launch(ctx, []string{"-p", "Reply with native-effort-ok", "--model", "opus", "--max-turns", "1", "--dangerously-skip-permissions", "--output-format", "stream-json", "--verbose", "--system-prompt", "Local synthetic catalog effort acceptance fixture."}, Config{Version: "smoke", Home: home, CodexBaseURL: upstream.URL, Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr})
		for suffix, data := range map[string][]byte{"-stdout.jsonl": stdout.Bytes(), "-stderr.txt": stderr.Bytes()} {
			if err := os.WriteFile(filepath.Join(artifacts, current.name+suffix), data, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		if err != nil || !strings.Contains(stdout.String(), "native-effort-ok") || strings.Contains(stderr.String(), "unrecognized_model") || strings.Contains(stderr.String(), "using verified local Claude Code") {
			t.Fatalf("native effort %s: %v\n%s\n%s", current.name, err, stdout.String(), stderr.String())
		}
		mu.Lock()
		if len(requests) != index+1 {
			count := len(requests)
			mu.Unlock()
			t.Fatalf("native effort %s: requests=%d, want %d", current.name, count, index+1)
		}
		last := requests[len(requests)-1]
		mu.Unlock()
		if last["model"] != "gpt-8.2-sol" {
			t.Fatalf("native effort routed wrong model: %v", last["model"])
		}
		assertCapturedReasoningEffort(t, last, current.desired)
		if after, err := os.ReadFile(settingsPath); err != nil || string(after) != current.settings {
			t.Fatalf("native effort modified saved settings: %s, %v", after, err)
		}
	}
	mu.Lock()
	captured := append([]map[string]any(nil), requests...)
	mu.Unlock()
	result := map[string]any{"passed": true, "claude_version": version, "paid_model_requests": 0, "mixed_none_default": true, "mixed_minimal_default": true, "saved_per_model_effort": true, "explicit_native_environment_effort": true}
	for name, value := range map[string]any{"result.json": result, "requests.json": captured} {
		data, _ := json.MarshalIndent(value, "", "  ")
		if err := os.WriteFile(filepath.Join(artifacts, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("native effort acceptance artifacts: %s", artifacts)
}

// This helper runs only as a child inside the test's private tmux server.
func TestInstalledClaudeCatalogTmuxChild(t *testing.T) {
	if os.Getenv("CLAUDODEX_CATALOG_TMUX_CHILD") != "1" {
		t.Skip("private tmux child only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	args := []string{"--model", "opus", "--effort", "max", "--settings", `{"fastMode":true}`, "--dangerously-skip-permissions", "--system-prompt", "Local fake-backend acceptance. Use the Agent tool when asked.", "Use an Agent worker to return nested-catalog-ok."}
	if err := (ProcessLauncher{}).Launch(ctx, args, Config{Version: "smoke", Home: os.Getenv("CLAUDODEX_CATALOG_TMUX_HOME"), CodexBaseURL: os.Getenv("CLAUDODEX_CATALOG_TMUX_BACKEND"), Interactive: true}); err != nil {
		t.Fatal(err)
	}
}

func TestInstalledClaudeCatalogTmuxInheritedFastFakeUpstream(t *testing.T) {
	version := nativeCatalogRequire(t)
	tmux, err := exec.LookPath("tmux")
	if err != nil {
		t.Skipf("tmux not available: %v", err)
	}
	home, userHome, work := t.TempDir(), t.TempDir(), t.TempDir()
	work, err = filepath.EvalSymlinks(work)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := nativeCatalogArtifacts(t)
	nativeCatalogAuth(t, home)
	settingsDir := filepath.Join(userHome, ".claude")
	if err := os.MkdirAll(settingsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(filepath.Join(settingsDir, "settings.json"), map[string]any{"fastMode": true}, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(filepath.Join(userHome, ".claude.json"), map[string]any{"hasCompletedOnboarding": true, "theme": "dark", "projects": map[string]any{work: map[string]any{"hasTrustDialogAccepted": true}}}, 0o600); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var requests []map[string]any
	var sentAgent atomic.Bool
	finished := make(chan struct{}, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/codex/models" {
			_ = json.NewEncoder(w).Encode(codex.ModelsResponse{Models: nativeCatalogFixture()})
			return
		}
		if r.URL.Path != "/codex/responses" {
			http.Error(w, "unexpected fake path", 404)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusUpgradeRequired)
			return
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			if err == io.EOF {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			t.Errorf("decode native request: %v", err)
			return
		}
		mu.Lock()
		requests = append(requests, request)
		mu.Unlock()
		if request["model"] == "gpt-8-luna" {
			nativeCatalogText(w, "nested-catalog-ok")
			return
		}
		if !sentAgent.Swap(true) {
			arguments, _ := json.Marshal(map[string]any{"description": "Check inherited speed", "prompt": "Reply only nested-catalog-ok", "subagent_type": "general-purpose", "model": "gpt-8-luna", "run_in_background": false})
			item := map[string]any{"type": "function_call", "id": "fc_catalog_agent", "call_id": "call_catalog_agent", "name": "Agent", "arguments": string(arguments)}
			w.Header().Set("content-type", "text/event-stream")
			for _, event := range []map[string]any{{"type": "response.created", "response": map[string]any{"id": "resp_agent"}}, {"type": "response.output_item.added", "output_index": 0, "item": item}, {"type": "response.output_item.done", "output_index": 0, "item": item}, {"type": "response.completed", "response": map[string]any{"stop_reason": "tool_calls", "usage": map[string]any{"input_tokens": 2, "output_tokens": 1}}}} {
				data, _ := json.Marshal(event)
				_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event["type"], data)
			}
			return
		}
		nativeCatalogText(w, "catalog-main-finished")
		select {
		case finished <- struct{}{}:
		default:
		}
	}))
	defer upstream.Close()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// tmux uses a Unix socket with a small platform path limit. Go's test
	// directory includes the long test name and can exceed that limit on macOS.
	socketDir, err := os.MkdirTemp("/tmp", "clx-tmux-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(socketDir)
	socket := filepath.Join(socketDir, "tmux.sock")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	tracePath := filepath.Join(artifacts, "proxy.jsonl")
	assignments := map[string]string{"HOME": userHome, "CLAUDE_CONFIG_DIR": "", "CLAUDE_SECURESTORAGE_CONFIG_DIR": "", "CLAUDODEX_DISABLE_REAL_CLAUDE_BRIDGE_AUTH": "1", "CLAUDODEX_CATALOG_TMUX_CHILD": "1", "CLAUDODEX_CATALOG_TMUX_HOME": home, "CLAUDODEX_CATALOG_TMUX_BACKEND": upstream.URL, "CLAUDODEX_CODEX_BASE_URL": upstream.URL, "CLAUDODEX_PROXY_LOG": tracePath, "CLAUDODEX_DISABLE_CLAUDE_PATCH": "", "TERM": "xterm-256color"}
	command := "cd " + shellQuote(work) + " && env"
	for key, value := range assignments {
		command += " " + key + "=" + shellQuote(value)
	}
	command += " " + shellQuote(executable) + " -test.run='^TestInstalledClaudeCatalogTmuxChild$' -test.v"
	if output, err := exec.CommandContext(ctx, tmux, "-S", socket, "new-session", "-d", "-s", "catalog", "-x", "130", "-y", "40", command).CombinedOutput(); err != nil {
		t.Fatalf("start private tmux: %v: %s", err, output)
	}
	defer func() {
		_ = exec.Command(tmux, "-S", socket, "send-keys", "-t", "catalog:0.0", "C-d").Run()
		deadline := time.Now().Add(8 * time.Second)
		for time.Now().Before(deadline) {
			if exec.Command(tmux, "-S", socket, "has-session", "-t", "catalog").Run() != nil {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		_ = exec.Command(tmux, "-S", socket, "kill-server").Run()
	}()
	capture := func() string {
		captureCtx, captureCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer captureCancel()
		output, _ := exec.CommandContext(captureCtx, tmux, "-S", socket, "capture-pane", "-p", "-S", "-1000", "-t", "catalog:0.0").CombinedOutput()
		return string(output)
	}
	defer func() { _ = os.WriteFile(filepath.Join(artifacts, "terminal-final.txt"), []byte(capture()), 0o600) }()
	complete := false
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	var screen string
	for !complete {
		select {
		case <-finished:
			complete = true
		case <-ctx.Done():
			t.Fatalf("native tmux did not complete Agent flow:\n%s", capture())
		case <-ticker.C:
			screen = capture()
			if strings.Contains(screen, "Yes, I accept") || strings.Contains(screen, "Yes, I trust this folder") {
				_ = exec.CommandContext(ctx, tmux, "-S", socket, "send-keys", "-t", "catalog:0.0", "Down", "Enter").Run()
			}
		}
	}
	// Wait for the received final response to appear in the real terminal UI.
	for !strings.Contains(screen, "catalog-main-finished") && ctx.Err() == nil {
		<-ticker.C
		screen = capture()
	}
	if ctx.Err() != nil {
		t.Fatalf("native final response did not reach terminal UI: %s", screen)
	}
	if !strings.Contains(screen, "gpt-8.2-sol") || !strings.Contains(screen, "gpt-8-luna") {
		t.Fatalf("real terminal did not expose resolved parent and Agent model labels: %s", screen)
	}
	if !strings.Contains(screen, "using Claude Code v"+version) {
		t.Fatalf("native terminal did not use installed verified Claude %s: %s", version, screen)
	}
	if err := os.WriteFile(filepath.Join(artifacts, "terminal.txt"), []byte(screen), 0o600); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	captured := append([]map[string]any(nil), requests...)
	mu.Unlock()
	var mainFast, nestedStandard bool
	for _, request := range captured {
		if request["model"] == "gpt-8.2-sol" && request["service_tier"] == "priority" {
			mainFast = true
		}
		if request["model"] == "gpt-8-luna" && request["service_tier"] == nil {
			nestedStandard = true
			assertCapturedReasoningEffort(t, request, "high")
		}
	}
	if !mainFast || !nestedStandard {
		t.Fatalf("native inherited speed flow mainFast=%t nestedStandard=%t requests=%d\n%s", mainFast, nestedStandard, len(captured), screen)
	}
	trace, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	fallback := bytes.Contains(trace, []byte(`"fast_mode_metadata_fallback":true`))
	result := map[string]any{"passed": true, "claude_version": version, "paid_model_requests": 0, "actual_agent_call": true, "parent_priority": mainFast, "unsupported_subagent_standard": nestedStandard, "inherited_fast_seen_by_proxy": fallback, "future_model_in_terminal": strings.Contains(screen, "gpt-8.2-sol")}
	for name, value := range map[string]any{"result.json": result, "requests.json": captured} {
		data, _ := json.MarshalIndent(value, "", "  ")
		if err := os.WriteFile(filepath.Join(artifacts, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("native tmux acceptance artifacts: %s; inherited fast reached proxy=%t", artifacts, fallback)
}
