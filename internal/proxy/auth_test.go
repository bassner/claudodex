package proxy

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bassner/claudodex/internal/auth"
	"github.com/bassner/claudodex/internal/codex"
	"github.com/gorilla/websocket"
)

func saveProviderAuth(t *testing.T, store auth.Store, token, user, account string) {
	t.Helper()
	if err := store.Save(auth.File{AuthMode: "chatgpt", Tokens: auth.Tokens{
		AccessToken: token, RefreshToken: "synthetic-refresh", UserID: user, AccountID: account,
		AccessTokenExpiresAt: time.Now().Add(time.Hour).Unix(),
	}}); err != nil {
		t.Fatal(err)
	}
}

func TestProviderIdentityChangeStopsNextHTTPAndUsageRequest(t *testing.T) {
	for _, changed := range []string{"user", "account"} {
		t.Run(changed, func(t *testing.T) {
			home := t.TempDir()
			store := auth.NewStore(home)
			saveProviderAuth(t, store, "first-access", "user", "account")
			var calls atomic.Int32
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path == "/wham/usage" {
					_, _ = w.Write([]byte(`{}`))
					return
				}
				w.Header().Set("content-type", "text/event-stream")
				_, _ = w.Write([]byte("event: response.completed\ndata: {\"type\":\"response.completed\"}\n\n"))
			}))
			defer upstream.Close()
			provider := auth.NewProvider(store, nil, "")
			server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthProvider: provider, AuthUserID: "user", AuthAccountID: "account"})
			defer server.Close()
			r := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
			resp, _, err := server.createCodexResponse(r, codex.Request{Model: "fixture"}, codex.Route{}, 2)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if _, err := server.fetchCodexUsage(r); err != nil {
				t.Fatal(err)
			}
			if calls.Load() != 2 {
				t.Fatalf("initial calls=%d", calls.Load())
			}
			user, account := "user", "account"
			if changed == "user" {
				user = "other-user"
			} else {
				account = "other-account"
			}
			saveProviderAuth(t, store, "changed-access", user, account)
			if _, _, err := server.createCodexResponse(r, codex.Request{Model: "fixture"}, codex.Route{}, 2); !errors.Is(err, auth.ErrInvalidAuth) {
				t.Fatalf("changed identity response error=%v", err)
			}
			if _, err := server.fetchCodexUsage(r); !errors.Is(err, auth.ErrInvalidAuth) {
				t.Fatalf("changed identity usage error=%v", err)
			}
			if calls.Load() != 2 {
				t.Fatal("changed identity reached upstream")
			}
		})
	}
}

func TestProviderIdentityChangeClosesWebSocketBeforeReuse(t *testing.T) {
	t.Setenv("CLAUDODEX_FORCE_CODEX_WEBSOCKET", "true")
	home := t.TempDir()
	store := auth.NewStore(home)
	saveProviderAuth(t, store, "first-access", "user", "account")
	var handshakes, requests atomic.Int32
	upgrader := websocket.Upgrader{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handshakes.Add(1)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
			requests.Add(1)
			if err := conn.WriteJSON(map[string]any{"type": "response.completed", "response": map[string]any{"id": "fixture-response"}}); err != nil {
				return
			}
		}
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, AuthProvider: auth.NewProvider(store, nil, ""), AuthUserID: "user", AuthAccountID: "account"})
	defer server.Close()
	r := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	req := codex.Request{Model: "fixture", Tools: []codex.Tool{{Name: "fixture_tool"}}}
	route := codex.Route{ThreadID: "thread"}
	resp, _, err := server.createCodexResponse(r, req, route, 2)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if !server.hasWebSocket("thread") {
		t.Fatal("first request did not retain WebSocket")
	}
	saveProviderAuth(t, store, "changed-access", "other-user", "account")
	if _, _, err := server.createCodexResponse(r, req, route, 2); !errors.Is(err, auth.ErrInvalidAuth) {
		t.Fatalf("changed identity error=%v", err)
	}
	if handshakes.Load() != 1 || requests.Load() != 1 {
		t.Fatalf("changed identity reused network: handshakes=%d requests=%d", handshakes.Load(), requests.Load())
	}
	if server.hasWebSocket("thread") {
		t.Fatal("changed identity retained WebSocket")
	}
}

func TestProvider401UsesRejectedTokenWithoutRefreshingAlreadyRotatedAuth(t *testing.T) {
	for _, usage := range []bool{false, true} {
		t.Run(map[bool]string{false: "responses", true: "usage"}[usage], func(t *testing.T) {
			home := t.TempDir()
			store := auth.NewStore(home)
			saveProviderAuth(t, store, "rejected-access", "user", "account")
			var refreshes, calls atomic.Int32
			tokens := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				refreshes.Add(1)
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer tokens.Close()
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					saveProviderAuth(t, store, "already-rotated-access", "user", "account")
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if r.Header.Get("authorization") != "Bearer already-rotated-access" {
					t.Error("retry did not use authoritative rotated token")
				}
				_, _ = w.Write([]byte(`{}`))
			}))
			defer upstream.Close()
			provider := auth.NewProvider(store, tokens.Client(), tokens.URL)
			server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthProvider: provider, AuthUserID: "user", AuthAccountID: "account"})
			defer server.Close()
			r := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
			if usage {
				if _, err := server.fetchCodexUsage(r); err != nil {
					t.Fatal(err)
				}
			} else {
				resp, _, err := server.createCodexResponse(r, codex.Request{Model: "fixture"}, codex.Route{}, 2)
				if err != nil {
					t.Fatal(err)
				}
				_ = resp.Body.Close()
			}
			if refreshes.Load() != 0 || calls.Load() != 2 {
				t.Fatalf("refreshes=%d calls=%d", refreshes.Load(), calls.Load())
			}
		})
	}
}

func TestProvider401RefreshUsesConfiguredSharedProvider(t *testing.T) {
	home := t.TempDir()
	store := auth.NewStore(home)
	saveProviderAuth(t, store, "rejected-access", "user", "account")
	var refreshes, calls atomic.Int32
	tokens := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshes.Add(1)
		_, _ = w.Write([]byte(`{"access_token":"refreshed-access","refresh_token":"new-synthetic-refresh"}`))
	}))
	defer tokens.Close()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("authorization") != "Bearer refreshed-access" {
			t.Error("retry did not use refreshed access")
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer upstream.Close()
	provider := auth.NewProvider(store, tokens.Client(), tokens.URL)
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthProvider: provider, TokenEndpoint: "http://127.0.0.1:1/must-not-use", AuthUserID: "user", AuthAccountID: "account"})
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r := httptest.NewRequest(http.MethodPost, "/v1/messages", nil).WithContext(ctx)
	resp, _, err := server.createCodexResponse(r, codex.Request{Model: "fixture"}, codex.Route{}, 2)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if refreshes.Load() != 1 || calls.Load() != 2 {
		t.Fatalf("refreshes=%d calls=%d", refreshes.Load(), calls.Load())
	}
}

func TestProviderResumedWebSocket401RefreshesThroughSameProvider(t *testing.T) {
	t.Setenv("CLAUDODEX_FORCE_CODEX_WEBSOCKET", "true")
	home := t.TempDir()
	store := auth.NewStore(home)
	saveProviderAuth(t, store, "rejected-access", "user", "account")
	var refreshes, handshakes atomic.Int32
	tokens := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshes.Add(1)
		_, _ = w.Write([]byte(`{"access_token":"refreshed-access","refresh_token":"new-synthetic-refresh"}`))
	}))
	defer tokens.Close()
	upgrader := websocket.Upgrader{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !websocket.IsWebSocketUpgrade(r) {
			t.Error("resumed WebSocket fell back to HTTP")
			w.WriteHeader(500)
			return
		}
		if handshakes.Add(1) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("authorization") != "Bearer refreshed-access" {
			t.Error("resumed WebSocket did not use refreshed token")
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		_ = conn.WriteJSON(map[string]any{"type": "response.completed", "response": map[string]any{"id": "fixture-response"}})
	}))
	defer upstream.Close()
	provider := auth.NewProvider(store, tokens.Client(), tokens.URL)
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, AuthProvider: provider, AuthUserID: "user", AuthAccountID: "account"})
	defer server.Close()
	r := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	req := codex.Request{Model: "fixture", PreviousResponseID: "previous-response", Tools: []codex.Tool{{Name: "fixture_tool"}}}
	resp, _, err := server.createCodexResponse(r, req, codex.Route{ThreadID: "thread"}, 2)
	if err != nil {
		t.Fatalf("resumed WebSocket auth recovery failed: %v (refreshes=%d handshakes=%d)", err, refreshes.Load(), handshakes.Load())
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if refreshes.Load() != 1 || handshakes.Load() != 2 {
		t.Fatalf("refreshes=%d handshakes=%d", refreshes.Load(), handshakes.Load())
	}
}
