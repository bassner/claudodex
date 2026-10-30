package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func prepareSource(version, out string) (string, error) {
	status, err := exec.Command("git", "status", "--porcelain", "--untracked-files=normal").Output()
	if err != nil {
		return "", err
	}
	if len(status) != 0 {
		return "", errors.New("source packaging requires a clean git checkout")
	}
	tmp, err := os.MkdirTemp("", "clx-source-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	archive, err := exec.Command("git", "archive", "--format=tar", "HEAD").Output()
	if err != nil {
		return "", err
	}
	if err := extractSource(archive, tmp); err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(tmp, "vendor")); !errors.Is(err, os.ErrNotExist) {
		return "", errors.New("vendor must not be checked into the repository")
	}
	cmd := exec.Command("go", "mod", "vendor")
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off", "GOFLAGS=-mod=mod")
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("vendor source dependencies: %w\n%s", err, output)
	}
	data, err := sourceTar(tmp, "claudodex-"+version)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(out, sourceName(version))
	return path, writeNew(path, data, 0644)
}

func extractSource(data []byte, target string) error {
	reader := tar.NewReader(bytes.NewReader(data))
	for {
		h, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(h.Name)
		if filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
			return errors.New("unsafe git archive path")
		}
		path := filepath.Join(target, name)
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(h.Mode)&0777)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(f, reader)
			closeErr := f.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		case tar.TypeXGlobalHeader:
			continue
		default:
			return fmt.Errorf("unsupported git archive entry %q", h.Name)
		}
	}
}

// Sort entries and reset timestamps/owners, including go mod vendor output, so
// a failed release can be resumed without replacing an already uploaded asset.
func sourceTar(root, prefix string) ([]byte, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != root {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	var data bytes.Buffer
	gz := gzip.NewWriter(&data)
	tarWriter := tar.NewWriter(gz)
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return nil, errors.New("source archive contains a non-regular file")
		}
		h, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil, err
		}
		h.Name = prefix + "/" + filepath.ToSlash(rel)
		h.ModTime = time.Unix(0, 0)
		h.AccessTime, h.ChangeTime = time.Time{}, time.Time{}
		h.Uid, h.Gid, h.Uname, h.Gname = 0, 0, "", ""
		if err := tarWriter.WriteHeader(h); err != nil {
			return nil, err
		}
		if info.IsDir() {
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		_, copyErr := io.Copy(tarWriter, f)
		closeErr := f.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
	}
	if err := tarWriter.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return data.Bytes(), nil
}
