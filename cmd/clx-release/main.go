// clx-release prepares the source package, Homebrew formula and signed metadata.
// It never publishes assets or reads a signing key from a file.
package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/bassner/claudodex/internal/launcher"
	"github.com/bassner/claudodex/internal/selfupdate"
)

const signingKeyEnv = "CLAUDODEX_RELEASE_SIGNING_KEY"

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "clx-release:", err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	if len(args) == 0 {
		return errors.New("use source, formula, bottles, manifest, verify-manifest, verify-claude, or supported-claude")
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	version := flags.String("version", "", "stable release version")
	source := flags.String("source", "", "uploaded vendored source archive")
	bottleJSON := flags.String("bottle-json", "", "Homebrew bottle JSON")
	assetDir := flags.String("asset-dir", "", "artifact directory (verify-manifest also checks these files)")
	out := flags.String("out", ".", "output directory or formula file")
	notes := flags.String("notes", "", "plain-text release notes file")
	keyID := flags.String("key-id", selfupdate.ReleaseKeyID, "embedded release signing key ID")
	file := flags.String("file", "", "local file to verify")
	goos := flags.String("os", "darwin", "registered Claude operating system")
	arch := flags.String("arch", "arm64", "registered Claude architecture")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if args[0] == "supported-claude" {
		return json.NewEncoder(output).Encode(launcher.SupportedClaudeBuilds())
	}
	if args[0] == "verify-manifest" {
		data, err := os.ReadFile(filepath.Join(*out, selfupdate.ManifestName))
		if err != nil {
			return err
		}
		sig, err := os.ReadFile(filepath.Join(*out, selfupdate.SignatureName))
		if err != nil {
			return err
		}
		manifest, err := selfupdate.VerifyManifest(data, sig, selfupdate.TrustedKeys())
		if err != nil {
			return err
		}
		if *version != "" {
			v, err := selfupdate.StableVersion(*version)
			if err != nil || manifest.Version != v {
				return errors.New("signed manifest has an unexpected release version")
			}
		}
		if *assetDir != "" {
			return verifyManifestAssets(manifest, *assetDir)
		}
		return nil
	}
	v, err := selfupdate.StableVersion(*version)
	if err != nil {
		return err
	}
	switch args[0] {
	case "source":
		name, err := prepareSource(v, *out)
		if err == nil {
			fmt.Fprintln(output, name)
		}
		return err
	case "verify-claude":
		return verifyClaude(*file, v, *goos, *arch)
	case "bottles":
		_, err := prepareBottles(v, *bottleJSON, *assetDir, *out)
		return err
	case "formula":
		data, err := makeFormula(v, *source, *bottleJSON)
		if err != nil {
			return err
		}
		return writeNew(*out, data, 0644)
	case "manifest":
		return makeManifest(v, *notes, *source, *bottleJSON, *assetDir, *out, *keyID, os.Getenv(signingKeyEnv), selfupdate.TrustedKeys())
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func verifyManifestAssets(manifest selfupdate.Manifest, dir string) error {
	for _, expected := range manifest.Artifacts {
		actual, err := fileArtifact(filepath.Join(dir, expected.Name), expected.Name, expected.Kind, expected.OS, expected.Arch)
		if err != nil {
			return err
		}
		if actual.SHA256 != expected.SHA256 || actual.Size != expected.Size {
			return fmt.Errorf("release asset does not match the signed manifest: %s", expected.Name)
		}
	}
	return nil
}

func sourceName(version string) string { return "claudodex-" + version + "-source.tar.gz" }
func releaseRoot(version string) string {
	return "https://github.com/" + selfupdate.Repository + "/releases/download/v" + version
}

func fileArtifact(path, name, kind, goos, arch string) (selfupdate.Artifact, error) {
	a := selfupdate.Artifact{Name: name, Kind: kind, OS: goos, Arch: arch}
	entry, err := os.Lstat(path)
	if err != nil {
		return a, err
	}
	if !entry.Mode().IsRegular() {
		return a, fmt.Errorf("artifact is not a regular file: %s", name)
	}
	f, err := os.Open(path)
	if err != nil {
		return a, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return a, fmt.Errorf("artifact is not a regular file: %s", name)
	}
	h := sha256.New()
	a.Size, err = io.Copy(h, io.LimitReader(f, (1<<30)+1))
	if err != nil {
		return a, err
	}
	a.SHA256 = hex.EncodeToString(h.Sum(nil))
	check := selfupdate.Manifest{Schema: 1, Repository: selfupdate.Repository, Channel: selfupdate.Channel, KeyID: selfupdate.ReleaseKeyID, Version: "0.0.1", Artifacts: []selfupdate.Artifact{a}}
	return a, check.Validate()
}

func sourceArtifact(version, path string) (selfupdate.Artifact, error) {
	if filepath.Base(path) != sourceName(version) {
		return selfupdate.Artifact{}, errors.New("source archive name does not match release version")
	}
	return fileArtifact(path, sourceName(version), "source", "", "")
}

func makeManifest(version, notesPath, source, bottleJSON, assetDir, out, keyID, secret string, keys map[string]ed25519.PublicKey) error {
	key, err := decodeSigningKey(secret, keyID, keys)
	if err != nil {
		return err
	}
	notes, err := os.ReadFile(notesPath)
	if err != nil {
		return err
	}
	if len(notes) > selfupdate.MaxReleaseNotesBytes {
		return errors.New("release notes are too large")
	}
	artifact, err := sourceArtifact(version, source)
	if err != nil {
		return err
	}
	bottle, err := readBottleMetadata(bottleJSON, version)
	if err != nil {
		return err
	}
	artifacts := []selfupdate.Artifact{artifact}
	for _, tag := range sortedTags(bottle) {
		entry := bottle.Bottle.Tags[tag]
		a, err := fileArtifact(filepath.Join(assetDir, entry.Filename), entry.Filename, "bottle", "darwin", "arm64")
		if err != nil {
			return err
		}
		if a.SHA256 != entry.SHA256 {
			return errors.New("bottle checksum does not match Homebrew metadata")
		}
		artifacts = append(artifacts, a)
	}
	m := selfupdate.Manifest{Schema: 1, Repository: selfupdate.Repository, Channel: selfupdate.Channel, KeyID: keyID, Version: version, ReleaseNotes: string(notes), Artifacts: artifacts}
	if err := m.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if len(data) > selfupdate.MaxManifestBytes {
		return errors.New("release manifest is too large")
	}
	sig := []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(key, data)) + "\n")
	if _, err := selfupdate.VerifyManifest(data, sig, keys); err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	if err := writeNew(filepath.Join(out, selfupdate.ManifestName), data, 0644); err != nil {
		return err
	}
	return writeNew(filepath.Join(out, selfupdate.SignatureName), sig, 0644)
}

func decodeSigningKey(secret, keyID string, keys map[string]ed25519.PublicKey) (ed25519.PrivateKey, error) {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(secret))
	if err != nil {
		return nil, errors.New("release signing environment variable is not valid base64")
	}
	var key ed25519.PrivateKey
	switch len(decoded) {
	case ed25519.SeedSize:
		key = ed25519.NewKeyFromSeed(decoded)
	case ed25519.PrivateKeySize:
		key = ed25519.NewKeyFromSeed(decoded[:ed25519.SeedSize])
		if !key.Equal(ed25519.PrivateKey(decoded)) {
			return nil, errors.New("release signing key is inconsistent")
		}
	default:
		return nil, errors.New("release signing key must be a base64 Ed25519 seed or private key")
	}
	if !key.Public().(ed25519.PublicKey).Equal(keys[keyID]) {
		return nil, errors.New("release signing key does not match the embedded trust root")
	}
	return key, nil
}

type bottleFile struct {
	Filename      string `json:"filename"`
	LocalFilename string `json:"local_filename"`
	SHA256        string `json:"sha256"`
}
type bottleMetadata struct {
	Formula struct {
		Name    string `json:"name"`
		Version string `json:"pkg_version"`
	} `json:"formula"`
	Bottle struct {
		RootURL string                `json:"root_url"`
		Cellar  string                `json:"cellar"`
		Rebuild int                   `json:"rebuild"`
		Tags    map[string]bottleFile `json:"tags"`
	} `json:"bottle"`
}

var bottleTag = regexp.MustCompile(`^arm64_[a-z][a-z0-9_]{0,39}$`)

func readBottleMetadata(path, version string) (bottleMetadata, error) {
	var entries map[string]bottleMetadata
	var result bottleMetadata
	data, err := os.ReadFile(path)
	if err != nil {
		return result, err
	}
	if len(data) > 4<<20 {
		return result, errors.New("bottle metadata is too large")
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return result, err
	}
	if len(entries) != 1 {
		return result, errors.New("expected exactly one Homebrew formula")
	}
	result, ok := entries["bassner/tap/claudodex"]
	if !ok || result.Formula.Name != "claudodex" || result.Formula.Version != version || result.Bottle.RootURL != releaseRoot(version) || result.Bottle.Rebuild != 0 {
		return result, errors.New("bottle metadata has an unexpected formula, version, root URL, or rebuild")
	}
	if result.Bottle.Cellar != "any" && result.Bottle.Cellar != "any_skip_relocation" && result.Bottle.Cellar != "/opt/homebrew/Cellar" {
		return result, errors.New("unsupported Homebrew bottle cellar")
	}
	if len(result.Bottle.Tags) == 0 || len(result.Bottle.Tags) > 8 {
		return result, errors.New("invalid Homebrew bottle count")
	}
	for tag, file := range result.Bottle.Tags {
		expected := "claudodex-" + version + "." + tag + ".bottle.tar.gz"
		local := "claudodex--" + version + "." + tag + ".bottle.tar.gz"
		if !bottleTag.MatchString(tag) || file.Filename != expected || (file.LocalFilename != local && file.LocalFilename != expected) {
			return result, errors.New("unexpected Homebrew bottle filename or platform")
		}
		hash, err := hex.DecodeString(file.SHA256)
		if err != nil || len(hash) != 32 || strings.ToLower(file.SHA256) != file.SHA256 {
			return result, errors.New("invalid Homebrew bottle checksum")
		}
	}
	return result, nil
}

func sortedTags(bottle bottleMetadata) []string {
	tags := make([]string, 0, len(bottle.Bottle.Tags))
	for tag := range bottle.Bottle.Tags {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

// prepareBottles uses Homebrew's published filename, which can differ from the
// local filename. Only the small public formula/bottle fields are retained.
func prepareBottles(version, metadata, assetDir, out string) (bottleMetadata, error) {
	bottle, err := readBottleMetadata(metadata, version)
	if err != nil {
		return bottle, err
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return bottle, err
	}
	for _, tag := range sortedTags(bottle) {
		entry := bottle.Bottle.Tags[tag]
		path := filepath.Join(assetDir, entry.LocalFilename)
		a, err := fileArtifact(path, entry.Filename, "bottle", "darwin", "arm64")
		if err != nil {
			return bottle, err
		}
		if a.SHA256 != entry.SHA256 {
			return bottle, errors.New("bottle checksum does not match Homebrew metadata")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return bottle, err
		}
		if err := writeNew(filepath.Join(out, entry.Filename), data, 0644); err != nil {
			return bottle, err
		}
		entry.LocalFilename = entry.Filename
		bottle.Bottle.Tags[tag] = entry
	}
	data, err := json.MarshalIndent(map[string]bottleMetadata{"bassner/tap/claudodex": bottle}, "", "  ")
	if err != nil {
		return bottle, err
	}
	return bottle, writeNew(filepath.Join(out, "claudodex-bottles.json"), append(data, '\n'), 0644)
}

func makeFormula(version, source, metadata string) ([]byte, error) {
	a, err := sourceArtifact(version, source)
	if err != nil {
		return nil, err
	}
	var block strings.Builder
	if metadata != "" {
		bottle, err := readBottleMetadata(metadata, version)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&block, "\n  bottle do\n    root_url %q\n", releaseRoot(version))
		cellar := ":" + bottle.Bottle.Cellar
		if strings.HasPrefix(bottle.Bottle.Cellar, "/") {
			cellar = fmt.Sprintf("%q", bottle.Bottle.Cellar)
		}
		for _, tag := range sortedTags(bottle) {
			fmt.Fprintf(&block, "    sha256 cellar: %s, %s: %q\n", cellar, tag, bottle.Bottle.Tags[tag].SHA256)
		}
		block.WriteString("  end\n")
	}
	return []byte(fmt.Sprintf(`class Claudodex < Formula
  desc "Claude Code routed through your OpenAI Codex subscription"
  homepage "https://github.com/bassner/claudodex"
  url %q
  version %q
  sha256 %q
  license "MIT"
%s
  depends_on "go" => :build
  depends_on :macos
  depends_on arch: :arm64

  def install
    ENV["GOPROXY"] = "off"
    ENV["GOTOOLCHAIN"] = "local"
    ENV["GOWORK"] = "off"
    ENV["GOFLAGS"] = "-mod=vendor"
    # Go issue 52372: -trimpath removes the recorded -ldflags used to verify
    # version/install method before an update handoff. Keep those settings.
    system "go", "build", "-mod=vendor",
           "-ldflags", "-s -w -X main.version=#{version} -X main.installMethod=homebrew",
           "-o", bin/"claudodex", "./cmd/claudodex"
  end

  test do
    ENV["CLAUDODEX_NO_AUTO_UPDATE"] = "1"
    assert_equal "claudodex #{version}",
                 shell_output("#{bin}/claudodex clx:version").strip
  end
end
`, releaseRoot(version)+"/"+a.Name, version, a.SHA256, block.String())), nil
}

func verifyClaude(path, version, goos, arch string) error {
	artifact, err := fileArtifact(path, "claude", "binary", goos, arch)
	if err != nil {
		return err
	}
	for _, build := range launcher.SupportedClaudeBuilds() {
		if build.Version == version && build.GOOS == goos && build.GOARCH == arch && build.SHA256 == artifact.SHA256 {
			return nil
		}
	}
	return errors.New("Claude binary does not match the exact registered version, platform, and SHA256")
}

// Existing identical files make interrupted local preparation safe to resume.
// Different bytes are never replaced, including a previously signed manifest.
func writeNew(path string, data []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if errors.Is(err, os.ErrExist) {
		existing, readErr := os.ReadFile(path)
		if readErr == nil && string(existing) == string(data) {
			return nil
		}
		return fmt.Errorf("refusing to replace existing file %s", filepath.Base(path))
	}
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(path)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return err
	}
	return nil
}
