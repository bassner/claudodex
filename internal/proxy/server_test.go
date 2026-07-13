package proxy

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
)

func TestRoutes(t *testing.T) {
	server := New(Config{Version: "test", AuthPresent: true, Models: testModels()})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	base := "http://" + addr
	resp, err := http.Get(base + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d", resp.StatusCode)
	}
	var health struct {
		UpstreamAuth string `json:"upstream_auth"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatal(err)
	}
	if health.UpstreamAuth != "present" {
		t.Fatalf("upstream_auth = %q", health.UpstreamAuth)
	}

	resp, err = http.Get(base + "/api/v1/models?limit=1000")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("models status = %d", resp.StatusCode)
	}
	var body struct {
		Data []struct {
			ID             string `json:"id"`
			MaxInputTokens int64  `json:"max_input_tokens"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) == 0 {
		t.Fatal("models response had no models")
	}
	for _, model := range body.Data {
		if strings.Contains(model.ID, "[1m]") {
			t.Fatalf("long-context runtime suffix leaked into visible models response: %#v", body.Data)
		}
		switch model.ID {
		case "claude-opus-4-6", "claude-opus-4-7", "claude-opus-4-8", "gpt-5.6-sol":
			if model.MaxInputTokens != 111000 {
				t.Fatalf("%s max_input_tokens = %d, want 111000", model.ID, model.MaxInputTokens)
			}
		case "claude-sonnet-4-6", "gpt-5.6-terra":
			if model.MaxInputTokens != 222000 {
				t.Fatalf("%s max_input_tokens = %d, want 222000", model.ID, model.MaxInputTokens)
			}
		case "claude-haiku-4-5", "gpt-5.6-luna":
			if model.MaxInputTokens != 333000 {
				t.Fatalf("%s max_input_tokens = %d, want 333000", model.ID, model.MaxInputTokens)
			}
		}
	}
}

func TestUnixSocketRoutes(t *testing.T) {
	server := New(Config{Version: "test", AuthPresent: true, Models: testModels()})
	socketPath := filepath.Join(t.TempDir(), "api.sock")
	if _, err := server.StartUnix(socketPath); err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	client := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}}
	resp, err := client.Get("http://api.anthropic.com/v1/models")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("models status = %d", resp.StatusCode)
	}
}

func TestModelsRequireDynamicMetadata(t *testing.T) {
	server := New(Config{Version: "test", AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Get("http://" + addr + "/v1/models")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("models status = %d, want 503", resp.StatusCode)
	}
}

func TestModelsUseConfiguredTargets(t *testing.T) {
	server := New(Config{
		Version:     "test",
		AuthPresent: true,
		ModelConfig: modelconfig.Config{
			Opus:   "gpt-opus-next",
			Sonnet: "gpt-sonnet-next",
			Haiku:  "gpt-haiku-next",
		},
		Models: []codex.ModelInfo{
			{Slug: "gpt-opus-next", ContextWindow: 444000},
			{Slug: "gpt-sonnet-next", ContextWindow: 555000},
			{Slug: "gpt-haiku-next", ContextWindow: 666000},
		},
	})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Get("http://" + addr + "/v1/models")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("models status = %d", resp.StatusCode)
	}
	var body struct {
		Data []struct {
			ID             string `json:"id"`
			MaxInputTokens int64  `json:"max_input_tokens"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	foundDirect := false
	foundAlias := false
	for _, model := range body.Data {
		if strings.Contains(model.ID, "[1m]") {
			t.Fatalf("long-context runtime suffix leaked into custom models response: %#v", body.Data)
		}
		if model.ID == "gpt-sonnet-next" && model.MaxInputTokens == 555000 {
			foundDirect = true
		}
		if model.ID == "claude-sonnet-4-6" && model.MaxInputTokens == 555000 {
			foundAlias = true
		}
		if model.ID == "gpt-5.4" {
			t.Fatalf("default sonnet target leaked into custom models response: %#v", body.Data)
		}
	}
	if !foundDirect || !foundAlias {
		t.Fatalf("custom target models missing: %#v", body.Data)
	}
}

func TestWrongMethodReturns405(t *testing.T) {
	server := New(Config{Version: "test"})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/models", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

func TestClaudeLocalOAuthCompatibilityRoutes(t *testing.T) {
	server := New(Config{Version: "test"})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	base := "http://" + addr
	resp, err := http.Get(base + "/api/oauth/profile")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("profile status = %d, want 200", resp.StatusCode)
	}
	var profile map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		t.Fatal(err)
	}
	org := profile["organization"].(map[string]any)
	if org["organization_type"] != "claude_max" {
		t.Fatalf("profile organization = %#v", org)
	}

	resp, err = http.Get(base + "/api/claude_code/settings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("settings status = %d, want 204", resp.StatusCode)
	}

	resp, err = http.Get(base + "/api/claude_code/policy_limits")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("policy status = %d, want 200", resp.StatusCode)
	}
	var policy map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		t.Fatal(err)
	}
	if _, ok := policy["restrictions"].(map[string]any); !ok {
		t.Fatalf("policy body = %#v", policy)
	}
	restrictions := policy["restrictions"].(map[string]any)
	for _, key := range []string{"allow_remote_control", "allow_remote_sessions"} {
		restriction, ok := restrictions[key].(map[string]any)
		if !ok || restriction["allowed"] != true {
			t.Fatalf("%s restriction = %#v", key, restrictions[key])
		}
	}

	resp, err = http.Get(base + "/api/claude_code_penguin_mode")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fast mode status = %d, want 200", resp.StatusCode)
	}
	var fastMode map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&fastMode); err != nil {
		t.Fatal(err)
	}
	if fastMode["enabled"] != true {
		t.Fatalf("fast mode body = %#v", fastMode)
	}
}

func TestBatchesReturnsAnthropic501(t *testing.T) {
	server := New(Config{Version: "test"})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages/batches?beta=true", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", resp.StatusCode)
	}
	var body struct {
		Type  string `json:"type"`
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Type != "error" || body.Error.Type != "invalid_request_error" {
		t.Fatalf("unexpected error body: %#v", body)
	}
}

func TestCountTokensEstimatesRequestSize(t *testing.T) {
	server := New(Config{Version: "test"})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	body := `{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":"hello world hello world hello world"}]}`
	resp, err := http.Post("http://"+addr+"/v1/messages/count_tokens", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var got struct {
		InputTokens int `json:"input_tokens"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.InputTokens <= 1 {
		t.Fatalf("input_tokens = %d, want estimate > 1", got.InputTokens)
	}
}

func TestCountTokensAddsImagePadding(t *testing.T) {
	textOnly := estimateImagePadding([]byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"x"}]}]}`))
	withImage := estimateImagePadding([]byte(`{"messages":[{"role":"user","content":[{"type":"image","source":{"type":"url","url":"https://example.com/image.png"}}]}]}`))
	if textOnly != 0 {
		t.Fatalf("text image padding = %d, want 0", textOnly)
	}
	if withImage < 8500 {
		t.Fatalf("image padding = %d, want at least 8500", withImage)
	}
}

func testModels() []codex.ModelInfo {
	return []codex.ModelInfo{
		{Slug: "gpt-5.6-sol", ContextWindow: 111000},
		{Slug: "gpt-5.6-terra", ContextWindow: 222000},
		{Slug: "gpt-5.6-luna", ContextWindow: 333000},
	}
}
