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

func TestClaude294PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.294", claude294SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.294 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.294", claude292SHA); got != nil {
		t.Fatalf("Claude 2.1.294 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.292", claude294SHA); got != nil {
		t.Fatalf("Claude 2.1.294 SHA matched wrong version: %#v", got)
	}
}

func TestClaude294WrongSHAFallsBackToUnpatchedExecutable(t *testing.T) {
	claudePath := t.TempDir() + "/2.1.294"
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

func TestClaude294PatchTargetsMaintenanceBinary(t *testing.T) {
	path := os.Getenv("CLAUDODEX_MAINTENANCE_CLAUDE_REALPATH")
	if path == "" {
		t.Skip("maintenance Claude path is unavailable")
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, transformation := range claude294Transformations("test") {
		t.Run(transformation.name, func(t *testing.T) {
			candidate := append([]byte(nil), source...)
			if !transformation.apply(candidate) {
				t.Fatalf("%s target does not match Claude 2.1.294", transformation.name)
			}
		})
	}
	if got := sha256Hex(source); got != claude294SHA {
		t.Fatalf("maintenance Claude SHA = %s, want %s", got, claude294SHA)
	}
	if !validateClaude209UIBrandingReplacements(source, claude294UIBrandingReplacements) {
		t.Fatal("Claude 2.1.294 branding prerequisites do not match the maintenance binary")
	}
	records, hashes, ok := claude294EmbeddedBunModuleHashes(source)
	if !ok {
		t.Fatal("Claude 2.1.294 Bun module table is unavailable")
	}
	patched := append([]byte(nil), source...)
	if !applyClaudeUIPatches_2_1_294(patched, "test", "2.1.294", modelconfig.Default()) {
		t.Fatal("complete Claude 2.1.294 patch did not apply")
	}
	start := bytes.Index(patched, []byte("function CDX294("))
	if start < 0 {
		t.Fatal("patched picker marker is absent")
	}
	end := bytes.Index(patched[start:], []byte("function GJ("))
	if end < 0 || strings.Count(string(patched[start:start+end]), `r("`) != 3 {
		t.Fatal("patched picker does not contain exactly three tiers")
	}
	for _, forbidden := range []string{`r("fable",`, `r("mythos",`, "Fable 5", "Mythos 5"} {
		if bytes.Contains(patched[start:start+end], []byte(forbidden)) {
			t.Fatalf("patched picker retained forbidden fourth-tier marker %q", forbidden)
		}
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
	broken := bytes.Replace(append([]byte(nil), source...), claude294RequiredLogoAnchor(), []byte("function MISSING_TARGET(){"), 1)
	if applyClaudeUIPatches_2_1_294(broken, "test", "2.1.294", modelconfig.Default()) {
		t.Fatal("patch succeeded without the required logo transformation")
	}
}

func TestClaude294RemoteControlTargetsMaintenanceBinary(t *testing.T) {
	path := os.Getenv("CLAUDODEX_MAINTENANCE_CLAUDE_REALPATH")
	if path == "" {
		t.Skip("maintenance Claude path is unavailable")
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, transformation := range claude294RemoteControlTransformations() {
		t.Run(transformation.name, func(t *testing.T) {
			candidate := append([]byte(nil), source...)
			if !transformation.apply(candidate) {
				t.Fatalf("remote-control %s target does not match Claude 2.1.294", transformation.name)
			}
		})
	}
}
