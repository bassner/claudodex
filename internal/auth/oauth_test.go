package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAuthorizeURL(t *testing.T) {
	raw := AuthorizeURL("http://127.0.0.1:1234/auth/callback", "state", "challenge")
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	q := parsed.Query()
	want := map[string]string{
		"client_id":                  ClientID,
		"response_type":              "code",
		"redirect_uri":               "http://127.0.0.1:1234/auth/callback",
		"scope":                      OAuthScopes,
		"code_challenge":             "challenge",
		"code_challenge_method":      "S256",
		"state":                      "state",
		"id_token_add_organizations": "true",
		"codex_cli_simplified_flow":  "true",
		"originator":                 "opencode",
	}
	for key, value := range want {
		if got := q.Get(key); got != value {
			t.Fatalf("%s = %q, want %q", key, got, value)
		}
	}
	if !strings.Contains(raw, "scope=openid+profile+email+offline_access") {
		t.Fatalf("scope should use + separators in raw authorize URL: %s", raw)
	}
}

func TestGeneratePKCE(t *testing.T) {
	pkce, err := GeneratePKCE()
	if err != nil {
		t.Fatal(err)
	}
	if pkce.Verifier == "" || pkce.Challenge == "" || pkce.Verifier == pkce.Challenge {
		t.Fatalf("invalid pkce: %#v", pkce)
	}
}

func TestLoginLoopbackAndExchange(t *testing.T) {
	home := t.TempDir()
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Fatalf("grant_type = %q", r.Form.Get("grant_type"))
		}
		if r.Form.Get("client_id") != ClientID {
			t.Fatalf("client_id = %q", r.Form.Get("client_id"))
		}
		if r.Form.Get("code") != "code-123" {
			t.Fatalf("code = %q", r.Form.Get("code"))
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token":  "access",
			"refresh_token": "refresh",
			"id_token": fakeJWT(t, map[string]any{
				"https://api.openai.com/profile": map[string]any{"email": "pat@example.com"},
				"https://api.openai.com/auth": map[string]any{
					"chatgpt_plan_type":  "pro",
					"chatgpt_account_id": "account-123",
				},
			}),
			"token_type": "Bearer",
		})
	}))
	defer tokenServer.Close()

	file, err := Login(context.Background(), LoginOptions{
		Home:          home,
		TokenEndpoint: tokenServer.URL,
		Timeout:       time.Second,
		CallbackPorts: []int{0},
		OpenBrowser: func(raw string) error {
			parsed, err := url.Parse(raw)
			if err != nil {
				return err
			}
			redirect := parsed.Query().Get("redirect_uri")
			state := parsed.Query().Get("state")
			go func() {
				_, _ = http.Get(redirect + "?code=code-123&state=" + url.QueryEscape(state))
			}()
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if file.Tokens.AccessToken != "access" || file.Tokens.RefreshToken != "refresh" || file.Tokens.AccountID != "account-123" || file.Tokens.PlanType != "pro" {
		t.Fatalf("tokens = %#v", file.Tokens)
	}
	loaded, err := NewStore(home).Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Tokens.RefreshToken != "refresh" {
		t.Fatalf("stored refresh = %q", loaded.Tokens.RefreshToken)
	}
}

func TestCallbackHandlerIgnoresStrayBadRequests(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{name: "state mismatch", url: "/auth/callback?state=wrong&code=code"},
		{name: "missing code", url: "/auth/callback?state=state"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codeCh := make(chan string, 1)
			errCh := make(chan error, 1)
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rec := httptest.NewRecorder()

			callbackHandler("state", codeCh, errCh).ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
			select {
			case err := <-errCh:
				t.Fatalf("unexpected callback error: %v", err)
			default:
			}
		})
	}
}

func TestCallbackHandlerReportsOAuthProviderError(t *testing.T) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	req := httptest.NewRequest(http.MethodGet, "/auth/callback?state=state&error=access_denied", nil)
	rec := httptest.NewRecorder()

	callbackHandler("state", codeCh, errCh).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("nil callback error")
		}
	default:
		t.Fatal("OAuth provider error was not reported")
	}
}
