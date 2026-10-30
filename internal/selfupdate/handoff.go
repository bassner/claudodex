package selfupdate

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bassner/claudodex/internal/auth"
)

const handoffEnv = "CLAUDODEX_UPDATE_HANDOFF"

type handoff struct {
	Path    string    `json:"path"`
	SHA256  string    `json:"sha256"`
	Version string    `json:"version"`
	Expires time.Time `json:"expires"`
}

func executableHash(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0022 != 0 {
		return "", errors.New("executable is not a protected regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func handoffDir(home string) (string, error) {
	dir, err := auth.DataDir(home)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "updates", "handoffs"), nil
}

func createHandoff(home, path, version string) (string, func(), error) {
	path, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", nil, err
	}
	hash, err := executableHash(path)
	if err != nil {
		return "", nil, err
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", nil, err
	}
	token := hex.EncodeToString(nonce)
	dir, err := handoffDir(home)
	if err != nil {
		return "", nil, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", nil, err
	}
	file := filepath.Join(dir, token+".json")
	f, err := os.OpenFile(file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.Remove(file) }
	err = json.NewEncoder(f).Encode(handoff{Path: path, SHA256: hash, Version: version, Expires: time.Now().Add(2 * time.Minute)})
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		cleanup()
		return "", nil, err
	}
	return token, cleanup, nil
}

// consumeHandoff always removes the environment marker before hooks or tools can
// inherit it. The record is one-use even if its contents fail validation.
func consumeHandoff(home, currentPath, version string) bool {
	token := os.Getenv(handoffEnv)
	_ = os.Unsetenv(handoffEnv)
	decoded, err := hex.DecodeString(token)
	if err != nil || len(decoded) != 32 || token != strings.ToLower(token) {
		return false
	}
	dir, err := handoffDir(home)
	if err != nil {
		return false
	}
	path := filepath.Join(dir, token+".json")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return false
	}
	data, err := os.ReadFile(path)
	_ = os.Remove(path)
	if err != nil || len(data) > 4096 {
		return false
	}
	var h handoff
	if json.Unmarshal(data, &h) != nil || h.Version != version || time.Now().After(h.Expires) || time.Until(h.Expires) > 2*time.Minute {
		return false
	}
	actual, err := filepath.EvalSymlinks(currentPath)
	if err != nil || actual != h.Path {
		return false
	}
	hash, err := executableHash(actual)
	return err == nil && hash == h.SHA256
}

func handoffEnvironment(env []string, token string) []string {
	out := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if !strings.HasPrefix(entry, handoffEnv+"=") {
			out = append(out, entry)
		}
	}
	return append(out, fmt.Sprintf("%s=%s", handoffEnv, token))
}
