package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bassner/claudodex/internal/selfupdate"
)

func fixture(t *testing.T) (string, string, string, string) {
	t.Helper()
	dir := t.TempDir()
	source := filepath.Join(dir, sourceName("0.3.40"))
	if err := os.WriteFile(source, []byte("source package"), 0644); err != nil {
		t.Fatal(err)
	}
	bottle := bottleMetadata{}
	bottle.Formula.Name, bottle.Formula.Version = "claudodex", "0.3.40"
	bottle.Bottle.RootURL, bottle.Bottle.Cellar = releaseRoot("0.3.40"), "any_skip_relocation"
	name := "claudodex-0.3.40.arm64_sequoia.bottle.tar.gz"
	local := strings.Replace(name, "claudodex-", "claudodex--", 1)
	content := []byte("Homebrew-built bottle")
	hash := sha256.Sum256(content)
	bottle.Bottle.Tags = map[string]bottleFile{"arm64_sequoia": {Filename: name, LocalFilename: local, SHA256: hex.EncodeToString(hash[:])}}
	if err := os.WriteFile(filepath.Join(dir, local), content, 0644); err != nil {
		t.Fatal(err)
	}
	metadata := filepath.Join(dir, "brew.json")
	writeMetadata(t, metadata, bottle)
	notes := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(notes, []byte("Changes for this release.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return dir, source, metadata, notes
}

func writeMetadata(t *testing.T, path string, bottle bottleMetadata) {
	t.Helper()
	data, err := json.Marshal(map[string]bottleMetadata{"bassner/tap/claudodex": bottle})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestSignedManifestBindsNotesAndVerifiedHomebrewAssets(t *testing.T) {
	dir, source, metadata, notes := fixture(t)
	assets := filepath.Join(dir, "assets")
	if _, err := prepareBottles("0.3.40", metadata, dir, assets); err != nil {
		t.Fatal(err)
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]ed25519.PublicKey{"test-key": public}
	secret := base64.StdEncoding.EncodeToString(private)
	out := filepath.Join(dir, "signed")
	metadata = filepath.Join(assets, "claudodex-bottles.json")
	if err := makeManifest("0.3.40", notes, source, metadata, assets, out, "test-key", secret, keys); err != nil {
		t.Fatal(err)
	}
	// Signing and writing are deterministic, allowing a failed release to resume.
	if err := makeManifest("0.3.40", notes, source, metadata, assets, out, "test-key", secret, keys); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(out, selfupdate.ManifestName))
	sig, _ := os.ReadFile(filepath.Join(out, selfupdate.SignatureName))
	m, err := selfupdate.VerifyManifest(data, sig, keys)
	if err != nil {
		t.Fatal(err)
	}
	if m.ReleaseNotes != "Changes for this release.\n" || len(m.Artifacts) != 2 || m.Artifacts[0].Kind != "source" || m.Artifacts[1].Kind != "bottle" || m.Artifacts[1].Arch != "arm64" {
		t.Fatalf("unexpected signed manifest: %+v", m)
	}
	sourceBytes, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assets, filepath.Base(source)), sourceBytes, 0644); err != nil {
		t.Fatal(err)
	}
	if err := verifyManifestAssets(m, assets); err != nil {
		t.Fatal(err)
	}
	if _, err := selfupdate.VerifyManifest(bytes.Replace(data, []byte("Changes"), []byte("Altered"), 1), sig, keys); err == nil {
		t.Fatal("changed inline notes passed signature verification")
	}
	if err := os.WriteFile(filepath.Join(assets, m.Artifacts[1].Name), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := verifyManifestAssets(m, assets); err == nil {
		t.Fatal("changed release asset passed signed readback verification")
	}
	if err := makeManifest("0.3.40", notes, source, metadata, assets, t.TempDir(), "test-key", secret, keys); err == nil {
		t.Fatal("changed bottle passed checksum verification")
	}
}

func TestSigningKeyMustMatchEmbeddedTrustRoot(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]ed25519.PublicKey{"test": public}
	for _, data := range [][]byte{private, private.Seed()} {
		if _, err := decodeSigningKey(base64.StdEncoding.EncodeToString(data), "test", keys); err != nil {
			t.Fatal(err)
		}
	}
	badPrivate := append([]byte(nil), private...)
	badPrivate[len(badPrivate)-1] ^= 1
	for _, secret := range []string{"", "not base64", base64.StdEncoding.EncodeToString(badPrivate)} {
		if _, err := decodeSigningKey(secret, "test", keys); err == nil {
			t.Fatal("invalid signing key accepted")
		}
	}
	if _, err := decodeSigningKey(base64.StdEncoding.EncodeToString(private), "unknown-key", keys); err == nil {
		t.Fatal("unknown trust root accepted")
	}
	other, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeSigningKey(base64.StdEncoding.EncodeToString(private), "test", map[string]ed25519.PublicKey{"test": other}); err == nil {
		t.Fatal("wrong public key accepted")
	}
}

func TestFormulaUsesPinnedSourceOfflineBuildAndHomebrewMetadata(t *testing.T) {
	_, source, metadata, _ := fixture(t)
	formula, err := makeFormula("0.3.40", source, metadata)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{releaseRoot("0.3.40") + "/" + sourceName("0.3.40"), `version "0.3.40"`, `root_url "` + releaseRoot("0.3.40") + `"`, `cellar: :any_skip_relocation, arm64_sequoia:`, `ENV["GOPROXY"] = "off"`, `ENV["GOTOOLCHAIN"] = "local"`, `ENV["GOFLAGS"] = "-mod=vendor"`, `Go issue 52372`, `"-mod=vendor"`, `main.version=#{version}`, `main.installMethod=homebrew`, `depends_on "go" => :build`} {
		if !strings.Contains(string(formula), expected) {
			t.Errorf("formula does not contain %q", expected)
		}
	}
	for _, forbidden := range []string{"bin.install", "cask", "xattr", "quarantine", `"-trimpath"`, "std_go_args"} {
		if strings.Contains(string(formula), forbidden) {
			t.Errorf("formula contains %q", forbidden)
		}
	}
	if _, err := makeFormula("0.3.41", source, metadata); err == nil {
		t.Fatal("wrong source version accepted")
	}
}

func TestBottlePreparationUsesPublishedFilenameAndRejectsUnsafeMetadata(t *testing.T) {
	dir, _, metadata, _ := fixture(t)
	bottle, err := readBottleMetadata(metadata, "0.3.40")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "public")
	if _, err := prepareBottles("0.3.40", metadata, dir, out); err != nil {
		t.Fatal(err)
	}
	normalized, err := readBottleMetadata(filepath.Join(out, "claudodex-bottles.json"), "0.3.40")
	if err != nil {
		t.Fatal(err)
	}
	file := normalized.Bottle.Tags["arm64_sequoia"]
	if file.LocalFilename != file.Filename {
		t.Fatal("metadata still uses local double-hyphen filename")
	}
	if _, err := os.Stat(filepath.Join(out, file.Filename)); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*bottleMetadata){
		func(b *bottleMetadata) { b.Bottle.RootURL = "https://example.com" },
		func(b *bottleMetadata) { b.Formula.Version = "0.3.41" },
		func(b *bottleMetadata) { b.Bottle.Cellar = `any; puts "bad"` },
		func(b *bottleMetadata) { b.Bottle.Rebuild = 1 },
		func(b *bottleMetadata) {
			e := b.Bottle.Tags["arm64_sequoia"]
			e.LocalFilename = "../escape"
			b.Bottle.Tags["arm64_sequoia"] = e
		},
		func(b *bottleMetadata) {
			e := b.Bottle.Tags["arm64_sequoia"]
			e.SHA256 = strings.Repeat("A", 64)
			b.Bottle.Tags["arm64_sequoia"] = e
		},
	} {
		copyData, _ := json.Marshal(bottle)
		var changed bottleMetadata
		if err := json.Unmarshal(copyData, &changed); err != nil {
			t.Fatal(err)
		}
		mutate(&changed)
		writeMetadata(t, metadata, changed)
		if _, err := readBottleMetadata(metadata, "0.3.40"); err == nil {
			t.Fatal("unsafe bottle metadata accepted")
		}
	}
}

func TestSourceArchiveIsReproducibleAndRejectsUnsafeEntries(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0755); err != nil {
		t.Fatal(err)
	}
	// The fixture tests executable mode retention, independent of the caller's umask.
	if err := os.Chmod(filepath.Join(dir, "main.go"), 0755); err != nil {
		t.Fatal(err)
	}
	first, err := sourceTar(dir, "claudodex-0.3.40")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(dir, "main.go"), time.Now().Add(time.Hour), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	second, err := sourceTar(dir, "claudodex-0.3.40")
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("source archive changed with timestamp: %v", err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(first))
	if err != nil {
		t.Fatal(err)
	}
	tarReader := tar.NewReader(gz)
	header, err := tarReader.Next()
	if err != nil || header.Name != "claudodex-0.3.40/main.go" || header.Mode != 0755 {
		t.Fatalf("bad archive header: %+v %v", header, err)
	}
	for _, name := range []string{"../outside", "/absolute"} {
		var raw bytes.Buffer
		tw := tar.NewWriter(&raw)
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: 1}); err != nil {
			t.Fatal(err)
		}
		_, _ = io.WriteString(tw, "x")
		_ = tw.Close()
		if err := extractSource(raw.Bytes(), t.TempDir()); err == nil {
			t.Fatal("unsafe archive entry accepted")
		}
	}
}

func TestSourceCommandRequiresCleanGitAndArchivesCommittedFiles(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	if err := os.WriteFile("go.mod", []byte("module example.com/packagefixture\n\ngo 1.25.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("main.go", []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "go.mod", "main.go"}, {"-c", "user.name=Release fixture", "-c", "user.email=fixture@example.invalid", "commit", "-qm", "Fixture"}} {
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git: %v %s", err, output)
		}
	}
	out := t.TempDir()
	path, err := prepareSource("0.3.40", out)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("untracked", []byte("private content"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareSource("0.3.40", out); err == nil {
		t.Fatal("dirty checkout was packaged")
	}
}

func TestCLIAndExistingAssetsFailClosed(t *testing.T) {
	for _, args := range [][]string{nil, {"formula", "--version", "0.3"}, {"formula", "--version", "0.3.40-rc1"}, {"unknown", "--version", "0.3.40"}, {"supported-claude", "extra"}} {
		if err := run(args, io.Discard); err == nil {
			t.Fatalf("accepted %#v", args)
		}
	}
	path := filepath.Join(t.TempDir(), "asset")
	if err := writeNew(path, []byte("first"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeNew(path, []byte("second"), 0644); err == nil {
		t.Fatal("replaced existing asset")
	}
	if err := verifyClaude(path, "2.1.314", "darwin", "arm64"); err == nil {
		t.Fatal("unregistered binary accepted")
	}
}
