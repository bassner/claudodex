package launcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRealClaudeBridgeAccessTokenFromJSON(t *testing.T) {
	token, ok := realClaudeBridgeAccessTokenFromJSON([]byte(`{"claudeAiOauth":{"accessToken":"real-claude-token","scopes":["user:profile","user:inference"],"subscriptionType":"max"}}`))
	if !ok || token != "real-claude-token" {
		t.Fatalf("token = %q, ok = %v", token, ok)
	}
}

func TestRealClaudeBridgeAccessTokenFromJSONRejectsLocalToken(t *testing.T) {
	token, ok := realClaudeBridgeAccessTokenFromJSON([]byte(`{"claudeAiOauth":{"accessToken":"` + localOAuthAccessToken + `"}}`))
	if ok || token != "" {
		t.Fatalf("token = %q, ok = %v, want rejected", token, ok)
	}
}

func TestRealClaudeBridgeOrganizationUUIDFromJSON(t *testing.T) {
	organizationUUID, ok := realClaudeBridgeOrganizationUUIDFromJSON([]byte(`{"oauthAccount":{"organizationUuid":"real-org-uuid"}}`))
	if !ok || organizationUUID != "real-org-uuid" {
		t.Fatalf("organization UUID = %q, ok = %v", organizationUUID, ok)
	}
}

func TestRealClaudeBridgeOrganizationUUIDFromJSONRejectsMissingOrganization(t *testing.T) {
	organizationUUID, ok := realClaudeBridgeOrganizationUUIDFromJSON([]byte(`{"oauthAccount":{"emailAddress":"real@example.com"}}`))
	if ok || organizationUUID != "" {
		t.Fatalf("organization UUID = %q, ok = %v, want rejected", organizationUUID, ok)
	}
}

func TestWithRealClaudeBridgeAuthPreservesExplicitIdentityWhenDisabled(t *testing.T) {
	env := WithRealClaudeBridgeAuth([]string{
		"CLAUDE_BRIDGE_OAUTH_TOKEN=explicit",
		"CLAUDE_CODE_ORGANIZATION_UUID=explicit-org",
		"CLAUDODEX_DISABLE_REAL_CLAUDE_BRIDGE_AUTH=1",
	})
	joined := strings.Join(env, "\n")
	if !strings.Contains(joined, "CLAUDE_BRIDGE_OAUTH_TOKEN=explicit") {
		t.Fatalf("explicit bridge token was not preserved:\n%s", joined)
	}
	if !strings.Contains(joined, "CLAUDE_CODE_ORGANIZATION_UUID=explicit-org") {
		t.Fatalf("explicit organization UUID was not preserved:\n%s", joined)
	}
}

func TestWithRealClaudeBridgeAuthLoadsIdentityFromRealClaudeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("USER", "claudodex-bridge-auth-test")
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(home, ".claude", claudeCredentialsFileName),
		[]byte(`{"claudeAiOauth":{"accessToken":"real-claude-token"}}`),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(home, claudeGlobalConfigName),
		[]byte(`{"oauthAccount":{"organizationUuid":"real-org-uuid"}}`),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	env := envMap(WithRealClaudeBridgeAuth([]string{"PATH=" + os.Getenv("PATH")}))
	if token := env["CLAUDE_BRIDGE_OAUTH_TOKEN"]; token != "real-claude-token" {
		t.Fatalf("bridge token = %q", token)
	}
	if organizationUUID := env["CLAUDE_CODE_ORGANIZATION_UUID"]; organizationUUID != "real-org-uuid" {
		t.Fatalf("organization UUID = %q", organizationUUID)
	}
}

func TestWithRealClaudeBridgeAuthPreservesExplicitIdentityWhenEnabled(t *testing.T) {
	env := envMap(WithRealClaudeBridgeAuth([]string{
		"CLAUDE_BRIDGE_OAUTH_TOKEN=explicit",
		"CLAUDE_CODE_ORGANIZATION_UUID=explicit-org",
	}))
	if token := env["CLAUDE_BRIDGE_OAUTH_TOKEN"]; token != "explicit" {
		t.Fatalf("bridge token = %q", token)
	}
	if organizationUUID := env["CLAUDE_CODE_ORGANIZATION_UUID"]; organizationUUID != "explicit-org" {
		t.Fatalf("organization UUID = %q", organizationUUID)
	}
}

func TestWithRealClaudeBridgeAuthWithoutClaudeLoginLeavesIdentityUnset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("USER", "claudodex-no-login-test")

	env := envMap(WithRealClaudeBridgeAuth([]string{"PATH=" + os.Getenv("PATH")}))
	if token := strings.TrimSpace(env["CLAUDE_BRIDGE_OAUTH_TOKEN"]); token != "" {
		t.Fatalf("bridge token = %q, want unset", token)
	}
	if organizationUUID := strings.TrimSpace(env["CLAUDE_CODE_ORGANIZATION_UUID"]); organizationUUID != "" {
		t.Fatalf("organization UUID = %q, want unset", organizationUUID)
	}
}
