package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEnsureLoggedInRejectsIncompleteAuth(t *testing.T) {
	home := t.TempDir()
	if err := NewStore(home).Save(File{AuthMode: "chatgpt", Issuer: Issuer, ClientID: ClientID}); err != nil {
		t.Fatal(err)
	}

	_, err := EnsureLoggedIn(context.Background(), home, false)
	if err == nil || !strings.Contains(err.Error(), "missing refresh_token") {
		t.Fatalf("EnsureLoggedIn error = %v", err)
	}
}

func TestEnsureLoggedInRefreshesExpiredAccessToken(t *testing.T) {
	home := t.TempDir()
	store := NewStore(home)
	if err := store.Save(File{
		AuthMode: "chatgpt",
		Issuer:   Issuer,
		ClientID: ClientID,
		Tokens: Tokens{
			AccessToken:          fakeJWT(t, map[string]any{"exp": time.Now().Add(-time.Hour).Unix()}),
			RefreshToken:         "old-refresh",
			AccessTokenExpiresAt: time.Now().Add(-time.Hour).Unix(),
		},
	}); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"access_token":"new-access","refresh_token":"new-refresh"}`))
	}))
	defer server.Close()
	oldEndpoint := defaultTokenEndpoint
	defaultTokenEndpoint = server.URL
	defer func() { defaultTokenEndpoint = oldEndpoint }()

	file, err := EnsureLoggedIn(context.Background(), home, false)
	if err != nil {
		t.Fatal(err)
	}
	if file.Tokens.AccessToken != "new-access" || file.Tokens.RefreshToken != "new-refresh" {
		t.Fatalf("tokens = %#v", file.Tokens)
	}
}
