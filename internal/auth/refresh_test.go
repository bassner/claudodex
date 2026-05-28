package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRefreshRotatesToken(t *testing.T) {
	home := t.TempDir()
	store := NewStore(home)
	if err := store.Save(File{
		AuthMode:    "chatgpt",
		Issuer:      Issuer,
		ClientID:    ClientID,
		LastRefresh: time.Unix(1, 0).UTC(),
		Tokens: Tokens{
			AccessToken:  "old-access",
			RefreshToken: "old-refresh",
			IDToken:      "old-id",
		},
	}); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["refresh_token"] != "old-refresh" {
			t.Fatalf("refresh_token = %q", body["refresh_token"])
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token":  "new-access",
			"refresh_token": "new-refresh",
			"id_token":      "new-id",
		})
	}))
	defer server.Close()

	file, err := NewRefresher(store, server.Client(), server.URL).Refresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if file.Tokens.AccessToken != "new-access" || file.Tokens.RefreshToken != "new-refresh" {
		t.Fatalf("tokens = %#v", file.Tokens)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Tokens.RefreshToken != "new-refresh" {
		t.Fatalf("stored refresh token = %q", loaded.Tokens.RefreshToken)
	}
}

func TestRefreshKeepsRefreshTokenWhenOmitted(t *testing.T) {
	home := t.TempDir()
	store := NewStore(home)
	if err := store.Save(File{
		AuthMode: "chatgpt",
		Issuer:   Issuer,
		ClientID: ClientID,
		Tokens: Tokens{
			AccessToken:  "old-access",
			RefreshToken: "old-refresh",
		},
	}); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "new-access"})
	}))
	defer server.Close()

	file, err := NewRefresher(store, server.Client(), server.URL).Refresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if file.Tokens.AccessToken != "new-access" {
		t.Fatalf("access token = %q", file.Tokens.AccessToken)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Tokens.RefreshToken != "old-refresh" {
		t.Fatalf("stored refresh token changed to %q", loaded.Tokens.RefreshToken)
	}
}

func TestRefreshDeletesAuthOnPermanentFailure(t *testing.T) {
	home := t.TempDir()
	store := NewStore(home)
	if err := store.Save(File{
		AuthMode: "chatgpt",
		Issuer:   Issuer,
		ClientID: ClientID,
		Tokens: Tokens{
			AccessToken:  "old-access",
			RefreshToken: "old-refresh",
		},
	}); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"code": "refresh_token_reused"},
		})
	}))
	defer server.Close()

	if _, err := NewRefresher(store, server.Client(), server.URL).Refresh(context.Background()); err == nil {
		t.Fatal("expected permanent refresh error")
	}
	if _, err := store.Load(); err != ErrNotLoggedIn {
		t.Fatalf("Load after permanent refresh failure = %v, want ErrNotLoggedIn", err)
	}
}

func TestRefreshDeletesAuthOnFlatPermanentFailure(t *testing.T) {
	home := t.TempDir()
	store := NewStore(home)
	if err := store.Save(File{
		AuthMode: "chatgpt",
		Issuer:   Issuer,
		ClientID: ClientID,
		Tokens: Tokens{
			AccessToken:  "old-access",
			RefreshToken: "old-refresh",
		},
	}); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "refresh_token_expired",
		})
	}))
	defer server.Close()

	_, err := NewRefresher(store, server.Client(), server.URL).Refresh(context.Background())
	if !errors.Is(err, ErrPermanentRefreshFailure) {
		t.Fatalf("Refresh error = %v, want ErrPermanentRefreshFailure", err)
	}
	if _, err := store.Load(); err != ErrNotLoggedIn {
		t.Fatalf("Load after permanent refresh failure = %v, want ErrNotLoggedIn", err)
	}
}

func TestRefreshPermanentFailureDoesNotDeleteNewerToken(t *testing.T) {
	home := t.TempDir()
	store := NewStore(home)
	if err := store.Save(File{
		AuthMode: "chatgpt",
		Issuer:   Issuer,
		ClientID: ClientID,
		Tokens: Tokens{
			AccessToken:  "old-access",
			RefreshToken: "old-refresh",
		},
	}); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := store.Save(File{
			AuthMode: "chatgpt",
			Issuer:   Issuer,
			ClientID: ClientID,
			Tokens: Tokens{
				AccessToken:  "new-access",
				RefreshToken: "new-refresh",
			},
		}); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
	}))
	defer server.Close()

	file, err := NewRefresher(store, server.Client(), server.URL).Refresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if file.Tokens.RefreshToken != "new-refresh" {
		t.Fatalf("returned refresh token = %q, want new-refresh", file.Tokens.RefreshToken)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Tokens.RefreshToken != "new-refresh" {
		t.Fatalf("stored refresh token = %q, want new-refresh", loaded.Tokens.RefreshToken)
	}
}

func TestRefreshWaitsForAuthFileLock(t *testing.T) {
	oldRetry := refreshLockRetry
	refreshLockRetry = time.Millisecond
	defer func() { refreshLockRetry = oldRetry }()

	home := t.TempDir()
	store := NewStore(home)
	if err := store.Save(File{
		AuthMode: "chatgpt",
		Issuer:   Issuer,
		ClientID: ClientID,
		Tokens: Tokens{
			AccessToken:  "old-access",
			RefreshToken: "old-refresh",
		},
	}); err != nil {
		t.Fatal(err)
	}
	path, err := store.path()
	if err != nil {
		t.Fatal(err)
	}
	lockPath := path + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, []byte("locked"), 0o600); err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = os.Remove(lockPath)
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "new-access"})
	}))
	defer server.Close()

	start := time.Now()
	file, err := NewRefresher(store, server.Client(), server.URL).Refresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) < 15*time.Millisecond {
		t.Fatal("refresh did not wait for auth file lock")
	}
	if file.Tokens.AccessToken != "new-access" {
		t.Fatalf("access token = %q, want new-access", file.Tokens.AccessToken)
	}
}
