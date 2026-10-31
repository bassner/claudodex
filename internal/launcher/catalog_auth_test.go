package launcher

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bassner/claudodex/internal/auth"
	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
)

func catalogAuthFixture() []codex.ModelInfo {
	models := make([]codex.ModelInfo, 0, 4)
	for _, slug := range modelconfig.Default().RequiredModels() {
		models = append(models, codex.ModelInfo{Slug: slug, DisplayName: slug, ContextWindow: 500000, MaxContextWindow: 872000,
			SupportedInAPI: true, Visibility: "list", DefaultReasoningLevel: "medium",
			SupportedReasoningLevels: []codex.ReasoningEffortPreset{{Effort: "low"}, {Effort: "medium"}, {Effort: "high"}, {Effort: "max"}}})
	}
	return models
}

func catalogAuthIDToken(t *testing.T, user, account string) string {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"iss": auth.Issuer, "exp": time.Now().Add(time.Hour).Unix(), "https://api.openai.com/auth": map[string]any{
		"chatgpt_user_id": user, "chatgpt_account_id": account, "chatgpt_account_is_fedramp": true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`)) + "." + base64.RawURLEncoding.EncodeToString(payload) + ".fixture"
}

func TestCatalog401UsesSharedProviderAndRetriesFreshMetadataOnce(t *testing.T) {
	for _, mode := range []string{"refresh-success", "repeated-401", "changed-user", "changed-account", "already-rotated", "403-no-refresh", "no-provider"} {
		t.Run(mode, func(t *testing.T) {
			home := t.TempDir()
			store := auth.NewStore(home)
			initial := auth.File{AuthMode: "chatgpt", Issuer: auth.Issuer, ClientID: auth.ClientID,
				Tokens: auth.Tokens{AccessToken: "rejected-access", RefreshToken: "synthetic-refresh", UserID: "user", AccountID: "account", AccessTokenExpiresAt: time.Now().Add(time.Hour).Unix()}}
			if err := store.Save(initial); err != nil {
				t.Fatal(err)
			}
			var metadataRequests, refreshRequests atomic.Int32
			refreshUser, refreshAccount := "user", "account"
			if mode == "changed-user" {
				refreshUser = "other-user"
			}
			if mode == "changed-account" {
				refreshAccount = "other-account"
			}
			idToken := catalogAuthIDToken(t, refreshUser, refreshAccount)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/token":
					refreshRequests.Add(1)
					if err := json.NewEncoder(w).Encode(map[string]string{"access_token": "refreshed-access", "refresh_token": "rotated-synthetic-refresh", "id_token": idToken}); err != nil {
						t.Error(err)
					}
				case "/codex/models":
					requestNumber := metadataRequests.Add(1)
					if r.URL.Query().Get("client_version") == "" {
						t.Error("metadata request omitted discovery protocol version")
					}
					if mode == "403-no-refresh" {
						w.WriteHeader(http.StatusForbidden)
						return
					}
					if requestNumber == 1 {
						if r.Header.Get("authorization") != "Bearer rejected-access" {
							t.Error("first metadata request did not use initial credentials")
						}
						if mode == "already-rotated" {
							rotated := initial
							rotated.Tokens.AccessToken = "authoritative-rotated-access"
							if err := store.Save(rotated); err != nil {
								t.Error(err)
							}
						}
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					wantAccess := "Bearer refreshed-access"
					if mode == "already-rotated" {
						wantAccess = "Bearer authoritative-rotated-access"
					}
					if r.Header.Get("authorization") != wantAccess {
						t.Error("retry did not use the shared provider's current access token")
					}
					if r.Header.Get("chatgpt-account-id") != "account" {
						t.Error("retry used unexpected account identity")
					}
					if mode != "already-rotated" && (r.Header.Get("x-openai-fedramp") != "true" || r.Header.Get("x-openai-internal-codex-residency") != "us") {
						t.Error("retry omitted refreshed residency constraints")
					}
					if mode == "repeated-401" {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					if err := json.NewEncoder(w).Encode(codex.ModelsResponse{Models: catalogAuthFixture()}); err != nil {
						t.Error(err)
					}
				default:
					t.Errorf("unexpected synthetic endpoint: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer upstream.Close()
			provider := auth.NewProvider(store, upstream.Client(), upstream.URL+"/token")
			cfg := Config{Home: home, Version: "dev", CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), TokenEndpoint: "http://127.0.0.1:1/must-not-use", Models: modelconfig.Default()}
			providers := []*auth.Provider{provider}
			if mode == "no-provider" {
				providers = nil
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			snapshot, err := ResolveCodexModels(ctx, cfg, initial, providers...)
			wantMetadata, wantRefresh := int32(2), int32(1)
			switch mode {
			case "refresh-success", "already-rotated":
				if err != nil {
					t.Fatal(err)
				}
				if snapshot.FromCache || len(snapshot.Models) != 4 || snapshot.Config != modelconfig.Default() {
					t.Fatal("401 recovery did not resolve the fresh four-role catalog")
				}
				if mode == "already-rotated" {
					wantRefresh = 0
				}
			case "repeated-401":
				var rejected *codex.UpstreamError
				if !errors.As(err, &rejected) || rejected.Status != http.StatusUnauthorized {
					t.Fatalf("second rejection error=%v", err)
				}
			case "changed-user", "changed-account":
				wantMetadata = 1
				if err == nil || !strings.Contains(err.Error(), "login changed") {
					t.Fatalf("identity change error=%v", err)
				}
			case "403-no-refresh":
				wantMetadata, wantRefresh = 1, 0
				var forbidden *codex.UpstreamError
				if !errors.As(err, &forbidden) || forbidden.Status != http.StatusForbidden {
					t.Fatalf("403 error=%v", err)
				}
			case "no-provider":
				wantMetadata, wantRefresh = 1, 0
				var rejected *codex.UpstreamError
				if !errors.As(err, &rejected) || rejected.Status != http.StatusUnauthorized {
					t.Fatalf("metadata-only helper error=%v", err)
				}
			}
			if metadataRequests.Load() != wantMetadata || refreshRequests.Load() != wantRefresh {
				t.Fatalf("metadata requests=%d (want %d), OAuth refreshes=%d (want %d)", metadataRequests.Load(), wantMetadata, refreshRequests.Load(), wantRefresh)
			}
		})
	}
}
