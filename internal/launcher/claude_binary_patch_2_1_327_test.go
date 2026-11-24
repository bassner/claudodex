package launcher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

func TestClaude327PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.327", claude327SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.327 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.327", claude296SHA); got != nil {
		t.Fatalf("Claude 2.1.327 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.296", claude327SHA); got != nil {
		t.Fatalf("Claude 2.1.327 SHA matched wrong version: %#v", got)
	}
}

func TestClaude327WrongSHAFailsStrictSelection(t *testing.T) {
	claudePath := t.TempDir() + "/2.1.327"
	if err := os.WriteFile(claudePath, []byte("not the verified binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	var stderr strings.Builder
	_, err := selectClaudeExecutable(context.Background(), t.TempDir(), claudePath, "test", modelconfig.Default(), &stderr)
	if err == nil || !strings.Contains(err.Error(), "claude install "+claudeUIPatches[0].Version) {
		t.Fatalf("wrong-SHA source did not fail closed with an exact install command: %v", err)
	}
}

func TestClaude327AgentValidatorRejectsAmbiguousTargets(t *testing.T) {
	const target = `model:B(Z6a).optional()`
	for _, input := range []string{"missing", target + ";" + target} {
		data := []byte(input)
		if patchAgentModelValidator_2_1_327(data) || string(data) != input {
			t.Fatal("ambiguous agent validator changed")
		}
	}
	data := []byte(target)
	if !patchAgentModelValidator_2_1_327(data) || len(data) != len(target) {
		t.Fatal("unique agent validator failed to fit")
	}
}

func TestInstalledClaude327PatchTargets(t *testing.T) {
	if os.Getenv("CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE") != "1" {
		t.Skip("set CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 to run installed Claude smoke test")
	}
	path, err := exec.LookPath("claude")
	if err != nil {
		t.Fatal(err)
	}
	if version := detectClaudeVersion(context.Background(), path); version != "2.1.327" {
		if installedClaudeTargetsCoveredByLaterTest(version, "2.1.327") {
			t.Logf("installed Claude %s targets are covered by its version-specific test", version)
			return
		}
		if installedClaudeTargetsCoveredByLaterTest(version, "2.1.327") {
			t.Logf("installed Claude %s targets are covered by its version-specific test", version)
			return
		}
		t.Fatalf("installed Claude version = %s, want 2.1.327", version)
	}
	for name, check := range map[string]func(context.Context, string) bool{
		"settings":           claudeFastModeSettingsFallbackRequired,
		"picker-args":        claudeThreeTierPickerArgsRequired,
		"source-picker-args": claudeSourcePatchedPickerArgsRequired,
	} {
		if !check(context.Background(), path) {
			t.Fatalf("installed Claude is missing its %s compatibility", name)
		}
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := sha256Hex(source); got != claude327SHA {
		t.Fatalf("maintenance Claude SHA = %s, want %s", got, claude327SHA)
	}
	for _, transformation := range claude327Transformations("test") {
		t.Run(transformation.name, func(t *testing.T) {
			candidate := append([]byte(nil), source...)
			if !transformation.apply(candidate) {
				t.Fatalf("%s target does not match Claude 2.1.327", transformation.name)
			}
		})
	}
	if !validateClaude209UIBrandingReplacements(source, claude327UIBrandingReplacements) {
		for _, replacement := range claude327UIBrandingReplacements {
			if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
				t.Errorf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
			}
		}
		t.FailNow()
	}
	records, hashes, ok := claude281EmbeddedBunModuleHashes(source)
	if !ok {
		t.Fatal("Claude 2.1.327 Bun module table is unavailable")
	}
	patched := append([]byte(nil), source...)
	if !applyClaudeUIPatches_2_1_327(patched, "test", "2.1.327", modelconfig.Default()) {
		t.Fatal("complete Claude 2.1.327 patch did not apply")
	}
	start := bytes.Index(patched, []byte("function CDX327("))
	if start < 0 {
		t.Fatal("patched picker marker is absent")
	}
	end := bytes.Index(patched[start:], []byte("function vE("))
	if end < 0 || strings.Count(string(patched[start:start+end]), `r("`) != 4 {
		t.Fatal("patched picker does not contain exactly four tiers")
	}
	for _, forbidden := range []string{`r("mythos",`, "Mythos 5"} {
		if bytes.Contains(patched[start:start+end], []byte(forbidden)) {
			t.Fatalf("patched picker retained forbidden fourth-tier marker %q", forbidden)
		}
	}
	if !patchClaudeCatalogRecognition(patched, claudeSource{Version: "2.1.327", SHA256: claude327SHA}) {
		t.Fatal("catalog overlay did not apply")
	}
	if len(patched) != len(source) {
		t.Fatal("patch changed binary size")
	}
	changedModules, parsedModules := 0, 0
	for index, record := range records {
		current := sha256.Sum256(patched[record.contentOffset : record.contentOffset+record.contentLength])
		if current == hashes[index] {
			continue
		}
		changedModules++
		// Branding also changes embedded Markdown assets. Parse each changed
		// JavaScript module, while retaining bytecode invalidation for all assets.
		content := patched[record.contentOffset : record.contentOffset+record.contentLength]
		if bytes.HasPrefix(content, []byte("// @bun")) {
			parsedModules++
			node, err := exec.LookPath("node")
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(node, "--input-type=module", "--check")
			cmd.Stdin = bytes.NewReader(content)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("changed Bun module %d has invalid syntax: %v: %s", index, err, output)
			}
		}
		if bytecodeLength := binary.LittleEndian.Uint32(patched[record.bytecodeLength : record.bytecodeLength+4]); bytecodeLength != 0 {
			t.Errorf("changed Bun module %d retained %d bytes of stale bytecode", index, bytecodeLength)
		}
	}
	if parsedModules < 2 {
		t.Fatalf("parsed JavaScript module count = %d, want multiple patched modules", parsedModules)
	}
	if changedModules < 2 {
		t.Fatalf("changed Bun module count = %d, want multiple patched modules", changedModules)
	}
	broken := append([]byte(nil), source...)
	copy(broken[bytes.Index(source, claude327RequiredLogoAnchor()):], "function MISSING_TARGET(){")
	if applyClaudeUIPatches_2_1_327(broken, "test", "2.1.327", modelconfig.Default()) {
		t.Fatal("patch succeeded without the required logo transformation")
	}
}

func TestClaude327DefaultModelCursorMatchesResolvedModel(t *testing.T) {
	picker := []byte(`function $Xe(e=!1,n=null){` + strings.Repeat(" ", 5000) + `}function vE(e,n){}function HHa(e,n){` + strings.Repeat(" ", 500) + `}function Am(){return!(Fr()||{}).availableModels}`)
	if !patchModelPickerOptions_2_1_327(picker, modelconfig.Default()) || !patchModelPickerSelectionValue_2_1_327(picker) {
		t.Fatal("picker fixture did not patch")
	}
	focus := []byte(`function EF(h,R){return R===null?SP:HHa(h,R)??R}`)
	if !patchModelPickerDefaultFocus_2_1_327(focus) {
		t.Fatal("default cursor target did not patch")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable for picker behavior check")
	}
	script := `const s={};` + string(picker) + string(focus) + `;let SP="__NO_PREFERENCE__",effective="gpt-6.1-terra";function it(){return effective}function focused(model){return EF($Xe(),model)}if(focused(null)!=="sonnet")throw Error("default Sonnet cursor starts elsewhere");if(focused("gpt-6.1-terra[1m]")!=="sonnet")throw Error("explicit Sonnet mapping changed");if(focused("fable")!=="fable")throw Error("explicit Fable mapping changed");effective="gpt-6-astra";if(focused(null)!=="fable")throw Error("default Fable cursor starts elsewhere");if(focused("unknown")!=="unknown")throw Error("unknown fallback changed");`
	if output, err := exec.Command(node, "-e", script).CombinedOutput(); err != nil {
		t.Fatalf("default cursor behavior: %v: %s", err, output)
	}
}

func TestClaude327PickerLabelsAndOverrides(t *testing.T) {
	cfg := modelconfig.Config{Opus: "custom-sol", Sonnet: "custom-terra", Haiku: "custom-luna", Fable: "custom-astra"}
	data := []byte(`function $Xe(e=!1,n=null){` + strings.Repeat(" ", 5000) + `}function vE(e,n){}`)
	size := len(data)
	if !patchModelPickerOptions_2_1_327(data, cfg) || len(data) != size {
		t.Fatal("picker override changed binary size or failed to fit")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	script := `const s={ANTHROPIC_DEFAULT_OPUS_MODEL_NAME:"live-sol",ANTHROPIC_DEFAULT_SONNET_MODEL_NAME:"live-terra",ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME:"live-luna",ANTHROPIC_DEFAULT_FABLE_MODEL_NAME:"live-astra"};` + string(data) + `;const rows=$Xe();if(JSON.stringify(rows.map(x=>[x.value,x.label,x.description]))!==JSON.stringify([["opus","Opus","live-sol"],["sonnet","Sonnet","live-terra"],["haiku","Haiku","live-luna"],["fable","Fable","live-astra"]]))throw Error("four live role labels changed");for(const [model,role] of [["custom-sol","opus"],["custom-terra","sonnet"],["custom-luna","haiku"],["custom-astra","fable"]])if(CDX327(model)!==role)throw Error("override lost");for(const key of Object.keys(s))delete s[key];if(JSON.stringify($Xe().map(x=>x.description))!==JSON.stringify(["custom-sol","custom-terra","custom-luna","custom-astra"]))throw Error("configured fallback labels lost");`
	if output, err := exec.Command(node, "-e", script).CombinedOutput(); err != nil {
		t.Fatalf("picker behavior: %v: %s", err, output)
	}
	original := []byte(`function $Xe(e=!1,n=null){return []}function vE(e,n){}`)
	overflow := append([]byte(nil), original...)
	cfg.Opus = strings.Repeat("x", 10000)
	if patchModelPickerOptions_2_1_327(overflow, cfg) || !bytes.Equal(overflow, original) {
		t.Fatal("overflowing picker replacement did not fail without modifying bytes")
	}
}
