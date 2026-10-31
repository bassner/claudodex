package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

const (
	nativeOpenAIBase  = "https://chatgpt.com/backend-api/codex"
	nativeChatGPTBase = "https://chatgpt.com/backend-api/"
	nativeTimeout     = 15 * time.Second
	nativeMaxOutput   = 4 << 20
	nativeMaxMessage  = 1 << 20
)

type nativeMessage struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

type nativeRPC struct {
	ctx      context.Context
	cancel   context.CancelFunc
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	messages chan nativeMessage
	errors   chan error
	nextID   int
	total    atomic.Int64
}

func readNativeCodex(ctx context.Context, home string, refresh bool) (File, error) {
	server, err := startNativeCodex(ctx, home)
	if err != nil {
		return File{}, err
	}
	defer server.close()
	if err := server.request("initialize", map[string]any{"clientInfo": map[string]string{"name": "claudodex_auth", "version": "1.0"}, "capabilities": map[string]bool{"experimentalApi": false}}, nil); err != nil {
		return File{}, err
	}
	if err := server.send(map[string]any{"method": "initialized", "params": map[string]any{}}); err != nil {
		return File{}, err
	}
	var config struct {
		Config map[string]json.RawMessage `json:"config"`
	}
	if err := server.request("config/read", map[string]any{"includeLayers": false, "cwd": home}, &config); err != nil {
		return File{}, err
	}
	storage, err := validateNativeConfig(config.Config)
	if err != nil {
		return File{}, err
	}
	var account struct {
		Account *struct {
			Type     string `json:"type"`
			Email    string `json:"email"`
			PlanType string `json:"planType"`
		} `json:"account"`
	}
	if err := server.request("account/read", map[string]bool{"refreshToken": refresh}, &account); err != nil {
		return File{}, err
	}
	if account.Account == nil || account.Account.Type != "chatgpt" {
		return File{}, fmt.Errorf("Codex requires a native ChatGPT login; API-key and external-token logins cannot be reused")
	}
	var status struct {
		AuthMethod string `json:"authMethod"`
		AuthToken  string `json:"authToken"`
	}
	if err := server.request("getAuthStatus", map[string]bool{"includeToken": true, "refreshToken": false}, &status); err != nil {
		return File{}, err
	}
	if status.AuthMethod != "chatgpt" || status.AuthToken == "" {
		return File{}, fmt.Errorf("Codex cannot export a native subscription access token")
	}
	claims, err := validatedNativeClaims(status.AuthToken)
	if err != nil {
		return File{}, err
	}
	if account.Account.Email != "" && claims.Email != "" && !strings.EqualFold(account.Account.Email, claims.Email) {
		return File{}, fmt.Errorf("Codex account and access-token identity disagree")
	}
	fedramp := claims.FedRAMP
	if !claims.FedRAMPKnown {
		if storage != "file" {
			return File{}, fmt.Errorf("Codex cannot export residency metadata for this credential store; use direct Claudodex browser login")
		}
		fedramp, err = readNativeResidency(ctx, home, claims)
		if err != nil {
			return File{}, err
		}
	}
	return File{Source: SourceCodex, AllowedOrigin: CodexAllowedOrigin, AuthMode: "chatgpt", Issuer: Issuer, ClientID: ClientID,
		Tokens: Tokens{AccessToken: status.AuthToken, AccountID: claims.AccountID, UserID: claims.UserID,
			Email: claims.Email, PlanType: account.Account.PlanType, AccessTokenExpiresAt: claims.Exp, ChatGPTAccountIsFedRAMP: fedramp}}, nil
}

func validatedNativeClaims(token string) (Claims, error) {
	if len(token) > nativeMaxMessage {
		return Claims{}, fmt.Errorf("Codex access token exceeds the supported size")
	}
	claims, err := ParseClaims(token)
	if err != nil || claims.Issuer != Issuer || claims.UserID == "" || claims.AccountID == "" || claims.Exp <= 0 || strings.ContainsAny(claims.UserID+claims.AccountID, "\r\n\x00") {
		return Claims{}, fmt.Errorf("Codex returned unsupported access-token identity claims")
	}
	return claims, nil
}

// Native Codex's ID-token parser defaults the optional FedRAMP bool to false.
// Read only this metadata when the effective authority is file storage, and bind
// it to both IDs from the native-exported access token. Never decode refresh data.
func readNativeResidency(ctx context.Context, home string, access Claims) (bool, error) {
	for attempt := 0; attempt < 3; attempt++ {
		fedramp, err := readNativeResidencyOnce(filepath.Join(home, "auth.json"), access)
		if err == nil {
			return fedramp, nil
		}
		if attempt == 2 {
			return false, err
		}
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false, ctx.Err()
		case <-timer.C:
		}
	}
	panic("unreachable")
}

func readNativeResidencyOnce(path string, access Claims) (bool, error) {
	file, err := openPrivateRegular(path, nativeMaxMessage)
	if err != nil {
		return false, fmt.Errorf("Codex residency metadata is unavailable")
	}
	defer file.Close()
	// This allowlist deliberately has no refresh-token or access-token field.
	var metadata struct {
		AuthMode string `json:"auth_mode"`
		Tokens   struct {
			IDToken   string `json:"id_token"`
			AccountID string `json:"account_id"`
		} `json:"tokens"`
	}
	decoder := json.NewDecoder(io.LimitReader(file, nativeMaxMessage))
	if err := decoder.Decode(&metadata); err != nil {
		return false, fmt.Errorf("Codex residency metadata is incomplete")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return false, fmt.Errorf("Codex residency metadata is incomplete")
	}
	if metadata.AuthMode != "chatgpt" || metadata.Tokens.IDToken == "" || (metadata.Tokens.AccountID != "" && metadata.Tokens.AccountID != access.AccountID) {
		return false, fmt.Errorf("Codex residency metadata does not match the native login")
	}
	claims, err := ParseClaims(metadata.Tokens.IDToken)
	if err != nil || claims.Issuer != Issuer || claims.UserID != access.UserID || claims.AccountID != access.AccountID {
		return false, fmt.Errorf("Codex residency metadata identity does not match the access token")
	}
	return claims.FedRAMP, nil
}

func validateNativeConfig(config map[string]json.RawMessage) (string, error) {
	if config == nil {
		return "", fmt.Errorf("Codex cannot expose its effective configuration")
	}
	for key, expected := range map[string]string{"forced_login_method": "chatgpt", "model_provider": "openai", "openai_base_url": nativeOpenAIBase, "chatgpt_base_url": nativeChatGPTBase} {
		var actual string
		if err := json.Unmarshal(config[key], &actual); err != nil || actual != expected {
			return "", fmt.Errorf("Codex effective %s is unsafe or unsupported", key)
		}
	}
	var providers map[string]map[string]json.RawMessage
	if raw := config["model_providers"]; len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &providers); err != nil {
			return "", fmt.Errorf("Codex provider configuration is unsupported")
		}
		for _, raw := range providers["openai"] {
			if s := string(raw); s != "null" && s != `""` && s != "{}" && s != "[]" {
				return "", fmt.Errorf("Codex OpenAI provider overrides cannot be used with login reuse")
			}
		}
	}
	var storage string
	if err := json.Unmarshal(config["cli_auth_credentials_store"], &storage); err != nil {
		return "", fmt.Errorf("Codex cannot expose its effective credential store")
	}
	switch storage {
	case "file", "keyring", "auto":
		return storage, nil
	default:
		return "", fmt.Errorf("Codex credential storage mode is unsupported")
	}
}

func startNativeCodex(parent context.Context, home string) (*nativeRPC, error) {
	binary, err := exec.LookPath("codex")
	if err != nil {
		return nil, fmt.Errorf("Codex CLI is not installed")
	}
	binary, err = filepath.EvalSymlinks(binary)
	if err != nil {
		return nil, fmt.Errorf("Codex CLI path cannot be resolved")
	}
	ctx, cancel := context.WithTimeout(parent, nativeTimeout)
	cmd := exec.CommandContext(ctx, binary, "app-server", "--stdio", "-c", `forced_login_method="chatgpt"`, "-c", `model_provider="openai"`, "-c", `openai_base_url="`+nativeOpenAIBase+`"`, "-c", `chatgpt_base_url="`+nativeChatGPTBase+`"`)
	cmd.Env = nativeEnvironment(home)
	cmd.Dir = home
	configureNativeProcess(cmd)
	cmd.WaitDelay = 200 * time.Millisecond
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("Codex native input is unavailable")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		stdin.Close()
		return nil, fmt.Errorf("Codex native output is unavailable")
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("Codex native error output is unavailable")
	}
	server := &nativeRPC{ctx: ctx, cancel: cancel, cmd: cmd, stdin: stdin, messages: make(chan nativeMessage, 8), errors: make(chan error, 1)}
	if err := cmd.Start(); err != nil {
		cancel()
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, fmt.Errorf("Codex native authentication could not start")
	}
	go server.readMessages(stdout)
	go func() {
		buffer := make([]byte, 8192)
		for {
			n, err := stderr.Read(buffer)
			if n > 0 && server.total.Add(int64(n)) > nativeMaxOutput {
				server.fail(fmt.Errorf("Codex native authentication exceeded its output limit"))
				return
			}
			if err != nil {
				return
			}
		}
	}()
	return server, nil
}

func nativeEnvironment(home string) []string {
	blocked := map[string]bool{"CODEX_HOME": true, "OPENAI_API_KEY": true, "CODEX_API_KEY": true, "ANTHROPIC_API_KEY": true,
		"OPENAI_BASE_URL": true, "CODEX_APP_SERVER_CHATGPT_BASE_URL": true, "CODEX_REFRESH_TOKEN_URL_OVERRIDE": true,
		"CODEX_REVOKE_TOKEN_URL_OVERRIDE": true, "CODEX_APP_SERVER_LOGIN_CLIENT_ID": true}
	env := make([]string, 0, len(os.Environ())+1)
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if !blocked[key] {
			env = append(env, value)
		}
	}
	return append(env, "CODEX_HOME="+home)
}

func (s *nativeRPC) fail(err error) {
	select {
	case s.errors <- err:
	default:
	}
	s.cancel()
}

func (s *nativeRPC) readMessages(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 8192), nativeMaxMessage)
	for scanner.Scan() {
		line := scanner.Bytes()
		if s.total.Add(int64(len(line)+1)) > nativeMaxOutput {
			s.fail(fmt.Errorf("Codex native authentication exceeded its output limit"))
			return
		}
		var message nativeMessage
		if json.Unmarshal(line, &message) != nil {
			s.fail(fmt.Errorf("Codex native authentication returned invalid JSON"))
			return
		}
		select {
		case s.messages <- message:
		case <-s.ctx.Done():
			return
		}
	}
	if scanner.Err() != nil {
		s.fail(fmt.Errorf("Codex native authentication returned an oversized or unreadable message"))
		return
	}
	s.fail(fmt.Errorf("Codex native authentication ended before completing the request"))
}

func (s *nativeRPC) send(value any) error {
	if err := json.NewEncoder(s.stdin).Encode(value); err != nil {
		return fmt.Errorf("Codex native authentication input closed")
	}
	return nil
}

func (s *nativeRPC) request(method string, params any, result any) error {
	s.nextID++
	id := s.nextID
	if err := s.send(map[string]any{"id": id, "method": method, "params": params}); err != nil {
		return err
	}
	for {
		select {
		case err := <-s.errors:
			return err
		case <-s.ctx.Done():
			if errors.Is(s.ctx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("Codex native authentication timed out: %w", s.ctx.Err())
			}
			return s.ctx.Err()
		case message := <-s.messages:
			if message.Method != "" {
				if len(message.ID) > 0 {
					return fmt.Errorf("Codex native authentication requested unsupported interaction")
				}
				continue
			}
			var responseID int
			if json.Unmarshal(message.ID, &responseID) != nil || responseID != id {
				return fmt.Errorf("Codex native authentication returned an unexpected response")
			}
			if len(message.Error) > 0 && string(message.Error) != "null" {
				return fmt.Errorf("Codex CLI does not support the required %s authentication method", method)
			}
			if len(message.Result) == 0 {
				return fmt.Errorf("Codex native authentication omitted the %s result", method)
			}
			if result != nil && json.Unmarshal(message.Result, result) != nil {
				return fmt.Errorf("Codex native authentication returned unsupported %s metadata", method)
			}
			return nil
		}
	}
}

func (s *nativeRPC) close() {
	_ = s.stdin.Close()
	s.cancel()
	_ = s.cmd.Wait()
}
