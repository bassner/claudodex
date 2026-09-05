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

const claude260SHA = "3c269f66801028823e24a63ced9fdd3988cb86cf85fccd9f03f87e463b9d3e3c"

func TestClaude260PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.260", claude260SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.260 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.260", claude259SHA); got != nil {
		t.Fatalf("Claude 2.1.260 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.259", claude260SHA); got != nil {
		t.Fatalf("Claude 2.1.260 SHA matched wrong version: %#v", got)
	}
}

func TestClaude260WrongSHAFallsBackToUnpatchedExecutable(t *testing.T) {
	claudePath := t.TempDir() + "/2.1.260"
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

func TestClaude260ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function scn(e=!1){` + strings.Repeat(" ", 5000) + `function dQr(e,n){}`)
	if !patchModelPickerOptions_2_1_260(data, modelconfig.Default()) {
		t.Fatal("patchModelPickerOptions_2_1_260 reported no changes")
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
	if !strings.Contains(got, `function Gce(e=!1,n=null){return CDXOpts260(e,n)}`) {
		t.Fatal("model picker replacement removed Claude's exported options builder")
	}
	for _, forbidden := range []string{"fable", "Fable", "mythos", "Mythos", "ANTHROPIC_DEFAULT_FABLE_MODEL"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("model picker retained forbidden fourth-tier marker %q", forbidden)
		}
	}
}

func TestClaude260ModelPickerRejectsOverrideAndCustomRows(t *testing.T) {
	data := []byte(claude260ModelOptionsOverrideTarget + claude260ModelSelectionSourceTarget + claude260ModelExtraOptionsTarget + claude260ModelPickerValueTarget)
	if !patchModelPickerExtraOptions_2_1_260(data) {
		t.Fatal("model picker extra-options patch did not apply")
	}
	got := string(data)
	for _, want := range []string{`At=q(()=>scn(jt),[jt])`, `to=ao===null?Lv:vpt(At,ao)??ao,`, `mn=At.slice(0,3)`, `options:Sr.slice(0,3)`} {
		if !strings.Contains(got, want) {
			t.Fatalf("model picker missing %q", want)
		}
	}
	if strings.Contains(got, `selectedValue:`) {
		t.Fatal("model picker must not pass a selected value that the list can re-add as a custom fourth row")
	}
	for _, forbidden := range []string{"Current model", "Base model", "Oe??scn"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("model picker retained custom-row path %q", forbidden)
		}
	}
}

func TestClaude260ModelPickerResolverReturnsOnlyThreeCanonicalRows(t *testing.T) {
	data := []byte(`function gQr(e,n){` + strings.Repeat(" ", 5000) + `function icn(e){}`)
	if !patchModelPickerResolver_2_1_260(data) {
		t.Fatal("model picker resolver patch did not apply")
	}
	if !strings.Contains(string(data), `function gQr(e,n){return CDXOpts260(e,n).slice(0,3)}`) {
		t.Fatal("model picker resolver retained noncanonical option sources")
	}
}

func TestClaude260LogoPatchFailsClosedOnOverflow(t *testing.T) {
	data := []byte(`function Lxe(){let l=a.DEMO_VERSION??` + strings.Repeat(" ", 1000) + `function LYt(l,b,x){}`)
	original := append([]byte(nil), data...)
	if patchLogoDisplayDataFunction_2_1_260(data, strings.Repeat("x", 4000), "2.1.260") {
		t.Fatal("oversized executable replacement unexpectedly succeeded")
	}
	if !bytes.Equal(data, original) {
		t.Fatal("overflowing executable replacement mutated the input")
	}
}

func TestClaude260DisablesStaleEmbeddedPatchedModuleBytecode(t *testing.T) {
	if version := os.Getenv("CLAUDODEX_MAINTENANCE_CLAUDE_VERSION"); version != "" && version != "2.1.260" {
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
	record, ok := claude260EmbeddedPatchedModuleRecord(data)
	if !ok {
		t.Fatal("maintenance binary patched module record is missing")
	}
	before := binary.LittleEndian.Uint32(data[record.bytecodeLength : record.bytecodeLength+4])
	if before == 0 {
		t.Fatal("maintenance binary patched module bytecode is already disabled")
	}
	if !patchModelPickerOptions_2_1_260(data, modelconfig.Default()) {
		t.Fatal("model picker source prerequisite did not apply")
	}
	if !disableClaude260EmbeddedPatchedModuleBytecode(data) {
		t.Fatal("embedded patched module bytecode was not disabled")
	}
	after := binary.LittleEndian.Uint32(data[record.bytecodeLength : record.bytecodeLength+4])
	if after != 0 {
		t.Fatalf("patched module bytecode length = %d, want zero", after)
	}
}

func TestClaude260PatchTargetsMaintenanceBinary(t *testing.T) {
	if version := os.Getenv("CLAUDODEX_MAINTENANCE_CLAUDE_VERSION"); version != "" && version != "2.1.260" {
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
	if !validateClaude209UIBrandingReplacements(source, claude260UIBrandingReplacements) {
		for _, replacement := range claude260UIBrandingReplacements {
			if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
				t.Logf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
			}
		}
		t.Fatal("Claude 2.1.260 branding prerequisites do not match the maintenance binary")
	}
	records, hashes, ok := claude260EmbeddedBunModuleHashes(source)
	if !ok {
		t.Fatal("Claude 2.1.260 Bun module table is unavailable")
	}
	patched := append([]byte(nil), source...)
	if !applyClaudeUIPatches_2_1_260(patched, "test", "2.1.260", modelconfig.Default()) {
		for _, transformation := range claude260SourceTransformationsForConfig("test", "2.1.260", modelconfig.Default()) {
			candidate := append([]byte(nil), source...)
			if !transformation.apply(candidate) {
				t.Logf("source transformation did not match: %s", transformation.name)
				if transformation.name == "remote-control" {
					for _, remote := range claude260RemoteControlTransformations() {
						remoteCandidate := append([]byte(nil), source...)
						if !remote.apply(remoteCandidate) {
							t.Logf("remote-control transformation did not match: %s", remote.name)
						}
					}
				}
			}
		}
		t.Fatal("complete Claude 2.1.260 patch did not apply")
	}
	for _, transformation := range claude260Transformations("test") {
		t.Run(transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Errorf("Claude 2.1.260 target %s does not match the maintenance binary", transformation.name)
			}
		})
	}
	for _, transformation := range claude260RemoteControlTransformations() {
		t.Run("remote-control/"+transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Errorf("Claude 2.1.260 remote-control target %s does not match the maintenance binary", transformation.name)
			}
		})
	}
	sequential := append([]byte(nil), source...)
	for _, transformation := range claude260Transformations("test") {
		if !transformation.apply(sequential) {
			t.Fatalf("Claude 2.1.260 sequential target %s failed after prior transformations", transformation.name)
		}
	}
	if tiers := claude260ModelPickerTierCount(patched); tiers != 3 {
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
	for _, want := range []string{"Claudodex Info", "test using Claude Code v2.1.260", "function TH(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}"} {
		if !bytes.Contains(patched, []byte(want)) {
			t.Fatalf("complete patch missing %q", want)
		}
	}
	start := bytes.Index(patched, []byte("function CDX260("))
	end := bytes.Index(patched[start:], []byte("function dQr("))
	if start < 0 || end < 0 {
		t.Fatal("patched model picker bounds are missing")
	}
	for _, forbidden := range []string{`r("fable",`, `r("mythos",`, "ANTHROPIC_DEFAULT_FABLE_MODEL"} {
		if bytes.Contains(patched[start:start+end], []byte(forbidden)) {
			t.Fatalf("patched picker retained forbidden fourth-tier marker %q", forbidden)
		}
	}
	broken := bytes.Replace(append([]byte(nil), source...), []byte("function Lxe(){let l=a.DEMO_VERSION??"), []byte("function Lxe(){let l=MISSING_TARGET??"), 1)
	if applyClaudeUIPatches_2_1_260(broken, "test", "2.1.260", modelconfig.Default()) {
		t.Fatal("patch succeeded without the required logo transformation")
	}
}
