package launcher

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const claudeBridgeAuthReadTimeout = 2 * time.Second

func WithRealClaudeBridgeAuth(envList []string) []string {
	env := envMap(envList)
	if strings.TrimSpace(env["CLAUDE_BRIDGE_OAUTH_TOKEN"]) != "" {
		return envList
	}
	if strings.TrimSpace(env["CLAUDODEX_DISABLE_REAL_CLAUDE_BRIDGE_AUTH"]) == "1" {
		return envList
	}
	token, ok := realClaudeBridgeAccessToken()
	if !ok {
		return envList
	}
	env["CLAUDE_BRIDGE_OAUTH_TOKEN"] = token
	return flattenEnv(env)
}

func realClaudeBridgeAccessToken() (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	if runtime.GOOS == "darwin" {
		if token, ok := realClaudeBridgeAccessTokenFromKeychain(); ok {
			return token, true
		}
	}
	return realClaudeBridgeAccessTokenFromFile(filepath.Join(home, ".claude", claudeCredentialsFileName))
}

func realClaudeBridgeAccessTokenFromFile(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false
		}
		return "", false
	}
	return realClaudeBridgeAccessTokenFromJSON(data)
}

func realClaudeBridgeAccessTokenFromKeychain() (string, bool) {
	username := os.Getenv("USER")
	if username == "" {
		if current, err := user.Current(); err == nil {
			username = current.Username
		}
	}
	if username == "" {
		return "", false
	}
	ctx, cancel := context.WithTimeout(context.Background(), claudeBridgeAuthReadTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/security", "find-generic-password", "-a", username, "-w", "-s", "Claude Code-credentials")
	out, err := cmd.Output()
	if err != nil || ctx.Err() != nil {
		return "", false
	}
	return realClaudeBridgeAccessTokenFromJSON(out)
}

func realClaudeBridgeAccessTokenFromJSON(data []byte) (string, bool) {
	var credentials claudeCredentials
	if err := json.Unmarshal(data, &credentials); err != nil {
		return "", false
	}
	token := strings.TrimSpace(credentials.ClaudeAIOAuth.AccessToken)
	if token == "" || token == localOAuthAccessToken {
		return "", false
	}
	return token, true
}
