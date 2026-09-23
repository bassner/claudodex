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

func TestClaude281PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.281", claude281SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.281 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.281", claude278SHA); got != nil {
		t.Fatalf("Claude 2.1.281 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.278", claude281SHA); got != nil {
		t.Fatalf("Claude 2.1.281 SHA matched wrong version: %#v", got)
	}
}

func TestClaude281WrongSHAFallsBackToUnpatchedExecutable(t *testing.T) {
	claudePath := t.TempDir() + "/2.1.281"
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

func TestClaude281ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function RW(e=!1){` + strings.Repeat(" ", 5000) + `function or(e){}`)
	if !patchModelPickerOptions_2_1_281(data, modelconfig.Default()) {
		t.Fatal("patchModelPickerOptions_2_1_281 reported no changes")
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
	for _, forbidden := range []string{"fable", "Fable", "mythos", "Mythos", "ANTHROPIC_DEFAULT_FABLE_MODEL"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("model picker retained forbidden fourth-tier marker %q", forbidden)
		}
	}
}

func TestClaude281ModelPickerEntryPointsReturnOnlyThreeCanonicalRows(t *testing.T) {
	tests := []struct {
		name  string
		input string
		patch func([]byte) bool
		want  string
	}{
		{"resolver", `function DW(e,n){` + strings.Repeat(" ", 5000) + `function egr(e){}`, patchModelPickerResolver_2_1_281, `function DW(e,n){return CDXOpts281(e).slice(0,3)}`},
		{"list", `function P6(e=!1){` + strings.Repeat(" ", 5000) + `var TW=`, patchModelListOptions_2_1_281, `function P6(e=!1){return CDXOpts281(e).slice(0,3)}`},
		{"extra", `function Que(e=!1,n=null){` + strings.Repeat(" ", 5000) + `function xW(e,n){}`, patchModelPickerExtraOptions_2_1_281, `function Que(e=!1,n=null){return CDXOpts281(e).slice(0,3)}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := []byte(test.input)
			if !test.patch(data) || !strings.Contains(string(data), test.want) {
				t.Fatalf("model picker entry point missing %q", test.want)
			}
			for _, forbidden := range []string{"Current model", "Base model", "fable", "Fable", "mythos", "Mythos"} {
				if strings.Contains(string(data), forbidden) {
					t.Fatalf("model picker retained custom-row path %q", forbidden)
				}
			}
		})
	}
}

func TestClaude281LogoPatchFailsClosedOnOverflow(t *testing.T) {
	data := []byte(`function f8e(){let o=a.DEMO_VERSION??` + strings.Repeat(" ", 1000) + `function k9n(o,l,t){}`)
	original := append([]byte(nil), data...)
	if patchLogoDisplayDataFunction_2_1_281(data, strings.Repeat("x", 4000), "2.1.281") {
		t.Fatal("oversized executable replacement unexpectedly succeeded")
	}
	if !bytes.Equal(data, original) {
		t.Fatal("overflowing executable replacement mutated the input")
	}
}

func TestClaude281PatchTargetsMaintenanceBinary(t *testing.T) {
	if version := os.Getenv("CLAUDODEX_MAINTENANCE_CLAUDE_VERSION"); version != "" && version != "2.1.281" {
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
	if got := sha256Hex(source); got != claude281SHA {
		t.Fatalf("maintenance Claude SHA = %s, want %s", got, claude281SHA)
	}
	if !validateClaude209UIBrandingReplacements(source, claude281UIBrandingReplacements) {
		for _, replacement := range claude281UIBrandingReplacements {
			if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
				t.Logf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
			}
		}
		t.Fatal("Claude 2.1.281 branding prerequisites do not match the maintenance binary")
	}
	records, hashes, ok := claude281EmbeddedBunModuleHashes(source)
	if !ok {
		t.Fatal("Claude 2.1.281 Bun module table is unavailable")
	}
	patched := append([]byte(nil), source...)
	if !applyClaudeUIPatches_2_1_281(patched, "test", "2.1.281", modelconfig.Default()) {
		for _, transformation := range claude281SourceTransformationsForConfig("test", "2.1.281", modelconfig.Default()) {
			candidate := append([]byte(nil), source...)
			if !transformation.apply(candidate) {
				t.Logf("source transformation did not match: %s", transformation.name)
				if transformation.name == "remote-control" {
					for _, remote := range claude281RemoteControlTransformations() {
						remoteCandidate := append([]byte(nil), source...)
						if !remote.apply(remoteCandidate) {
							t.Logf("remote-control transformation did not match: %s", remote.name)
						}
					}
				}
			}
		}
		t.Fatal("complete Claude 2.1.281 patch did not apply")
	}
	for _, transformation := range claude281Transformations("test") {
		t.Run(transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Errorf("Claude 2.1.281 target %s does not match the maintenance binary", transformation.name)
			}
		})
	}
	for _, transformation := range claude281RemoteControlTransformations() {
		t.Run("remote-control/"+transformation.name, func(t *testing.T) {
			data := append([]byte(nil), source...)
			if !transformation.apply(data) {
				t.Errorf("Claude 2.1.281 remote-control target %s does not match the maintenance binary", transformation.name)
			}
		})
	}
	if tiers := claude281ModelPickerTierCount(patched); tiers != 3 {
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
	for _, want := range []string{"Claudodex Info", "test using Claude Code v2.1.281", "function sN(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}"} {
		if !bytes.Contains(patched, []byte(want)) {
			t.Fatalf("complete patch missing %q", want)
		}
	}
	start := bytes.Index(patched, []byte("function CDX281("))
	endRelative := bytes.Index(patched[start:], []byte("function or("))
	for _, forbidden := range []string{`r("fable",`, `r("mythos",`, "ANTHROPIC_DEFAULT_FABLE_MODEL"} {
		if bytes.Contains(patched[start:start+endRelative], []byte(forbidden)) {
			t.Fatalf("patched picker retained forbidden fourth-tier marker %q", forbidden)
		}
	}
	broken := bytes.Replace(append([]byte(nil), source...), claude281RequiredLogoAnchor(), []byte("function MISSING_TARGET(){"), 1)
	if applyClaudeUIPatches_2_1_281(broken, "test", "2.1.281", modelconfig.Default()) {
		t.Fatal("patch succeeded without the required logo transformation")
	}
}
