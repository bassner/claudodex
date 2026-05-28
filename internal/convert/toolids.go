package convert

import (
	"crypto/sha256"
	"encoding/hex"
)

func ClampCallID(id string) string {
	if len(id) <= 64 {
		return id
	}
	sum := sha256.Sum256([]byte(id))
	return id[:47] + "-" + hex.EncodeToString(sum[:])[:16]
}
