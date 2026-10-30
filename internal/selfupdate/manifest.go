// Package selfupdate checks signed release metadata and asks before updating.
package selfupdate

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"golang.org/x/mod/semver"
)

const (
	Repository           = "bassner/claudodex"
	Channel              = "stable"
	ManifestName         = "claudodex-update.json"
	SignatureName        = ManifestName + ".sig"
	MaxManifestBytes     = 128 << 10
	MaxReleaseNotesBytes = 64 << 10
)

// Artifact describes a release asset. Source and bottle assets are installed by
// Homebrew, which also checks the hashes recorded in the formula.
type Artifact struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	OS     string `json:"os,omitempty"`
	Arch   string `json:"arch,omitempty"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type Manifest struct {
	Schema       int        `json:"schema"`
	Repository   string     `json:"repository"`
	Channel      string     `json:"channel"`
	KeyID        string     `json:"key_id"`
	Version      string     `json:"version"`
	ReleaseNotes string     `json:"release_notes"`
	Artifacts    []Artifact `json:"artifacts"`
}

var assetName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,199}$`)

var ErrUnsupportedSchema = errors.New("update info needs a newer client; run brew upgrade bassner/tap/claudodex")

// StableVersion accepts complete release versions, with an optional v prefix.
func StableVersion(version string) (string, error) {
	v := "v" + strings.TrimPrefix(version, "v")
	if !semver.IsValid(v) || semver.Canonical(v) != v || semver.Prerelease(v) != "" {
		return "", fmt.Errorf("invalid stable version %q", version)
	}
	return strings.TrimPrefix(v, "v"), nil
}

func (m Manifest) Validate() error {
	if m.Schema < 1 || m.Repository != Repository || m.Channel != Channel || !assetName.MatchString(m.KeyID) {
		return errors.New("unsupported release manifest identity")
	}
	v, err := StableVersion(m.Version)
	if err != nil || v != m.Version {
		return errors.New("invalid manifest version")
	}
	if len(m.ReleaseNotes) > MaxReleaseNotesBytes || len(m.Artifacts) > 32 {
		return errors.New("release metadata exceeds size limits")
	}
	seen := make(map[string]bool)
	for _, a := range m.Artifacts {
		if !assetName.MatchString(a.Name) || seen[a.Name] || a.Size <= 0 || a.Size > 1<<30 {
			return errors.New("invalid or duplicate release artifact")
		}
		seen[a.Name] = true
		if !assetName.MatchString(a.Kind) {
			return errors.New("invalid release artifact kind")
		}
		hash, err := hex.DecodeString(a.SHA256)
		if err != nil || len(hash) != 32 || strings.ToLower(a.SHA256) != a.SHA256 {
			return errors.New("invalid release artifact hash")
		}
		if (a.Kind == "binary" || a.Kind == "bottle") && (a.OS == "" || a.Arch == "") {
			return errors.New("release artifact has no platform")
		}
	}
	if m.Schema > 1 {
		return ErrUnsupportedSchema
	}
	return nil
}

func ParseManifest(data []byte) (Manifest, error) {
	var m Manifest
	if len(data) > MaxManifestBytes {
		return m, errors.New("release manifest too large")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&m); err != nil {
		return m, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return m, errors.New("trailing release manifest data")
	}
	return m, m.Validate()
}

// VerifyManifest authenticates the exact bytes, including the inline notes.
func VerifyManifest(data, signature []byte, keys map[string]ed25519.PublicKey) (Manifest, error) {
	m, parseErr := ParseManifest(data)
	if parseErr != nil && !errors.Is(parseErr, ErrUnsupportedSchema) {
		return m, parseErr
	}
	key := keys[m.KeyID]
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(signature)))
	if err != nil || len(key) != ed25519.PublicKeySize || !ed25519.Verify(key, data, sig) {
		return Manifest{}, errors.New("release manifest signature is invalid")
	}
	return m, parseErr
}

func ReleaseURL(version string) string {
	return "https://github.com/" + Repository + "/releases/tag/v" + version
}
