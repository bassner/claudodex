package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const (
	ClientID                  = "app_EMoamEEZ73f0CkXaXp7hrann"
	Issuer                    = "https://auth.openai.com"
	AuthorizePath             = "https://auth.openai.com/oauth/authorize"
	TokenEndpoint             = "https://auth.openai.com/oauth/token"
	OAuthScopes               = "openid profile email offline_access"
	DefaultOAuthCallbackPort  = 1455
	FallbackOAuthCallbackPort = 1457
)

type LoginOptions struct {
	Home              string
	HTTPClient        *http.Client
	AuthorizeEndpoint string
	TokenEndpoint     string
	OpenBrowser       func(string) error
	Timeout           time.Duration
	CallbackPorts     []int
}

type PKCE struct {
	Verifier  string
	Challenge string
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
}

func GeneratePKCE() (PKCE, error) {
	verifier, err := randomURLToken(48)
	if err != nil {
		return PKCE{}, err
	}
	sum := sha256.Sum256([]byte(verifier))
	return PKCE{
		Verifier:  verifier,
		Challenge: base64.RawURLEncoding.EncodeToString(sum[:]),
	}, nil
}

func GenerateState() (string, error) {
	return randomURLToken(32)
}

func AuthorizeURL(redirectURI, state, challenge string) string {
	return authorizeURL(AuthorizePath, redirectURI, state, challenge)
}

func Login(ctx context.Context, opts LoginOptions) (File, error) {
	if opts.HTTPClient == nil {
		opts.HTTPClient = http.DefaultClient
	}
	if opts.AuthorizeEndpoint == "" {
		opts.AuthorizeEndpoint = AuthorizePath
	}
	if opts.TokenEndpoint == "" {
		opts.TokenEndpoint = TokenEndpoint
	}
	if opts.OpenBrowser == nil {
		opts.OpenBrowser = OpenBrowser
	}
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Minute
	}

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	pkce, err := GeneratePKCE()
	if err != nil {
		return File{}, err
	}
	state, err := GenerateState()
	if err != nil {
		return File{}, err
	}

	listener, err := listenCallback(opts.CallbackPorts)
	if err != nil {
		return File{}, err
	}
	defer listener.Close()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		return File{}, err
	}
	redirectURI := "http://localhost:" + port + "/auth/callback"
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	server := &http.Server{Handler: callbackHandler(state, codeCh, errCh)}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			sendOAuthCallbackError(errCh, err)
		}
	}()
	defer server.Close()

	if err := opts.OpenBrowser(authorizeURL(opts.AuthorizeEndpoint, redirectURI, state, pkce.Challenge)); err != nil {
		return File{}, fmt.Errorf("open browser: %w", err)
	}

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return File{}, err
	case <-ctx.Done():
		return File{}, ctx.Err()
	}

	tokens, err := exchangeCode(ctx, opts.HTTPClient, opts.TokenEndpoint, code, redirectURI, pkce.Verifier)
	if err != nil {
		return File{}, err
	}
	if tokens.RefreshToken == "" {
		return File{}, fmt.Errorf("token response missing refresh_token")
	}

	file := File{
		AuthMode:    "chatgpt",
		Issuer:      Issuer,
		ClientID:    ClientID,
		LastRefresh: time.Now().UTC(),
		Tokens: Tokens{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			IDToken:      tokens.IDToken,
		},
	}
	ApplyClaims(&file)
	if err := NewStore(opts.Home).Save(file); err != nil {
		return File{}, err
	}
	return file, nil
}

func authorizeURL(endpoint, redirectURI, state, challenge string) string {
	query := []struct {
		key   string
		value string
	}{
		{"response_type", "code"},
		{"client_id", ClientID},
		{"redirect_uri", redirectURI},
		{"scope", strings.ReplaceAll(OAuthScopes, " ", "+")},
		{"code_challenge", challenge},
		{"code_challenge_method", "S256"},
		{"id_token_add_organizations", "true"},
		{"codex_cli_simplified_flow", "true"},
		{"state", state},
		{"originator", "opencode"},
	}
	parts := make([]string, 0, len(query))
	for _, item := range query {
		if item.key == "scope" {
			parts = append(parts, item.key+"="+item.value)
			continue
		}
		parts = append(parts, item.key+"="+oauthQueryEscape(item.value))
	}
	return endpoint + "?" + strings.Join(parts, "&")
}

func oauthQueryEscape(value string) string {
	return url.QueryEscape(value)
}

func listenCallback(ports []int) (net.Listener, error) {
	if len(ports) == 0 {
		ports = []int{DefaultOAuthCallbackPort, FallbackOAuthCallbackPort}
	}
	var lastErr error
	for _, port := range ports {
		listener, err := listenCallbackPort(port)
		if err == nil {
			return listener, nil
		}
		lastErr = err
		if !isAddrInUse(err) {
			return nil, err
		}
		if port != 0 {
			sendCancelRequest(port)
			for i := 0; i < 10; i++ {
				time.Sleep(200 * time.Millisecond)
				listener, err = listenCallbackPort(port)
				if err == nil {
					return listener, nil
				}
				lastErr = err
				if !isAddrInUse(err) {
					return nil, err
				}
			}
		}
	}
	return nil, fmt.Errorf("start OAuth callback server: %w", lastErr)
}

func listenCallbackPort(port int) (net.Listener, error) {
	return net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
}

func isAddrInUse(err error) bool {
	if errors.Is(err, syscall.EADDRINUSE) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "address already in use")
}

func sendCancelRequest(port int) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/cancel", port), nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		_ = resp.Body.Close()
	}
}

func callbackHandler(state string, codeCh chan<- string, errCh chan<- error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cancel" {
			w.Header().Set("content-type", "text/plain")
			_, _ = w.Write([]byte("Claudodex login canceled.\n"))
			sendOAuthCallbackError(errCh, fmt.Errorf("OAuth login canceled by a newer login attempt"))
			return
		}
		if r.URL.Path != "/auth/callback" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("state"); got != state {
			http.Error(w, "invalid OAuth state", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			if oauthErr := r.URL.Query().Get("error"); oauthErr != "" {
				description := r.URL.Query().Get("error_description")
				if description != "" {
					oauthErr += ": " + description
				}
				sendOAuthCallbackError(errCh, fmt.Errorf("OAuth callback error: %s", oauthErr))
			}
			http.Error(w, "missing OAuth code", http.StatusBadRequest)
			return
		}
		select {
		case codeCh <- code:
		default:
		}
		w.Header().Set("content-type", "text/plain")
		_, _ = w.Write([]byte("Claudodex login complete. You can close this window.\n"))
	})
}

func sendOAuthCallbackError(errCh chan<- error, err error) {
	select {
	case errCh <- err:
	default:
	}
}

func exchangeCode(ctx context.Context, client *http.Client, endpoint, code, redirectURI, verifier string) (tokenResponse, error) {
	values := url.Values{}
	values.Set("grant_type", "authorization_code")
	values.Set("code", code)
	values.Set("redirect_uri", redirectURI)
	values.Set("client_id", ClientID)
	values.Set("code_verifier", verifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return tokenResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return tokenResponse{}, fmt.Errorf("token exchange failed: HTTP %d", resp.StatusCode)
	}

	var tokens tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return tokenResponse{}, err
	}
	if tokens.AccessToken == "" {
		return tokenResponse{}, fmt.Errorf("token response missing access_token")
	}
	return tokens, nil
}

func OpenBrowser(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		return fmt.Errorf("unsupported browser open platform %s", runtime.GOOS)
	}
	return cmd.Start()
}

func randomURLToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
