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
		"--model", "claude-opus-5",
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
	if captured["model"] != "gpt-5.6-sol" {
		t.Fatalf("upstream model = %#v, want gpt-5.6-sol; request=%#v", captured["model"], captured)
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
	wants := []string{
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"smoke using Claude Code v" + claudeVersion,
		"Set the AI model for Claudodex",
		"Codex Plan",
	}
	switch claudeVersion {
	case "2.1.251":
		wants = append(wants, "function gH(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}")
	case "2.1.247":
		wants = append(wants, "function f(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}")
	case "2.1.246":
		wants = append(wants, "function f(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}")
	case "2.1.245":
		wants = append(wants, "function f(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}")
	case "2.1.234":
		wants = append(wants, "function k1e(){return V.CLAUDE_BRIDGE_OAUTH_TOKEN}")
	case "2.1.233":
		wants = append(wants, "function sYe(){return V.CLAUDE_BRIDGE_OAUTH_TOKEN}")
	case "2.1.229":
		wants = append(wants, "function _8e(){return Q.CLAUDE_BRIDGE_OAUTH_TOKEN}")
	case "2.1.228":
		wants = append(wants, "function cGe(){return X.CLAUDE_BRIDGE_OAUTH_TOKEN}")
	case "2.1.227":
		wants = append(wants, "function f5e(){return re.CLAUDE_BRIDGE_OAUTH_TOKEN}")
	case "2.1.226":
		wants = append(wants, "function eje(){return te.CLAUDE_BRIDGE_OAUTH_TOKEN}")
	case "2.1.223":
		wants = append(wants, "function u9e(){return te.CLAUDE_BRIDGE_OAUTH_TOKEN}")
	case "2.1.222":
		wants = append(wants, "function F4e(){return te.CLAUDE_BRIDGE_OAUTH_TOKEN}")
	case "2.1.221":
		wants = append(wants, "function V4e(){return re.CLAUDE_BRIDGE_OAUTH_TOKEN}")
	case "2.1.218":
		wants = append(wants, "function iFe(){return Z.CLAUDE_BRIDGE_OAUTH_TOKEN}")
	case "2.1.219":
		wants = append(wants, "function q2e(){return Z.CLAUDE_BRIDGE_OAUTH_TOKEN}")
	case "2.1.220":
		wants = append(wants, "function q2e(){return Z.CLAUDE_BRIDGE_OAUTH_TOKEN}")
	}
	for _, want := range wants {
		if !bytes.Contains(data, []byte(want)) {
			t.Fatalf("patched installed Claude missing %q for version=%s sha=%s", want, claudeVersion, sourceSHA)
		}
	}
	if claudeVersion == "2.1.216" || claudeVersion == "2.1.218" || claudeVersion == "2.1.219" || claudeVersion == "2.1.220" || claudeVersion == "2.1.221" || claudeVersion == "2.1.222" || claudeVersion == "2.1.223" || claudeVersion == "2.1.226" || claudeVersion == "2.1.227" || claudeVersion == "2.1.228" || claudeVersion == "2.1.229" || claudeVersion == "2.1.233" || claudeVersion == "2.1.234" || claudeVersion == "2.1.245" || claudeVersion == "2.1.246" || claudeVersion == "2.1.247" || claudeVersion == "2.1.251" {
		normalizer := "function CDX216("
		pickerEnd := "function tAe("
		switch claudeVersion {
		case "2.1.251":
			normalizer = "function CDX251("
			pickerEnd = "function S("
		case "2.1.247":
			normalizer = "function CDX247("
			pickerEnd = "function h("
		case "2.1.246":
			normalizer = "function CDX246("
			pickerEnd = "function h("
		case "2.1.245":
			normalizer = "function CDX245("
			pickerEnd = "function h("
		case "2.1.234":
			normalizer = "function CDX234("
			pickerEnd = "function GOe("
		case "2.1.233":
			normalizer = "function CDX233("
			pickerEnd = "function UPe("
		case "2.1.229":
			normalizer = "function CDX229("
			pickerEnd = "function lxe("
		case "2.1.228":
			normalizer = "function CDX228("
			pickerEnd = "function ixe("
		case "2.1.227":
			normalizer = "function CDX227("
			pickerEnd = "function bAe("
		case "2.1.226":
			normalizer = "function CDX226("
			pickerEnd = "function Kwe("
		case "2.1.223":
			normalizer = "function CDX223("
			pickerEnd = "function Cve("
		case "2.1.222":
			normalizer = "function CDX222("
			pickerEnd = "function QTe("
		case "2.1.221":
			normalizer = "function CDX221("
			pickerEnd = "function wTe("
		case "2.1.218":
			normalizer = "function CDX218("
			pickerEnd = "function vRe("
		case "2.1.219":
			normalizer = "function CDX219("
			pickerEnd = "function Fye("
		case "2.1.220":
			normalizer = "function CDX220("
			pickerEnd = "function Fye("
		}
		pickerStart := bytes.Index(data, []byte(normalizer))
		if pickerStart < 0 {
			t.Fatalf("patched installed Claude missing %s model picker normalizer", claudeVersion)
		}
		pickerEndRel := bytes.Index(data[pickerStart:], []byte(pickerEnd))
		if pickerEndRel < 0 {
			t.Fatalf("patched installed Claude missing %s model picker end marker", claudeVersion)
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
	case "2.1.251":
		brandingReplacements = claude251UIBrandingReplacements
	case "2.1.247":
		brandingReplacements = claude247UIBrandingReplacements
	case "2.1.246":
		brandingReplacements = claude246UIBrandingReplacements
	case "2.1.245":
		brandingReplacements = claude245UIBrandingReplacements
	case "2.1.234":
		brandingReplacements = claude234UIBrandingReplacements
	case "2.1.233":
		brandingReplacements = claude233UIBrandingReplacements
	case "2.1.229":
		brandingReplacements = claude229UIBrandingReplacements
	case "2.1.228":
		brandingReplacements = claude228UIBrandingReplacements
	case "2.1.227":
		brandingReplacements = claude227UIBrandingReplacements
	case "2.1.226":
		brandingReplacements = claude226UIBrandingReplacements
	case "2.1.223":
		brandingReplacements = claude223UIBrandingReplacements
	case "2.1.222":
		brandingReplacements = claude222UIBrandingReplacements
	case "2.1.221":
		brandingReplacements = claude221UIBrandingReplacements
	case "2.1.209":
		brandingReplacements = claude209UIBrandingReplacements
	case "2.1.211":
		brandingReplacements = claude211UIBrandingReplacements
	case "2.1.212":
		brandingReplacements = claude212UIBrandingReplacements
	case "2.1.216":
		brandingReplacements = claude216UIBrandingReplacements
	case "2.1.218":
		brandingReplacements = claude218UIBrandingReplacements
	case "2.1.219":
		brandingReplacements = claude219UIBrandingReplacements
	case "2.1.220":
		brandingReplacements = claude220UIBrandingReplacements
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

func installedClaudeTargetsCoveredByLaterTest(version, target string) bool {
	order := map[string]int{
		"2.1.220": 0,
		"2.1.221": 1,
		"2.1.222": 2,
		"2.1.223": 3,
		"2.1.226": 4,
		"2.1.227": 5,
		"2.1.228": 6,
		"2.1.229": 7,
		"2.1.233": 8,
		"2.1.234": 9,
		"2.1.245": 10,
		"2.1.246": 11,
		"2.1.247": 12,
		"2.1.251": 13,
	}
	return order[version] > order[target]
}

func TestInstalledClaude220PatchTargets(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		t.Skipf("claude binary not available: %v", err)
	}
	if version := detectClaudeVersion(context.Background(), claudePath); version != "2.1.220" {
		if installedClaudeTargetsCoveredByLaterTest(version, "2.1.220") {
			t.Logf("installed Claude %s targets are covered by its version-specific test", version)
			return
		}
		t.Fatalf("installed Claude version = %s, want a registered supported version", version)
	}
	source, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}

	transformations := []struct {
		name  string
		apply func([]byte) bool
	}{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_220(data, "test", "2.1.220") }},
		{"whats-new", patchWhatsNewFeedFunction_2_1_220},
		{"usage", patchUsageFetchFunction_2_1_220},
		{"model-options", patchModelPickerOptions_2_1_220},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_220},
		{"model-selection", patchModelPickerSelectionValue_2_1_220},
		{"agent-model-validator", patchAgentModelValidator_2_1_220},
		{"fast-mode-gate", func(data []byte) bool {
			return replaceFirstFixed(data, `function El(){if(xn()!=="firstParty")return!1;return!Z.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function El(){return!Z.CLAUDE_CODE_DISABLE_FAST_MODE}`)
		}},
		{"fast-mode-name", func(data []byte) bool {
			return replaceFirstFixed(data, `function pq(){return"Opus 5"}`, `function pq(){return"Codex"}`)
		}},
		{"fast-mode-model", func(data []byte) bool {
			return replaceFirstFixed(data, `function jkt(){return"opus"+(YM()?"[1m]":"")}`, `function jkt(){return"opus"}`)
		}},
		{"fast-mode-support", func(data []byte) bool {
			return replaceFirstFixed(data, `function fE(e){if(!El())return!1;let t=e??ZN(),r=Ei(t);if(LN(lo(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-7")||n.includes("opus-4-8")||n.includes("opus-5")}`, `function fE(e){return El()}`)
		}},
		{"fast-mode-pricing", patchFastModePricing_2_1_220},
		{"context-warning", patchContextWarningHint_2_1_220},
		{"resume-hints", patchResumeCommandHints_2_1_220},
		{"compact-progress", patchCompactProgressCurve_2_1_220},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_220},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude220UIBrandingReplacements)
		}},
	}
	for _, transformation := range transformations {
		t.Run(transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Fatalf("%s patch target did not match installed Claude 2.1.220", transformation.name)
			}
		})
	}
	for _, replacement := range claude220UIBrandingReplacements {
		if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
			t.Errorf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
		}
	}
}

func TestInstalledClaude221PatchTargets(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		t.Fatalf("claude binary not available: %v", err)
	}
	if version := detectClaudeVersion(context.Background(), claudePath); version != "2.1.221" {
		if installedClaudeTargetsCoveredByLaterTest(version, "2.1.221") {
			t.Logf("installed Claude %s targets are covered by its version-specific test", version)
			return
		}
		t.Fatalf("installed Claude version = %s, want a registered supported version", version)
	}
	source, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}

	transformations := []struct {
		name  string
		apply func([]byte) bool
	}{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_221(data, "test", "2.1.221") }},
		{"whats-new", patchWhatsNewFeedFunction_2_1_221},
		{"usage", patchUsageFetchFunction_2_1_221},
		{"model-options", patchModelPickerOptions_2_1_221},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_221},
		{"model-selection", patchModelPickerSelectionValue_2_1_221},
		{"agent-model-validator", patchAgentModelValidator_2_1_221},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_221},
		{"fast-mode-pricing", patchFastModePricing_2_1_221},
		{"context-warning", patchContextWarningHint_2_1_221},
		{"resume-hints", patchResumeCommandHints_2_1_221},
		{"compact-progress", patchCompactProgressCurve_2_1_221},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_221},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude221UIBrandingReplacements)
		}},
	}
	for _, transformation := range transformations {
		t.Run(transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Fatalf("%s patch target did not match installed Claude 2.1.221", transformation.name)
			}
		})
	}
	for _, replacement := range claude221UIBrandingReplacements {
		if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
			t.Errorf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
		}
	}
}

func TestInstalledClaude222PatchTargets(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		t.Fatalf("claude binary not available: %v", err)
	}
	if version := detectClaudeVersion(context.Background(), claudePath); version != "2.1.222" {
		if installedClaudeTargetsCoveredByLaterTest(version, "2.1.222") {
			t.Logf("installed Claude %s targets are covered by its version-specific test", version)
			return
		}
		t.Fatalf("installed Claude version = %s, want 2.1.222", version)
	}
	source, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}

	transformations := []struct {
		name  string
		apply func([]byte) bool
	}{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_222(data, "test", "2.1.222") }},
		{"whats-new", patchWhatsNewFeedFunction_2_1_222},
		{"usage", patchUsageFetchFunction_2_1_222},
		{"model-options", patchModelPickerOptions_2_1_222},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_222},
		{"model-selection", patchModelPickerSelectionValue_2_1_222},
		{"agent-model-validator", patchAgentModelValidator_2_1_222},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_222},
		{"fast-mode-pricing", patchFastModePricing_2_1_222},
		{"context-warning", patchContextWarningHint_2_1_222},
		{"resume-hints", patchResumeCommandHints_2_1_222},
		{"compact-progress", patchCompactProgressCurve_2_1_222},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_222},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude222UIBrandingReplacements)
		}},
	}
	for _, transformation := range transformations {
		t.Run(transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Fatalf("%s patch target did not match installed Claude 2.1.222", transformation.name)
			}
		})
	}
	for _, replacement := range claude222UIBrandingReplacements {
		if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
			t.Errorf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
		}
	}
}

func TestInstalledClaude223PatchTargets(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		t.Fatalf("claude binary not available: %v", err)
	}
	if version := detectClaudeVersion(context.Background(), claudePath); version != "2.1.223" {
		if installedClaudeTargetsCoveredByLaterTest(version, "2.1.223") {
			t.Logf("installed Claude %s targets are covered by its version-specific test", version)
			return
		}
		t.Fatalf("installed Claude version = %s, want 2.1.223", version)
	}
	source, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}

	transformations := []struct {
		name  string
		apply func([]byte) bool
	}{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_223(data, "test", "2.1.223") }},
		{"whats-new", patchWhatsNewFeedFunction_2_1_223},
		{"usage", patchUsageFetchFunction_2_1_223},
		{"model-options", patchModelPickerOptions_2_1_223},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_223},
		{"model-selection", patchModelPickerSelectionValue_2_1_223},
		{"agent-model-validator", patchAgentModelValidator_2_1_223},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_223},
		{"fast-mode-pricing", patchFastModePricing_2_1_223},
		{"context-warning", patchContextWarningHint_2_1_223},
		{"resume-hints", patchResumeCommandHints_2_1_223},
		{"compact-progress", patchCompactProgressCurve_2_1_223},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_223},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude223UIBrandingReplacements)
		}},
	}
	for _, transformation := range transformations {
		t.Run(transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Fatalf("%s patch target did not match installed Claude 2.1.223", transformation.name)
			}
		})
	}
	for _, replacement := range claude223UIBrandingReplacements {
		if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
			t.Errorf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
		}
	}
}

func TestInstalledClaude226PatchTargets(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		t.Fatalf("claude binary not available: %v", err)
	}
	if version := detectClaudeVersion(context.Background(), claudePath); version != "2.1.226" {
		if installedClaudeTargetsCoveredByLaterTest(version, "2.1.226") {
			t.Logf("installed Claude %s targets are covered by its version-specific test", version)
			return
		}
		t.Fatalf("installed Claude version = %s, want 2.1.226", version)
	}
	source, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, transformation := range claude226Transformations("test") {
		t.Run(transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Fatalf("%s patch target did not match installed Claude 2.1.226", transformation.name)
			}
		})
	}
	for _, replacement := range claude226UIBrandingReplacements {
		if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
			t.Errorf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
		}
	}
}

func TestInstalledClaude227PatchTargets(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		t.Fatalf("claude binary not available: %v", err)
	}
	if version := detectClaudeVersion(context.Background(), claudePath); version != "2.1.227" {
		if installedClaudeTargetsCoveredByLaterTest(version, "2.1.227") {
			t.Logf("installed Claude %s targets are covered by its version-specific test", version)
			return
		}
		t.Fatalf("installed Claude version = %s, want 2.1.227", version)
	}
	source, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, transformation := range claude227Transformations("test") {
		t.Run(transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Fatalf("%s patch target did not match installed Claude 2.1.227", transformation.name)
			}
		})
	}
	for _, replacement := range claude227UIBrandingReplacements {
		if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
			t.Errorf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
		}
	}
}

func TestInstalledClaude228PatchTargets(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		t.Fatalf("claude binary not available: %v", err)
	}
	if version := detectClaudeVersion(context.Background(), claudePath); version != "2.1.228" {
		if installedClaudeTargetsCoveredByLaterTest(version, "2.1.228") {
			t.Logf("installed Claude %s targets are covered by its version-specific test", version)
			return
		}
		t.Fatalf("installed Claude version = %s, want 2.1.228", version)
	}
	source, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, transformation := range claude228Transformations("test") {
		t.Run(transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Fatalf("%s patch target did not match installed Claude 2.1.228", transformation.name)
			}
		})
	}
	for _, replacement := range claude228UIBrandingReplacements {
		if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
			t.Errorf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
		}
	}
}

func TestInstalledClaude229PatchTargets(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		t.Fatalf("claude binary not available: %v", err)
	}
	if version := detectClaudeVersion(context.Background(), claudePath); version != "2.1.229" {
		if installedClaudeTargetsCoveredByLaterTest(version, "2.1.229") {
			t.Logf("installed Claude %s targets are covered by its version-specific test", version)
			return
		}
		t.Fatalf("installed Claude version = %s, want 2.1.229", version)
	}
	source, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, transformation := range claude229Transformations("test") {
		t.Run(transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Fatalf("%s patch target did not match installed Claude 2.1.229", transformation.name)
			}
		})
	}
	for _, replacement := range claude229UIBrandingReplacements {
		if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
			t.Errorf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
		}
	}
}

func TestInstalledClaude233PatchTargets(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		t.Fatalf("claude binary not available: %v", err)
	}
	if version := detectClaudeVersion(context.Background(), claudePath); version != "2.1.233" {
		if installedClaudeTargetsCoveredByLaterTest(version, "2.1.233") {
			t.Logf("installed Claude %s targets are covered by its version-specific test", version)
			return
		}
		t.Fatalf("installed Claude version = %s, want 2.1.233", version)
	}
	source, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, transformation := range claude233Transformations("test") {
		t.Run(transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Fatalf("%s patch target did not match installed Claude 2.1.233", transformation.name)
			}
		})
	}
	for _, replacement := range claude233UIBrandingReplacements {
		if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
			t.Errorf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
		}
	}
}

func TestInstalledClaude234PatchTargets(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		t.Fatalf("claude binary not available: %v", err)
	}
	if version := detectClaudeVersion(context.Background(), claudePath); version != "2.1.234" {
		if installedClaudeTargetsCoveredByLaterTest(version, "2.1.234") {
			t.Logf("installed Claude %s targets are covered by its version-specific test", version)
			return
		}
		t.Fatalf("installed Claude version = %s, want 2.1.234", version)
	}
	source, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, transformation := range claude234Transformations("test") {
		t.Run(transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Fatalf("%s patch target did not match installed Claude 2.1.234", transformation.name)
			}
		})
	}
	for _, replacement := range claude234UIBrandingReplacements {
		if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
			t.Errorf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
		}
	}
}

func TestInstalledClaude245PatchTargets(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		t.Fatalf("claude binary not available: %v", err)
	}
	if version := detectClaudeVersion(context.Background(), claudePath); version != "2.1.245" {
		if installedClaudeTargetsCoveredByLaterTest(version, "2.1.245") {
			t.Logf("installed Claude %s targets are covered by its version-specific test", version)
			return
		}
		t.Fatalf("installed Claude version = %s, want a registered supported version", version)
	}
	source, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, transformation := range claude245Transformations("test") {
		t.Run(transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Fatalf("%s patch target did not match installed Claude 2.1.245", transformation.name)
			}
		})
	}
	for _, replacement := range claude245UIBrandingReplacements {
		if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
			t.Errorf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
		}
	}
}

func TestInstalledClaude246PatchTargets(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		t.Fatalf("claude binary not available: %v", err)
	}
	if version := detectClaudeVersion(context.Background(), claudePath); version != "2.1.246" {
		if installedClaudeTargetsCoveredByLaterTest(version, "2.1.246") {
			t.Logf("installed Claude %s targets are covered by its version-specific test", version)
			return
		}
		t.Fatalf("installed Claude version = %s, want a registered supported version", version)
	}
	source, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, transformation := range claude246Transformations("test") {
		t.Run(transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Fatalf("%s patch target did not match installed Claude 2.1.246", transformation.name)
			}
		})
	}
	for _, replacement := range claude246UIBrandingReplacements {
		if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
			t.Errorf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
		}
	}
}

func TestInstalledClaude247PatchTargets(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		t.Fatalf("claude binary not available: %v", err)
	}
	if version := detectClaudeVersion(context.Background(), claudePath); version != "2.1.247" {
		if installedClaudeTargetsCoveredByLaterTest(version, "2.1.247") {
			t.Logf("installed Claude %s targets are covered by its version-specific test", version)
			return
		}
		t.Fatalf("installed Claude version = %s, want 2.1.247", version)
	}
	source, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, transformation := range claude247Transformations("test") {
		t.Run(transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Fatalf("%s patch target did not match installed Claude 2.1.247", transformation.name)
			}
		})
	}
	for _, replacement := range claude247UIBrandingReplacements {
		if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
			t.Errorf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
		}
	}
}

func TestInstalledClaude251PatchTargets(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		t.Fatalf("claude binary not available: %v", err)
	}
	if version := detectClaudeVersion(context.Background(), claudePath); version != "2.1.251" {
		t.Fatalf("installed Claude version = %s, want 2.1.251", version)
	}
	source, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, transformation := range claude251Transformations("test") {
		t.Run(transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Fatalf("%s patch target did not match installed Claude 2.1.251", transformation.name)
			}
		})
	}
	for _, replacement := range claude251UIBrandingReplacements {
		if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
			t.Errorf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
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
		t.Fatalf("service_tier = %#v, want priority", captured["service_tier"])
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
