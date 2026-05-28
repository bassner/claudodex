package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRevokeAndDelete(t *testing.T) {
	home := t.TempDir()
	store := NewStore(home)
	if err := store.Save(File{
		AuthMode: "chatgpt",
		Issuer:   Issuer,
		ClientID: ClientID,
		Tokens: Tokens{
			AccessToken:  "access",
			RefreshToken: "refresh",
		},
	}); err != nil {
		t.Fatal(err)
	}

	var sawRefresh bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		sawRefresh = body["token"] == "refresh" && body["token_type_hint"] == "refresh_token" && body["client_id"] == ClientID
	}))
	defer server.Close()

	if err := RevokeAndDelete(context.Background(), store, server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
	if !sawRefresh {
		t.Fatal("did not revoke refresh token")
	}
	if _, err := store.Load(); err != ErrNotLoggedIn {
		t.Fatalf("Load after revoke = %v, want ErrNotLoggedIn", err)
	}
}

func TestRevokeAndDeleteDeletesOnRevokeFailure(t *testing.T) {
	home := t.TempDir()
	store := NewStore(home)
	if err := store.Save(File{
		AuthMode: "chatgpt",
		Issuer:   Issuer,
		ClientID: ClientID,
		Tokens: Tokens{
			AccessToken:  "access",
			RefreshToken: "refresh",
		},
	}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	if err := RevokeAndDelete(context.Background(), store, server.Client(), server.URL); !errors.Is(err, ErrRevokeFailed) {
		t.Fatalf("RevokeAndDelete error = %v, want ErrRevokeFailed", err)
	}
	if _, err := store.Load(); err != ErrNotLoggedIn {
		t.Fatalf("Load after failed revoke = %v, want ErrNotLoggedIn", err)
	}
}
