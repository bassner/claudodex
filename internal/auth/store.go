package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultDirName = ".claudodex"
	authFileName   = "auth.json"
)

var ErrNotLoggedIn = errors.New("not logged in")

type Store struct {
	home string
}

type File struct {
	AuthMode    string    `json:"auth_mode"`
	Issuer      string    `json:"issuer"`
	ClientID    string    `json:"client_id"`
	LastRefresh time.Time `json:"last_refresh"`
	Tokens      Tokens    `json:"tokens"`
}

type Tokens struct {
	AccessToken             string `json:"access_token"`
	RefreshToken            string `json:"refresh_token"`
	IDToken                 string `json:"id_token"`
	AccountID               string `json:"account_id,omitempty"`
	PlanType                string `json:"plan_type,omitempty"`
	Email                   string `json:"email,omitempty"`
	UserID                  string `json:"user_id,omitempty"`
	AccessTokenExpiresAt    int64  `json:"access_token_expires_at,omitempty"`
	IDTokenExpiresAt        int64  `json:"id_token_expires_at,omitempty"`
	ChatGPTAccountIsFedRAMP bool   `json:"chatgpt_account_is_fedramp"`
}

func NewStore(home string) Store {
	return Store{home: home}
}

func (s Store) Load() (File, error) {
	path, err := s.path()
	if err != nil {
		return File{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return File{}, ErrNotLoggedIn
		}
		return File{}, err
	}
	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return File{}, fmt.Errorf("parse auth file: %w", err)
	}
	return file, nil
}

func (s Store) Save(file File) error {
	path, err := s.path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".auth-*.json")
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

func (s Store) Delete() error {
	path, err := s.path()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s Store) path() (string, error) {
	dir, err := dataDir(s.home)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, authFileName), nil
}

func DataDir(home string) (string, error) {
	return dataDir(home)
}

func dataDir(home string) (string, error) {
	if home != "" {
		return filepath.Join(home, defaultDirName), nil
	}
	if env := os.Getenv("CLAUDODEX_HOME"); env != "" {
		return env, nil
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userHome, defaultDirName), nil
}
