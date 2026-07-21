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
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bassner/claudodex/internal/modelconfig"
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
			_, _ = io.WriteString(w, `{"models":[{"slug":"gpt-5.6-sol","context_window":372000,"supports_reasoning_summaries":true},{"slug":"gpt-5.6-terra","context_window":372000,"supports_reasoning_summaries":true},{"slug":"gpt-5.6-luna","context_window":372000,"supports_reasoning_summaries":true}]}`)
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
			`event: response.reasoning_summary_part.added`,
			`data: {"type":"response.reasoning_summary_part.added","item_id":"reasoning_smoke","output_index":0,"summary_index":0,"part":{"type":"summary_text","text":""}}`,
			``,
			`event: response.reasoning_summary_text.delta`,
			`data: {"type":"response.reasoning_summary_text.delta","item_id":"reasoning_smoke","output_index":0,"summary_index":0,"delta":"OpenAI summary smoke"}`,
			``,
			`event: response.reasoning_summary_part.added`,
			`data: {"type":"response.reasoning_summary_part.added","item_id":"reasoning_smoke","output_index":0,"summary_index":1,"part":{"type":"summary_text","text":""}}`,
			``,
			`event: response.reasoning_summary_text.delta`,
			`data: {"type":"response.reasoning_summary_text.delta","item_id":"reasoning_smoke","output_index":0,"summary_index":1,"delta":"**Second summary section****Third summary section**"}`,
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
		"--output-format", "stream-json",
		"--verbose",
		"--include-partial-messages",
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
	if strings.Count(stdout.String(), `"type":"thinking"`) < 3 || !strings.Contains(stdout.String(), `OpenAI summary smoke`) || !strings.Contains(stdout.String(), `**Second summary section**`) || !strings.Contains(stdout.String(), `**Third summary section**`) {
		t.Fatalf("installed Claude did not accept and expose separate native thinking blocks\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if strings.Contains(stderr.String(), "no verified UI patch") || strings.Contains(stderr.String(), "unpatched Claude Code UI") {
		t.Fatalf("installed Claude launched without its verified UI patch\nstderr:\n%s", stderr.String())
	}
	if captured["model"] != "gpt-5.6-terra" {
		t.Fatalf("upstream model = %#v, want gpt-5.6-terra; request=%#v", captured["model"], captured)
	}
	assertCapturedReasoningEffort(t, captured, "max")
	assertCapturedReasoningSummary(t, captured, "auto")
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

func TestInstalledClaudeUIPatchSmoke(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		t.Skipf("claude binary not available: %v", err)
	}

	home := t.TempDir()
	claudeVersion, sourceSHA := requireInstalledClaudeUIPatch(t, claudePath)
	patched, claudeVersion, sourceSHA, err := preparePatchedClaude(context.Background(), home, claudePath, "smoke", modelconfig.Default())
	if err != nil {
		t.Fatalf("prepare patched installed Claude failed for version=%s sha=%s: %v", claudeVersion, sourceSHA, err)
	}
	if patched == claudePath {
		t.Fatalf("patched path = source path %q", patched)
	}
	data, err := os.ReadFile(patched)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"smoke using Claude Code v" + claudeVersion,
		"Set the AI model for Claudodex",
		"Codex Plan",
	} {
		if !bytes.Contains(data, []byte(want)) {
			t.Fatalf("patched installed Claude missing %q for version=%s sha=%s", want, claudeVersion, sourceSHA)
		}
	}
	if claudeVersion == "2.1.216" {
		pickerStart := bytes.Index(data, []byte("function CDX216("))
		if pickerStart < 0 {
			t.Fatal("patched installed Claude missing 2.1.216 model picker normalizer")
		}
		pickerEndRel := bytes.Index(data[pickerStart:], []byte("function tAe("))
		if pickerEndRel < 0 {
			t.Fatal("patched installed Claude missing 2.1.216 model picker end marker")
		}
		picker := data[pickerStart : pickerStart+pickerEndRel]
		if tiers := bytes.Count(picker, []byte(`n("`)); tiers != 3 {
			t.Fatalf("patched installed Claude picker tier count = %d, want 3", tiers)
		}
		for _, want := range []string{`n("opus",`, `n("sonnet",`, `n("haiku",`, "gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"} {
			if !bytes.Contains(picker, []byte(want)) {
				t.Fatalf("patched installed Claude picker missing %q", want)
			}
		}
		for _, forbidden := range []string{"fable", "Fable", "mythos", "Mythos", "ANTHROPIC_DEFAULT_FABLE_MODEL"} {
			if bytes.Contains(picker, []byte(forbidden)) {
				t.Fatalf("patched installed Claude picker retained forbidden fourth-tier marker %q", forbidden)
			}
		}
	}
	versionOutput, err := exec.Command(patched, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("patched installed Claude did not launch: %v\noutput:\n%s", err, versionOutput)
	}
	if !bytes.Contains(versionOutput, []byte(claudeVersion)) {
		t.Fatalf("patched installed Claude version output = %q, want version %s", versionOutput, claudeVersion)
	}
	for _, parseFailure := range []string{"Bun", "SyntaxError", "JavaScript parse error"} {
		if bytes.Contains(versionOutput, []byte(parseFailure)) {
			t.Fatalf("patched installed Claude reported parse failure %q: %s", parseFailure, versionOutput)
		}
	}
	if runtime.GOOS == "darwin" {
		if output, err := exec.Command("codesign", "--verify", "--deep", "--strict", "--verbose=2", patched).CombinedOutput(); err != nil {
			t.Fatalf("patched installed Claude has invalid code signature: %v\noutput:\n%s", err, output)
		}
		signatureOutput, err := exec.Command("codesign", "-dvv", patched).CombinedOutput()
		if err != nil {
			t.Fatalf("could not inspect patched installed Claude signature: %v\noutput:\n%s", err, signatureOutput)
		}
		if !bytes.Contains(signatureOutput, []byte("Signature=adhoc")) {
			t.Fatalf("patched installed Claude signature is not ad-hoc:\n%s", signatureOutput)
		}
	}
	var brandingReplacements []claude209UIBrandingReplacement
	switch claudeVersion {
	case "2.1.209":
		brandingReplacements = claude209UIBrandingReplacements
	case "2.1.211":
		brandingReplacements = claude211UIBrandingReplacements
	case "2.1.212":
		brandingReplacements = claude212UIBrandingReplacements
	case "2.1.216":
		brandingReplacements = claude216UIBrandingReplacements
	}
	for _, replacement := range brandingReplacements {
		if bytes.Contains(data, []byte(replacement.old)) {
			t.Fatalf("patched installed Claude retained %q for version=%s sha=%s", replacement.old, claudeVersion, sourceSHA)
		}
		if !bytes.Contains(data, []byte(replacement.replacement)) {
			t.Fatalf("patched installed Claude missing %q for version=%s sha=%s", replacement.replacement, claudeVersion, sourceSHA)
		}
	}
}

func TestInstalledClaudeFastModeSmokeWithFakeCodexUpstream(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		t.Skipf("claude binary not available: %v", err)
	}
	requireInstalledClaudeUIPatch(t, claudePath)

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
			_, _ = io.WriteString(w, `{"models":[{"slug":"gpt-5.6-sol","context_window":372000},{"slug":"gpt-5.6-terra","context_window":372000},{"slug":"gpt-5.6-luna","context_window":372000}]}`)
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
	err = (ProcessLauncher{}).Launch(ctx, []string{
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
	if captured["model"] != "gpt-5.6-sol" {
		t.Fatalf("upstream model = %#v, want gpt-5.6-sol; request=%#v", captured["model"], captured)
	}
	if captured["service_tier"] != "priority" {
		t.Fatalf("service_tier = %#v, want priority; request=%#v\nstdout:\n%s\nstderr:\n%s", captured["service_tier"], captured, stdout.String(), stderr.String())
	}
}

func requireInstalledClaudeUIPatch(t *testing.T, claudePath string) (string, string) {
	t.Helper()
	if strings.TrimSpace(os.Getenv("CLAUDODEX_DISABLE_CLAUDE_PATCH")) == "1" {
		t.Skip("installed Claude UI patch smoke requires CLAUDODEX_DISABLE_CLAUDE_PATCH unset")
	}
	claudeVersion := detectClaudeVersion(context.Background(), claudePath)
	sourceData, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	sourceSHA := sha256Hex(sourceData)
	if findClaudeUIPatch(claudeVersion, sourceSHA) == nil {
		t.Skipf("no verified installed Claude UI patch for version=%s sha=%s", claudeVersion, sourceSHA)
	}
	return claudeVersion, sourceSHA
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
			_, _ = io.WriteString(w, `{"models":[{"slug":"gpt-5.6-sol","context_window":372000},{"slug":"gpt-5.6-terra","context_window":372000},{"slug":"gpt-5.6-luna","context_window":372000}]}`)
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
	if captured["model"] != "gpt-5.6-terra" {
		t.Fatalf("upstream model = %#v, want gpt-5.6-terra; request=%#v", captured["model"], captured)
	}
	assertCapturedReasoningEffort(t, captured, "max")
}

func assertCapturedReasoningEffort(t *testing.T, captured map[string]any, want string) {
	t.Helper()
	reasoning, _ := captured["reasoning"].(map[string]any)
	if reasoning["effort"] != want {
		t.Fatalf("reasoning.effort = %#v, want %q; request=%#v", reasoning["effort"], want, captured)
	}
}

func assertCapturedReasoningSummary(t *testing.T, captured map[string]any, want string) {
	t.Helper()
	reasoning, _ := captured["reasoning"].(map[string]any)
	if reasoning["summary"] != want {
		t.Fatalf("reasoning.summary = %#v, want %q; request=%#v", reasoning["summary"], want, captured)
	}
}
