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

func TestClaude284PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.284", claude284SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.284 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.284", claude283SHA); got != nil {
		t.Fatalf("Claude 2.1.284 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.283", claude284SHA); got != nil {
		t.Fatalf("Claude 2.1.284 SHA matched wrong version: %#v", got)
	}
}

func TestClaude284WrongSHAFallsBackToUnpatchedExecutable(t *testing.T) {
	claudePath := t.TempDir() + "/2.1.284"
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

func TestClaude284ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function Kge(e=!1,n=null){` + strings.Repeat(" ", 5000) + `function d2(e,n){}`)
	if !patchModelPickerOptions_2_1_284(data, modelconfig.Default()) {
		t.Fatal("model picker patch reported no changes")
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

func TestClaude284LogoPatchFailsClosedOnOverflow(t *testing.T) {
	data := []byte(`function vQe(){let o=a.DEMO_VERSION??` + strings.Repeat(" ", 1000) + `function _or(o,l,t){}`)
	original := append([]byte(nil), data...)
	if patchLogoDisplayDataFunction_2_1_284(data, strings.Repeat("x", 4000), "2.1.284") {
		t.Fatal("oversized executable replacement unexpectedly succeeded")
	}
	if !bytes.Equal(data, original) {
		t.Fatal("overflowing executable replacement mutated the input")
	}
}

func TestClaude284PatchTargetsMaintenanceBinary(t *testing.T) {
	path := os.Getenv("CLAUDODEX_MAINTENANCE_CLAUDE_REALPATH")
	if path == "" {
		t.Skip("maintenance Claude path is unavailable")
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := sha256Hex(source); got != claude284SHA {
		t.Fatalf("maintenance Claude SHA = %s, want %s", got, claude284SHA)
	}
	if !validateClaude209UIBrandingReplacements(source, claude284UIBrandingReplacements) {
		for _, replacement := range claude284UIBrandingReplacements {
			if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
				t.Logf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
			}
		}
		t.Fatal("Claude 2.1.284 branding prerequisites do not match the maintenance binary")
	}
	records, hashes, ok := claude284EmbeddedBunModuleHashes(source)
	if !ok {
		t.Fatal("Claude 2.1.284 Bun module table is unavailable")
	}
	for _, transformation := range claude284SourceTransformationsForConfig("test", "2.1.284", modelconfig.Default()) {
		candidate := append([]byte(nil), source...)
		if !transformation.apply(candidate) {
			t.Logf("source transformation did not match: %s", transformation.name)
		}
	}
	patched := append([]byte(nil), source...)
	if !applyClaudeUIPatches_2_1_284(patched, "test", "2.1.284", modelconfig.Default()) {
		t.Fatal("complete Claude 2.1.284 patch did not apply")
	}
	for _, transformation := range claude284RemoteControlTransformations() {
		candidate := append([]byte(nil), source...)
		if !transformation.apply(candidate) {
			t.Errorf("remote-control transformation did not match: %s", transformation.name)
		}
	}
	start := bytes.Index(patched, []byte("function CDX284("))
	if start < 0 {
		t.Fatal("patched model picker normalizer is missing")
	}
	end := bytes.Index(patched[start:], []byte("function d2("))
	if end < 0 || strings.Count(string(patched[start:start+end]), `r("`) != 3 {
		t.Fatal("patched model picker does not contain exactly three tiers")
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
	for _, want := range []string{"Claudodex Info", "test using Claude Code v2.1.284", "function xH(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}"} {
		if !bytes.Contains(patched, []byte(want)) {
			t.Fatalf("complete patch missing %q", want)
		}
	}
	for _, forbidden := range []string{`r("fable",`, `r("mythos",`} {
		if bytes.Contains(patched[start:start+end], []byte(forbidden)) {
			t.Fatalf("patched picker retained forbidden fourth-tier marker %q", forbidden)
		}
	}
	broken := bytes.Replace(append([]byte(nil), source...), claude284RequiredLogoAnchor(), []byte("function MISSING_TARGET(){"), 1)
	if applyClaudeUIPatches_2_1_284(broken, "test", "2.1.284", modelconfig.Default()) {
		t.Fatal("patch succeeded without the required logo transformation")
	}
}
