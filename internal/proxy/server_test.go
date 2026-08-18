package proxy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
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
		switch model.ID {
		case "claude-opus-5", "claude-opus-4-6", "claude-opus-4-7", "claude-opus-4-8":
			if model.MaxInputTokens != 111000 {
				t.Fatalf("%s max_input_tokens = %d, want 111000", model.ID, model.MaxInputTokens)
			}
		case "claude-sonnet-4-6":
			if model.MaxInputTokens != 222000 {
				t.Fatalf("%s max_input_tokens = %d, want 222000", model.ID, model.MaxInputTokens)
			}
		case "claude-haiku-4-5":
			if model.MaxInputTokens != 333000 {
				t.Fatalf("%s max_input_tokens = %d, want 333000", model.ID, model.MaxInputTokens)
			}
		case "gpt-5.6-sol[1m]":
			if model.MaxInputTokens != 811000 {
				t.Fatalf("%s max_input_tokens = %d, want 811000", model.ID, model.MaxInputTokens)
			}
		case "gpt-5.6-terra[1m]":
			if model.MaxInputTokens != 822000 {
				t.Fatalf("%s max_input_tokens = %d, want 822000", model.ID, model.MaxInputTokens)
			}
		case "gpt-5.6-luna[1m]":
			if model.MaxInputTokens != 833000 {
				t.Fatalf("%s max_input_tokens = %d, want 833000", model.ID, model.MaxInputTokens)
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
			{Slug: "gpt-opus-next", ContextWindow: 444000, MaxContextWindow: 844000},
			{Slug: "gpt-sonnet-next", ContextWindow: 555000, MaxContextWindow: 855000},
			{Slug: "gpt-haiku-next", ContextWindow: 666000, MaxContextWindow: 866000},
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
		if model.ID == "gpt-sonnet-next[1m]" && model.MaxInputTokens == 855000 {
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

func TestTokenCountFallbackCountsImagePatchesWithoutBase64Text(t *testing.T) {
	prefix := `{"messages":[{"role":"user","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"`
	suffix := `"}}]}]}`

	tests := []struct {
		name       string
		width      int
		height     int
		wantTokens int
	}{
		{name: "common screenshot", width: 1600, height: 1200, wantTokens: 1900},
		{name: "wide screenshot", width: 1682, height: 594, wantTokens: 1007},
		{name: "small image", width: 270, height: 270, wantTokens: 81},
		{name: "Claude Code dimension ceiling", width: 2000, height: 2000, wantTokens: 3969},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded := testPNGBase64(t, test.width, test.height)
			body := []byte(prefix + encoded + suffix)
			normalized, imageTokens := normalizeTokenEstimateInput(body)
			if imageTokens != test.wantTokens {
				t.Fatalf("image tokens = %d, want %d", imageTokens, test.wantTokens)
			}
			if bytes.Contains(normalized, []byte(encoded)) {
				t.Fatal("normalized token input still contains base64 payload")
			}
			wantTotal := (len(normalized)+2)/3 + test.wantTokens
			if got := estimateTokenCountFromBytes(body, false); got != wantTotal {
				t.Fatalf("total estimate = %d, want %d", got, wantTotal)
			}
		})
	}
}

func TestTokenCountFallbackUsesBoundedEstimateForInvalidImage(t *testing.T) {
	prefix := `{"messages":[{"role":"user","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"`
	suffix := `"}}]}]}`
	small := estimateTokenCountFromBytes([]byte(prefix+strings.Repeat("A", 40)+suffix), false)
	large := estimateTokenCountFromBytes([]byte(prefix+strings.Repeat("A", 400_000)+suffix), false)
	if small != large {
		t.Fatalf("small invalid image estimate = %d, large invalid image estimate = %d", small, large)
	}
	if large > 2500 {
		t.Fatalf("invalid image fallback estimate = %d, want bounded estimate", large)
	}
}

func TestTokenCountFallbackIgnoresValidBase64PayloadLength(t *testing.T) {
	encoded := testPNGBase64(t, 1600, 1200)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	encodedWithTrailingData := base64.StdEncoding.EncodeToString(append(decoded, make([]byte, 300_000)...))
	prefix := `{"messages":[{"role":"user","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"`
	suffix := `"}}]}]}`
	small := estimateTokenCountFromBytes([]byte(prefix+encoded+suffix), false)
	large := estimateTokenCountFromBytes([]byte(prefix+encodedWithTrailingData+suffix), false)
	if small != large {
		t.Fatalf("small valid image estimate = %d, large valid image estimate = %d", small, large)
	}
}

func TestImageTokenEstimateReadsWebPDimensions(t *testing.T) {
	header := make([]byte, 30)
	copy(header[0:4], "RIFF")
	copy(header[8:12], "WEBP")
	copy(header[12:16], "VP8X")
	width, height := 1682, 594
	storedWidth, storedHeight := width-1, height-1
	header[24], header[25], header[26] = byte(storedWidth), byte(storedWidth>>8), byte(storedWidth>>16)
	header[27], header[28], header[29] = byte(storedHeight), byte(storedHeight>>8), byte(storedHeight>>16)
	encoded := base64.StdEncoding.EncodeToString(header)
	if got := encodedImageTokenEstimate(encoded, ""); got != 1007 {
		t.Fatalf("WebP token estimate = %d, want 1007", got)
	}
}

func TestLowDetailImageUsesFixed512PatchCount(t *testing.T) {
	encoded := testPNGBase64(t, 1600, 1200)
	if got := encodedImageTokenEstimate(encoded, "low"); got != 256 {
		t.Fatalf("low-detail token estimate = %d, want 256", got)
	}
}

func testPNGBase64(t *testing.T, width, height int) string {
	t.Helper()
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewGray(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(data.Bytes())
}

func testModels() []codex.ModelInfo {
	return []codex.ModelInfo{
		{Slug: "gpt-5.6-sol", ContextWindow: 111000, MaxContextWindow: 811000},
		{Slug: "gpt-5.6-terra", ContextWindow: 222000, MaxContextWindow: 822000},
		{Slug: "gpt-5.6-luna", ContextWindow: 333000, MaxContextWindow: 833000},
	}
}
