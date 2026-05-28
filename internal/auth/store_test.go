package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreSaveLoadDeletePermissions(t *testing.T) {
	home := t.TempDir()
	store := NewStore(home)
	file := File{
		AuthMode:    "chatgpt",
		Issuer:      Issuer,
		ClientID:    ClientID,
		LastRefresh: time.Unix(1770000000, 0).UTC(),
		Tokens: Tokens{
			AccessToken:  "access",
			RefreshToken: "refresh",
			IDToken:      "id",
			AccountID:    "acct",
		},
	}
	if err := store.Save(file); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(home, ".claudodex", "auth.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("auth mode = %o, want 600", got)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Tokens.RefreshToken != "refresh" || loaded.Tokens.AccountID != "acct" {
		t.Fatalf("unexpected loaded auth: %#v", loaded)
	}

	if err := store.Delete(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); err != ErrNotLoggedIn {
		t.Fatalf("Load after delete error = %v, want ErrNotLoggedIn", err)
	}
}

func TestStoreUsesClaudodexHomeEnv(t *testing.T) {
	envHome := t.TempDir()
	t.Setenv("CLAUDODEX_HOME", envHome)
	store := NewStore("")
	file := File{
		AuthMode: "chatgpt",
		Issuer:   Issuer,
		ClientID: ClientID,
		Tokens: Tokens{
			AccessToken:  "access",
			RefreshToken: "refresh",
		},
	}
	if err := store.Save(file); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(envHome, "auth.json")); err != nil {
		t.Fatal(err)
	}
}

func TestInstallationIDReset(t *testing.T) {
	home := t.TempDir()
	first, err := InstallationID(home)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ResetInstallationID(home)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("reset should change installation id")
	}
	loaded, err := InstallationID(home)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != second {
		t.Fatalf("loaded installation id = %q, want %q", loaded, second)
	}
}

func TestInstallationIDUsesClaudodexHomeEnv(t *testing.T) {
	envHome := t.TempDir()
	t.Setenv("CLAUDODEX_HOME", envHome)
	if _, err := ResetInstallationID(""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(envHome, "install.json")); err != nil {
		t.Fatal(err)
	}
}
