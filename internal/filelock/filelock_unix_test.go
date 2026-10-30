//go:build unix

package filelock

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestAcquireSerializesAndCancels(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", ".lock")
	release, err := Acquire(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	if _, err := Acquire(ctx, path); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("concurrent acquire error = %v", err)
	}
	release()
	release()
	again, err := Acquire(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	again()
	info, _ := os.Stat(filepath.Dir(path))
	if info.Mode().Perm() != 0o700 {
		t.Fatal("parent directory is not private")
	}
}

func TestAcquireRejectsSymlinksAndUnsafePermissions(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Acquire(context.Background(), link); err == nil {
		t.Fatal("symlink lock accepted")
	}
	if err := os.Chmod(target, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Acquire(context.Background(), target); err == nil {
		t.Fatal("nonprivate lock accepted")
	}
}

func TestAcquireDescriptorIsCloseOnExec(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".lock")
	release, err := Acquire(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	// Find the live descriptor by inode. F_GETFD verifies the actual flag,
	// rather than only checking the source's open options.
	var lockStat unix.Stat_t
	if err := unix.Stat(path, &lockStat); err != nil {
		t.Fatal(err)
	}
	for fd := 3; fd < 1024; fd++ {
		var stat unix.Stat_t
		if unix.Fstat(fd, &stat) == nil && stat.Dev == lockStat.Dev && stat.Ino == lockStat.Ino {
			flags, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0)
			if err != nil || flags&unix.FD_CLOEXEC == 0 {
				t.Fatal("lock descriptor lacks close-on-exec")
			}
			return
		}
	}
	t.Fatal("live lock descriptor not found")
}
