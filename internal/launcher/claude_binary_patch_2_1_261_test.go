package launcher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude261SHA = "5efecaff231b798be3c66def9be54183623b328b80eaef17f93c43987024e82a"

func TestClaude261PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.261", claude261SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.261 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.261", claude260SHA); got != nil {
		t.Fatalf("Claude 2.1.261 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.260", claude261SHA); got != nil {
		t.Fatalf("Claude 2.1.261 SHA matched wrong version: %#v", got)
	}
}

func TestClaude261WrongSHAFallsBackToUnpatchedExecutable(t *testing.T) {
	claudePath := t.TempDir() + "/2.1.261"
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

func TestClaude261ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function pdn(e=!1){` + strings.Repeat(" ", 5000) + `function rto(e,t){}`)
	if !patchModelPickerOptions_2_1_261(data, modelconfig.Default()) {
		t.Fatal("patchModelPickerOptions_2_1_261 reported no changes")
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
	if !strings.Contains(got, `function vue(e=!1,t=null){return CDXOpts261(e,t)}`) {
		t.Fatal("model picker replacement removed Claude's exported options builder")
	}
	for _, forbidden := range []string{"fable", "Fable", "mythos", "Mythos", "ANTHROPIC_DEFAULT_FABLE_MODEL"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("model picker retained forbidden fourth-tier marker %q", forbidden)
		}
	}
}

func TestClaude261ModelPickerRejectsOverrideAndCustomRows(t *testing.T) {
	data := []byte(claude261ModelOptionsOverrideTarget + claude261ModelSelectionSourceTarget + claude261ModelExtraOptionsTarget + claude261ModelPickerValueTarget)
	if !patchModelPickerExtraOptions_2_1_261(data) {
		t.Fatal("model picker extra-options patch did not apply")
	}
	got := string(data)
	for _, want := range []string{`wt=V(()=>pdn(Zt),[Zt])`, `zt=to===null?zw:cmt(wt,to)??to,`, `Go=wt.slice(0,3)`, `options:Fn.slice(0,3)`} {
		if !strings.Contains(got, want) {
			t.Fatalf("model picker missing %q", want)
		}
	}
	if strings.Contains(got, `selectedValue:`) {
		t.Fatal("model picker must not pass a selected value that the list can re-add as a custom fourth row")
	}
	for _, forbidden := range []string{"Current model", "Base model", "Oe??pdn"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("model picker retained custom-row path %q", forbidden)
		}
	}
}

func TestClaude261ModelPickerResolverReturnsOnlyThreeCanonicalRows(t *testing.T) {
	data := []byte(`function ato(e,t){` + strings.Repeat(" ", 5000) + `function fdn(e){}`)
	if !patchModelPickerResolver_2_1_261(data) {
		t.Fatal("model picker resolver patch did not apply")
	}
	if !strings.Contains(string(data), `function ato(e,t){return CDXOpts261(e,t).slice(0,3)}`) {
		t.Fatal("model picker resolver retained noncanonical option sources")
	}
}

func TestClaude261LogoPatchFailsClosedOnOverflow(t *testing.T) {
	data := []byte(`function PHe(){let l=a.DEMO_VERSION??` + strings.Repeat(" ", 1000) + `function zQt(l,b,x){}`)
	original := append([]byte(nil), data...)
	if patchLogoDisplayDataFunction_2_1_261(data, strings.Repeat("x", 4000), "2.1.261") {
		t.Fatal("oversized executable replacement unexpectedly succeeded")
	}
	if !bytes.Equal(data, original) {
		t.Fatal("overflowing executable replacement mutated the input")
	}
}

func TestClaude261PatchTargetsMaintenanceBinary(t *testing.T) {
	if version := os.Getenv("CLAUDODEX_MAINTENANCE_CLAUDE_VERSION"); version != "" && version != "2.1.261" {
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
	if !validateClaude209UIBrandingReplacements(source, claude261UIBrandingReplacements) {
		for _, replacement := range claude261UIBrandingReplacements {
			if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
				t.Logf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
			}
		}
		t.Fatal("Claude 2.1.261 branding prerequisites do not match the maintenance binary")
	}
	records, hashes, ok := claude260EmbeddedBunModuleHashes(source)
	if !ok {
		t.Fatal("Claude 2.1.261 Bun module table is unavailable")
	}
	patched := append([]byte(nil), source...)
	if !applyClaudeUIPatches_2_1_261(patched, "test", "2.1.261", modelconfig.Default()) {
		for _, transformation := range claude261SourceTransformationsForConfig("test", "2.1.261", modelconfig.Default()) {
			candidate := append([]byte(nil), source...)
			if !transformation.apply(candidate) {
				t.Logf("source transformation did not match: %s", transformation.name)
				if transformation.name == "remote-control" {
					for _, remote := range claude261RemoteControlTransformations() {
						remoteCandidate := append([]byte(nil), source...)
						if !remote.apply(remoteCandidate) {
							t.Logf("remote-control transformation did not match: %s", remote.name)
						}
					}
				}
			}
		}
		t.Fatal("complete Claude 2.1.261 patch did not apply")
	}
	for _, transformation := range claude261Transformations("test") {
		t.Run(transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Errorf("Claude 2.1.261 target %s does not match the maintenance binary", transformation.name)
			}
		})
	}
	for _, transformation := range claude261RemoteControlTransformations() {
		t.Run("remote-control/"+transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Errorf("Claude 2.1.261 remote-control target %s does not match the maintenance binary", transformation.name)
			}
		})
	}
	if tiers := claude261ModelPickerTierCount(patched); tiers != 3 {
		t.Fatalf("patched model picker tier count = %d, want 3", tiers)
	}
	changedModules := 0
	for index, record := range records {
		current := sha256.Sum256(patched[record.contentOffset : record.contentOffset+record.contentLength])
		if current == hashes[index] {
			continue
		}
		changedModules++
		if bytecodeLength := binary.LittleEndian.Uint32(patched[record.bytecodeLength : record.bytecodeLength+4]); bytecodeLength != 0 {
			t.Errorf("changed Bun module %d retained %d bytes of stale bytecode", index, bytecodeLength)
		}
	}
	if changedModules < 2 {
		t.Fatalf("changed Bun module count = %d, want multiple patched modules", changedModules)
	}
	for _, want := range []string{"Claudodex Info", "test using Claude Code v2.1.261", "function RH(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}"} {
		if !bytes.Contains(patched, []byte(want)) {
			t.Fatalf("complete patch missing %q", want)
		}
	}
	start := bytes.Index(patched, []byte("function CDX261("))
	if start < 0 {
		t.Fatal("patched model picker start is missing")
	}
	endRelative := bytes.Index(patched[start:], []byte("function rto("))
	if endRelative < 0 {
		t.Fatal("patched model picker end is missing")
	}
	for _, forbidden := range []string{`r("fable",`, `r("mythos",`, "ANTHROPIC_DEFAULT_FABLE_MODEL"} {
		if bytes.Contains(patched[start:start+endRelative], []byte(forbidden)) {
			t.Fatalf("patched picker retained forbidden fourth-tier marker %q", forbidden)
		}
	}
	broken := bytes.Replace(append([]byte(nil), source...), []byte("function PHe(){let l=a.DEMO_VERSION??"), []byte("function PHe(){let l=MISSING_TARGET??"), 1)
	if applyClaudeUIPatches_2_1_261(broken, "test", "2.1.261", modelconfig.Default()) {
		t.Fatal("patch succeeded without the required logo transformation")
	}
}
