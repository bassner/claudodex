package selfupdate

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

const fixtureKeyID = "fixture-release-key"

func newFixtureKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return public, private
}

func fixtureManifest() Manifest {
	return Manifest{
		Schema:       1,
		Repository:   Repository,
		Channel:      Channel,
		KeyID:        fixtureKeyID,
		Version:      "1.2.3",
		ReleaseNotes: "Signed fixture notes.",
		Artifacts: []Artifact{{
			Name:   "claudodex_1.2.3_source.tar.gz",
			Kind:   "source",
			Size:   128,
			SHA256: strings.Repeat("a", 64),
		}},
	}
}

func signFixture(t *testing.T, manifest Manifest, private ed25519.PrivateKey) ([]byte, []byte) {
	t.Helper()
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	signature := ed25519.Sign(private, data)
	return data, []byte(base64.StdEncoding.EncodeToString(signature))
}

func TestVerifyManifestAuthenticSignedFixture(t *testing.T) {
	public, private := newFixtureKey(t)
	data, signature := signFixture(t, fixtureManifest(), private)

	got, err := VerifyManifest(data, signature, map[string]ed25519.PublicKey{fixtureKeyID: public})
	if err != nil {
		t.Fatalf("VerifyManifest() error = %v", err)
	}
	if got.Version != "1.2.3" || got.ReleaseNotes != "Signed fixture notes." || got.KeyID != fixtureKeyID {
		t.Fatalf("VerifyManifest() = %#v", got)
	}
}

func TestVerifyManifestRejectsInvalidFixtures(t *testing.T) {
	public, private := newFixtureKey(t)
	keys := map[string]ed25519.PublicKey{fixtureKeyID: public}
	validData, validSignature := signFixture(t, fixtureManifest(), private)

	tests := []struct {
		name string
		data []byte
		sig  []byte
	}{
		{name: "changed release notes", data: changedNotes(t, validData), sig: validSignature},
		{name: "bad signature", data: validData, sig: badFixtureSignature(t, validSignature)},
	}
	for _, tt := range []struct {
		name   string
		mutate func(*Manifest)
	}{
		{name: "wrong repository", mutate: func(m *Manifest) { m.Repository = "someone/else" }},
		{name: "wrong channel", mutate: func(m *Manifest) { m.Channel = "preview" }},
		{name: "unknown key id", mutate: func(m *Manifest) { m.KeyID = "untrusted-fixture-key" }},
		{name: "invalid version", mutate: func(m *Manifest) { m.Version = "v1.2.3" }},
		{name: "duplicate assets", mutate: func(m *Manifest) { m.Artifacts = append(m.Artifacts, m.Artifacts[0]) }},
		{name: "invalid artifact hash", mutate: func(m *Manifest) { m.Artifacts[0].SHA256 = "not-a-sha256" }},
	} {
		manifest := fixtureManifest()
		tt.mutate(&manifest)
		data, signature := signFixture(t, manifest, private)
		tests = append(tests, struct {
			name string
			data []byte
			sig  []byte
		}{name: tt.name, data: data, sig: signature})
	}
	tests = append(tests,
		struct {
			name string
			data []byte
			sig  []byte
		}{name: "oversized manifest", data: make([]byte, MaxManifestBytes+1), sig: validSignature},
		struct {
			name string
			data []byte
			sig  []byte
		}{name: "trailing JSON", data: append(append([]byte(nil), validData...), []byte(` {}`)...), sig: validSignature},
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := VerifyManifest(tt.data, tt.sig, keys); err == nil {
				t.Fatal("VerifyManifest() accepted invalid manifest data")
			}
		})
	}
}

func TestVerifyManifestAcceptsRotationKeyMap(t *testing.T) {
	oldPublic, oldPrivate := newFixtureKey(t)
	newPublic, newPrivate := newFixtureKey(t)
	keys := map[string]ed25519.PublicKey{
		"release-old": oldPublic,
		"release-new": newPublic,
	}

	for _, fixture := range []struct {
		keyID   string
		private ed25519.PrivateKey
	}{
		{keyID: "release-old", private: oldPrivate},
		{keyID: "release-new", private: newPrivate},
	} {
		t.Run(fixture.keyID, func(t *testing.T) {
			manifest := fixtureManifest()
			manifest.KeyID = fixture.keyID
			data, signature := signFixture(t, manifest, fixture.private)
			if _, err := VerifyManifest(data, signature, keys); err != nil {
				t.Fatalf("VerifyManifest() rejected %s during rotation: %v", fixture.keyID, err)
			}
		})
	}
}

func changedNotes(t *testing.T, original []byte) []byte {
	t.Helper()
	var manifest Manifest
	if err := json.Unmarshal(original, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.ReleaseNotes += " changed"
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSignedManifestCanAddFieldsAndArtifactKinds(t *testing.T) {
	pub, key := newFixtureKey(t)
	m := fixtureManifest()
	m.Artifacts[0].Kind = "future-installer"
	data, _ := json.Marshal(m)
	var extended map[string]any
	json.Unmarshal(data, &extended)
	extended["future_field"] = map[string]any{"ignored": "still signed"}
	data, _ = json.Marshal(extended)
	sig := []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(key, data)))
	if _, err := VerifyManifest(data, sig, map[string]ed25519.PublicKey{fixtureKeyID: pub}); err != nil {
		t.Fatalf("additive metadata broke client: %v", err)
	}
}

func TestNewSchemaNoticeRequiresAuthenticSignature(t *testing.T) {
	pub, key := newFixtureKey(t)
	m := fixtureManifest()
	m.Schema = 2
	data, sig := signFixture(t, m, key)
	keys := map[string]ed25519.PublicKey{fixtureKeyID: pub}
	if _, err := VerifyManifest(data, sig, keys); !errors.Is(err, ErrUnsupportedSchema) {
		t.Fatalf("missing upgrade notice: %v", err)
	}
	if _, err := VerifyManifest(data, badFixtureSignature(t, sig), keys); err == nil || errors.Is(err, ErrUnsupportedSchema) {
		t.Fatalf("unauthenticated upgrade notice: %v", err)
	}
}

func badFixtureSignature(t *testing.T, signature []byte) []byte {
	t.Helper()
	decoded, err := base64.StdEncoding.DecodeString(string(signature))
	if err != nil {
		t.Fatal(err)
	}
	decoded[0] ^= 1
	return []byte(base64.StdEncoding.EncodeToString(decoded))
}
