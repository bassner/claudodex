package selfupdate

import (
	"crypto/ed25519"
	"encoding/base64"
)

const ReleaseKeyID = "release-2026-10"

// TrustedKeys returns a copy of the release trust roots. A replacement key must
// be shipped in a trusted package update before the old key is retired.
func TrustedKeys() map[string]ed25519.PublicKey {
	key, err := base64.StdEncoding.DecodeString("UdMAqaOK4+T7WGII6VGgAQi1M5hT4QNiEE/u274wABk=")
	if err != nil {
		panic(err)
	}
	return map[string]ed25519.PublicKey{ReleaseKeyID: key}
}
