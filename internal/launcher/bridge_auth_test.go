package launcher

import (
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

func TestWithRealClaudeBridgeAuthPreservesExplicitToken(t *testing.T) {
	env := WithRealClaudeBridgeAuth([]string{
		"CLAUDE_BRIDGE_OAUTH_TOKEN=explicit",
		"CLAUDODEX_DISABLE_REAL_CLAUDE_BRIDGE_AUTH=1",
	})
	joined := strings.Join(env, "\n")
	if !strings.Contains(joined, "CLAUDE_BRIDGE_OAUTH_TOKEN=explicit") {
		t.Fatalf("explicit bridge token was not preserved:\n%s", joined)
	}
}
