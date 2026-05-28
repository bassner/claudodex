package auth

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const installFileName = "install.json"

type installFile struct {
	InstallationID string `json:"installation_id"`
}

func ResetInstallationID(home string) (string, error) {
	id, err := uuidV4()
	if err != nil {
		return "", err
	}
	if err := saveInstallationID(home, id); err != nil {
		return "", err
	}
	return id, nil
}

func InstallationID(home string) (string, error) {
	path, err := installPath(home)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err == nil {
		var file installFile
		if err := json.Unmarshal(data, &file); err == nil && file.InstallationID != "" {
			return file.InstallationID, nil
		}
	}
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	return ResetInstallationID(home)
}

func saveInstallationID(home, id string) error {
	path, err := installPath(home)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(installFile{InstallationID: id}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".install-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func installPath(home string) (string, error) {
	dir, err := dataDir(home)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, installFileName), nil
}

func uuidV4() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
