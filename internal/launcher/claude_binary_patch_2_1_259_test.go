package launcher

import (
	"bytes"
	"context"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude259SHA = "884baa38fe1a624be25c4a91568bf5a08b5cf4e7d7acf29b7760e3525d964898"

func TestClaude259PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.259", claude259SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.259 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.259", claude258SHA); got != nil {
		t.Fatalf("Claude 2.1.259 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.258", claude259SHA); got != nil {
		t.Fatalf("Claude 2.1.259 SHA matched wrong version: %#v", got)
	}
}

func TestClaude259WrongSHAFallsBackToUnpatchedExecutable(t *testing.T) {
	claudePath := t.TempDir() + "/2.1.259"
	if err := os.WriteFile(claudePath, []byte("not the verified binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	var stderr strings.Builder
	got := prepareClaudeExecutable(context.Background(), t.TempDir(), claudePath, "test", modelconfig.Default(), &stderr)
	if got != claudePath {
		t.Fatalf("unsupported executable path = %q, want original %q", got, claudePath)
	}
	if !strings.Contains(stderr.String(), "no verified UI patch") || !strings.Contains(stderr.String(), "sha256:") {
		t.Fatalf("unsupported fallback warning = %q", stderr.String())
	}
}

func TestClaude259ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function Ufn(e=!1){` + strings.Repeat(" ", 5000) + `function _ro(e,n){}`)
	if !patchModelPickerOptions_2_1_259(data, modelconfig.Default()) {
		t.Fatal("patchModelPickerOptions_2_1_259 reported no changes")
	}
	got := string(data)
	for _, want := range []string{`r("opus","Opus",`, `r("sonnet","Sonnet",`, `r("haiku","Haiku",`, "gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"} {
		if !strings.Contains(got, want) {
			t.Fatalf("model picker missing %q", want)
		}
	}
	if tiers := strings.Count(got, `r("`); tiers != 3 {
		t.Fatalf("model picker tier count = %d, want 3", tiers)
	}
	if !strings.Contains(got, `.replaceAll("[1m]","")`) {
		t.Fatal("model picker normalizer does not strip Claude's long-context suffix")
	}
	for _, forbidden := range []string{"fable", "Fable", "mythos", "Mythos", "ANTHROPIC_DEFAULT_FABLE_MODEL"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("model picker retained forbidden fourth-tier marker %q", forbidden)
		}
	}
}

func TestClaude259ModelPickerRejectsOverrideAndCustomRows(t *testing.T) {
	data := []byte(claude259ModelOptionsOverrideTarget + claude259ModelSelectionSourceTarget + claude259ModelExtraOptionsTarget + claude259ModelPickerValueTarget)
	if !patchModelPickerExtraOptions_2_1_259(data) {
		t.Fatal("model picker extra-options patch did not apply")
	}
	got := string(data)
	for _, want := range []string{`ct=K(()=>Ufn(rt),[rt])`, `Rt=Ye??w,pt=CDX259(Rt===null?Cw:Rt),`, `po=ct.slice(0,3)`, `options:wn.slice(0,3)`} {
		if !strings.Contains(got, want) {
			t.Fatalf("model picker missing %q", want)
		}
	}
	if strings.Contains(got, `selectedValue:`) {
		t.Fatal("model picker must not pass a selected value that the list can re-add as a custom fourth row")
	}
	for _, forbidden := range []string{"Current model", "Base model", "Pe??Ufn"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("model picker retained custom-row path %q", forbidden)
		}
	}
}

func TestClaude259ModelPickerResolverReturnsOnlyThreeCanonicalRows(t *testing.T) {
	data := []byte(`function wro(e,n){` + strings.Repeat(" ", 5000) + `function Bfn(e){}`)
	if !patchModelPickerResolver_2_1_259(data) {
		t.Fatal("model picker resolver patch did not apply")
	}
	if !strings.Contains(string(data), `function wro(e,n){return vTe(e).slice(0,3)}`) {
		t.Fatal("model picker resolver retained noncanonical option sources")
	}
}

func TestClaude259LogoPatchFailsClosedOnOverflow(t *testing.T) {
	data := []byte(`function x0e(){let l=a.DEMO_VERSION??` + strings.Repeat(" ", 1000) + `function LZt(l,b,O){}`)
	original := append([]byte(nil), data...)
	if patchLogoDisplayDataFunction_2_1_259(data, strings.Repeat("x", 4000), "2.1.259") {
		t.Fatal("oversized executable replacement unexpectedly succeeded")
	}
	if !bytes.Equal(data, original) {
		t.Fatal("overflowing executable replacement mutated the input")
	}
}

func TestClaude259DisablesStaleEmbeddedPatchedModuleBytecode(t *testing.T) {
	if version := os.Getenv("CLAUDODEX_MAINTENANCE_CLAUDE_VERSION"); version != "" && version != "2.1.259" {
		return
	}
	path := os.Getenv("CLAUDODEX_MAINTENANCE_CLAUDE_REALPATH")
	if path == "" {
		t.Skip("maintenance Claude path is unavailable")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	before, ok := claude259EmbeddedPatchedModuleBytecodeLength(data)
	if !ok || before == 0 {
		t.Fatalf("maintenance binary entry bytecode length = %d, found = %v", before, ok)
	}
	if !patchModelPickerOptions_2_1_259(data, modelconfig.Default()) {
		t.Fatal("model picker source prerequisite did not apply")
	}
	if !disableClaude259EmbeddedPatchedModuleBytecode(data) {
		t.Fatal("embedded entry bytecode was not disabled")
	}
	after, ok := claude259EmbeddedPatchedModuleBytecodeLength(data)
	if !ok || after != 0 {
		t.Fatalf("patched entry bytecode length = %d, found = %v; want zero", after, ok)
	}
}

func TestClaude259PatchTargetsMaintenanceBinary(t *testing.T) {
	if version := os.Getenv("CLAUDODEX_MAINTENANCE_CLAUDE_VERSION"); version != "" && version != "2.1.259" {
		return
	}
	path := os.Getenv("CLAUDODEX_MAINTENANCE_CLAUDE_REALPATH")
	if path == "" {
		t.Skip("maintenance Claude path is unavailable")
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, transformation := range claude259Transformations("test") {
		t.Run(transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Errorf("Claude 2.1.259 target %s does not match the maintenance binary", transformation.name)
			}
		})
	}
	for _, transformation := range claude259RemoteControlTransformations() {
		t.Run("remote-control/"+transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Errorf("Claude 2.1.259 remote-control target %s does not match the maintenance binary", transformation.name)
			}
		})
	}
	patched := append([]byte(nil), source...)
	if !applyClaudeUIPatches_2_1_259(patched, "test", "2.1.259", modelconfig.Default()) {
		t.Fatal("complete Claude 2.1.259 patch did not apply")
	}
	if tiers := claude259ModelPickerTierCount(patched); tiers != 3 {
		t.Fatalf("patched model picker tier count = %d, want 3", tiers)
	}
	for _, want := range []string{"Claudodex Info", "test using Claude Code v2.1.259", "function p0(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}"} {
		if !bytes.Contains(patched, []byte(want)) {
			t.Fatalf("complete patch missing %q", want)
		}
	}
	for _, forbidden := range []string{`r("fable",`, `r("mythos",`, "ANTHROPIC_DEFAULT_FABLE_MODEL"} {
		start := bytes.Index(patched, []byte("function CDX259("))
		end := bytes.Index(patched[start:], []byte("function _ro("))
		if start < 0 || end < 0 {
			t.Fatal("patched model picker bounds are missing")
		}
		if bytes.Contains(patched[start:start+end], []byte(forbidden)) {
			t.Fatalf("patched picker retained forbidden fourth-tier marker %q", forbidden)
		}
	}
	broken := bytes.Replace(append([]byte(nil), source...), []byte("function x0e(){let l=a.DEMO_VERSION??"), []byte("function x0e(){let l=MISSING_TARGET??"), 1)
	if applyClaudeUIPatches_2_1_259(broken, "test", "2.1.259", modelconfig.Default()) {
		t.Fatal("patch succeeded without the required logo transformation")
	}
}
