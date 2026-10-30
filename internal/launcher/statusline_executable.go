package launcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bassner/claudodex/internal/auth"
	"github.com/bassner/claudodex/internal/filelock"
)

// PinStatusLineExecutable keeps a verified copy of the actual current helper.
// A running session can still call it after Homebrew removes the old keg.
// Helpers are never deleted automatically.
func PinStatusLineExecutable(home, sourcePath string) (string, error) {
	sourcePath, err := filepath.EvalSymlinks(sourcePath)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(sourcePath)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("statusline helper is not a regular executable")
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}
	digest := sha256Hex(data)
	dataDir, err := auth.DataDir(home)
	if err != nil {
		return "", err
	}
	root := filepath.Join(dataDir, "statusline-helpers")
	if err := ensurePrivateDirectory(root); err != nil {
		return "", err
	}
	unlock, err := filelock.Acquire(context.Background(), filepath.Join(root, ".lock"))
	if err != nil {
		return "", err
	}
	defer unlock()
	dir := filepath.Join(root, digest)
	if err := ensurePrivateDirectory(dir); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "claudodex")
	if existing, err := readPrivateFile(path, true); err == nil && sha256Hex(existing) == digest {
		return path, nil
	}
	if err := atomicPrivateFile(path, data, 0o700); err != nil {
		return "", err
	}
	return path, nil
}
