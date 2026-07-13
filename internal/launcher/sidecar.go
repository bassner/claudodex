package launcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/bassner/claudodex/internal/auth"
	"github.com/bassner/claudodex/internal/modelconfig"
)

const (
	claudeSidecarDirName       = "claude-config"
	claudodexShimDirName       = "claudodex-bin"
	claudeCredentialsFileName  = ".credentials.json"
	claudeLocalOAuthConfigName = ".claude-local-oauth.json"
	claudeGlobalConfigName     = ".claude.json"
	claudePolicyLimitsFileName = "policy-limits.json"
)

type claudeCredentials struct {
	ClaudeAIOAuth claudeOAuthToken `json:"claudeAiOauth"`
}

type claudeOAuthToken struct {
	AccessToken      string   `json:"accessToken"`
	RefreshToken     string   `json:"refreshToken"`
	ExpiresAt        int64    `json:"expiresAt"`
	Scopes           []string `json:"scopes"`
	SubscriptionType string   `json:"subscriptionType"`
	RateLimitTier    *string  `json:"rateLimitTier"`
}

func PrepareClaudeConfigSidecar(home string, modelCfg modelconfig.Config) (string, error) {
	dataDir, err := auth.DataDir(home)
	if err != nil {
		return "", err
	}
	sidecarDir := filepath.Join(dataDir, claudeSidecarDirName)
	if err := os.MkdirAll(sidecarDir, 0o700); err != nil {
		return "", err
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if err := syncClaudeConfigSidecar(sidecarDir, userHome, modelCfg); err != nil {
		return "", err
	}
	if err := writeClaudeLocalCredentials(sidecarDir); err != nil {
		return "", err
	}
	if err := writeClaudePolicyLimits(sidecarDir); err != nil {
		return "", err
	}
	if err := writeClaudeShims(sidecarDir); err != nil {
		return "", err
	}
	return sidecarDir, nil
}

func syncClaudeConfigSidecar(sidecarDir, userHome string, modelCfg modelconfig.Config) error {
	return withClaudeSidecarSetupLock(sidecarDir, claudeConfigLockWait, func() error {
		if err := linkClaudeConfigEntries(filepath.Join(userHome, ".claude"), sidecarDir); err != nil {
			return err
		}
		if err := normalizeSharedClaudeSettings(userHome, claudeConfigLockWait, modelCfg); err != nil {
			return err
		}
		return initializeClaudeConfigSidecar(sidecarDir, userHome)
	})
}

func linkClaudeConfigEntries(realClaudeDir, sidecarDir string) error {
	entries, err := os.ReadDir(realClaudeDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == claudeCredentialsFileName || name == claudeLocalOAuthConfigName || name == "cache" {
			continue
		}
		src := filepath.Join(realClaudeDir, name)
		dst := filepath.Join(sidecarDir, name)
		if err := replaceWithSymlink(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func replaceWithSymlink(src, dst string) error {
	if current, err := os.Readlink(dst); err == nil && current == src {
		return nil
	}
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	if err := os.Symlink(src, dst); err != nil {
		if errors.Is(err, os.ErrExist) {
			if current, readErr := os.Readlink(dst); readErr == nil && current == src {
				return nil
			}
		}
		return err
	}
	return nil
}

func readJSONMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if out == nil {
		return map[string]any{}, nil
	}
	return out, nil
}

func writeClaudeLocalCredentials(sidecarDir string) error {
	credentials := claudeCredentials{
		ClaudeAIOAuth: claudeOAuthToken{
			AccessToken:  "claudodex-local-oauth",
			RefreshToken: "claudodex-local-refresh",
			ExpiresAt:    time.Now().Add(365 * 24 * time.Hour).UnixMilli(),
			Scopes: []string{
				"user:profile",
				"user:inference",
				"user:sessions:claude_code",
				"user:mcp_servers",
				"user:file_upload",
			},
			SubscriptionType: "max",
			RateLimitTier:    nil,
		},
	}
	if err := writeJSONFile(filepath.Join(sidecarDir, claudeCredentialsFileName), credentials, 0o600); err != nil {
		return err
	}
	return nil
}

func writeClaudePolicyLimits(sidecarDir string) error {
	return writeJSONFile(filepath.Join(sidecarDir, claudePolicyLimitsFileName), map[string]any{
		"restrictions": map[string]any{
			"allow_remote_control":  map[string]bool{"allowed": true},
			"allow_remote_sessions": map[string]bool{"allowed": true},
		},
		"compliance_taints": []string{},
		"monitoring_notice": nil,
		"defaults":          map[string]any{},
	}, 0o600)
}

func writeClaudeShims(sidecarDir string) error {
	shimDir := filepath.Join(sidecarDir, claudodexShimDirName)
	if err := os.MkdirAll(shimDir, 0o700); err != nil {
		return err
	}
	for _, name := range []string{"claudodex-shell", "sh", "bash", "zsh", "fish"} {
		if err := os.WriteFile(filepath.Join(shimDir, name), []byte(claudodexShellShimScript), 0o700); err != nil {
			return err
		}
	}
	if runtime.GOOS == "darwin" {
		return os.WriteFile(filepath.Join(shimDir, "security"), []byte(claudeSecurityShimScript), 0o700)
	}
	return nil
}

const claudodexShellShimScript = `#!/bin/sh

restore_var() {
  name="$1"
  original="CLAUDODEX_ORIGINAL_$name"
  if value=$(printenv "$original" 2>/dev/null); then
    export "$name=$value"
  else
    unset "$name"
  fi
  unset "$original"
}

	real_shell=${CLAUDODEX_REAL_SHELL:-/bin/sh}
	unset CLAUDODEX_REAL_SHELL

restore_var SHELL
restore_var HTTP_PROXY
restore_var http_proxy
restore_var HTTPS_PROXY
restore_var https_proxy
restore_var NO_PROXY
restore_var no_proxy
restore_var ALL_PROXY
restore_var all_proxy
restore_var NODE_EXTRA_CA_CERTS

	case "$real_shell" in
	  */*) ;;
	  *) real_shell=$(command -v "$real_shell" || printf '/bin/sh') ;;
	esac
	
	exec "$real_shell" "$@"
	`

const claudeSecurityShimScript = `#!/bin/sh
set -efu

real_security=/usr/bin/security

is_claude_code_service() {
  case "$1" in
    "Claude Code"*) return 0 ;;
    *) return 1 ;;
  esac
}

is_generic_password_op() {
  case " $* " in
    *" find-generic-password "*|*" add-generic-password "*|*" delete-generic-password "*) return 0 ;;
    *) return 1 ;;
  esac
}

service_from_args() {
  prev=
  for arg in "$@"; do
    if [ "$prev" = "-s" ]; then
      printf '%s' "$arg"
      return 0
    fi
    prev="$arg"
  done
  return 1
}

case "${1-}" in
  -i)
    input=$(cat)
    case "$input" in
      *'generic-password'*'Claude Code'*|*'add-generic-password'*)
        exit 44
        ;;
    esac
    printf '%s\n' "$input" | "$real_security" -i
    exit $?
    ;;
esac

service=$(service_from_args "$@" || true)
if is_generic_password_op "$@" && is_claude_code_service "$service"; then
  exit 44
fi
if [ "${1-}" = "add-generic-password" ]; then
  exit 44
fi

exec "$real_security" "$@"
`

func writeJSONFile(path string, value any, mode os.FileMode) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), fmt.Sprintf(".%s-*.tmp", filepath.Base(path)))
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
