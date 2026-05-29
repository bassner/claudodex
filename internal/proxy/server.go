package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
)

type Config struct {
	Version       string
	Interactive   bool
	AuthPresent   bool
	Home          string
	CodexBaseURL  string
	TokenEndpoint string
	HTTPClient    *http.Client
	Models        []codex.ModelInfo
	ModelConfig   modelconfig.Config
}

type Server struct {
	cfg      Config
	server   *http.Server
	listener net.Listener
	once     sync.Once
	traceMu  sync.Mutex
	chainsMu sync.Mutex
	chains   map[string]responseChain
	wsMu     sync.Mutex
	ws       map[string]*codex.WebSocketConversation
}

func New(cfg Config) *Server {
	if cfg.Version == "" {
		cfg.Version = "dev"
	}
	if cfg.CodexBaseURL == "" {
		cfg.CodexBaseURL = os.Getenv("CLAUDODEX_CODEX_BASE_URL")
	}
	if cfg.CodexBaseURL == "" {
		cfg.CodexBaseURL = codex.DefaultBaseURL
	}
	cfg.ModelConfig = cfg.ModelConfig.Normalize()
	return &Server{cfg: cfg, chains: make(map[string]responseChain), ws: make(map[string]*codex.WebSocketConversation)}
}

func (s *Server) Start(host string, port int) (string, error) {
	if host == "" {
		host = "127.0.0.1"
	}
	if host != "127.0.0.1" && host != "localhost" {
		return "", fmt.Errorf("refusing non-loopback host %q without explicit future support", host)
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return "", err
	}
	s.listener = listener
	mux := http.NewServeMux()
	s.routes(mux)
	s.server = &http.Server{Handler: mux}
	go func() {
		_ = s.server.Serve(listener)
	}()
	return listener.Addr().String(), nil
}

func (s *Server) Port() int {
	if s.listener == nil {
		return 0
	}
	addr, ok := s.listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0
	}
	return addr.Port
}

func (s *Server) Close() error {
	var err error
	s.once.Do(func() {
		if s.server != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			err = s.server.Shutdown(ctx)
		}
		s.closeWebSockets()
	})
	return err
}

func (s *Server) websocketConversation(chainKey string) *codex.WebSocketConversation {
	chainKey = strings.TrimSpace(chainKey)
	if chainKey == "" {
		return nil
	}
	s.wsMu.Lock()
	defer s.wsMu.Unlock()
	conversation := s.ws[chainKey]
	if conversation == nil {
		conversation = &codex.WebSocketConversation{}
		s.ws[chainKey] = conversation
	}
	return conversation
}

func (s *Server) hasWebSocket(chainKey string) bool {
	chainKey = strings.TrimSpace(chainKey)
	if chainKey == "" {
		return false
	}
	s.wsMu.Lock()
	defer s.wsMu.Unlock()
	return s.ws[chainKey] != nil
}

func (s *Server) closeWebSocket(chainKey string) {
	chainKey = strings.TrimSpace(chainKey)
	if chainKey == "" {
		return
	}
	s.wsMu.Lock()
	conversation := s.ws[chainKey]
	delete(s.ws, chainKey)
	s.wsMu.Unlock()
	if conversation != nil {
		_ = conversation.Close()
	}
}

func (s *Server) closeWebSockets() {
	s.wsMu.Lock()
	conversations := make([]*codex.WebSocketConversation, 0, len(s.ws))
	for key, conversation := range s.ws {
		conversations = append(conversations, conversation)
		delete(s.ws, key)
	}
	s.wsMu.Unlock()
	for _, conversation := range conversations {
		if conversation != nil {
			_ = conversation.Close()
		}
	}
}

func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := normalizePath(r.URL.Path)
		s.logRequest(r.Method, path, r.URL.RawQuery)
		switch {
		case r.Method == http.MethodGet && path == "/healthz":
			writeJSON(w, http.StatusOK, map[string]any{
				"ok":            true,
				"upstream_auth": authStatus(s.cfg.AuthPresent),
				"version":       s.cfg.Version,
			})
		case r.Method == http.MethodGet && path == "/v1":
			writeJSON(w, http.StatusOK, map[string]any{
				"id":      "claudodex",
				"message": "Anthropic-compatible Codex proxy",
			})
		case r.Method == http.MethodGet && path == "/v1/models":
			body, err := modelsResponse(s.cfg.Models, s.cfg.ModelConfig)
			if err != nil {
				writeAnthropicError(w, http.StatusServiceUnavailable, "service_unavailable_error", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, body)
		case r.Method == http.MethodGet && path == "/v1/mcp_servers":
			writeJSON(w, http.StatusOK, map[string]any{"data": []any{}, "has_more": false})
		case r.Method == http.MethodPost && path == "/v1/messages/count_tokens":
			writeJSON(w, http.StatusOK, map[string]any{"input_tokens": estimateTokenCount(r)})
		case r.Method == http.MethodPost && path == "/v1/messages/batches":
			writeAnthropicError(w, http.StatusNotImplemented, "invalid_request_error", "message batches are not supported by Claudodex v1")
		case r.Method == http.MethodPost && path == "/v1/messages":
			s.handleMessages(w, r)
		case r.Method == http.MethodGet && path == "/api/claude_cli/bootstrap":
			writeJSON(w, http.StatusOK, bootstrapResponse(s.cfg.ModelConfig))
		case r.Method == http.MethodGet && path == "/api/oauth/usage":
			s.handleUsage(w, r)
		case r.Method == http.MethodGet && (path == "/api/oauth/profile" || path == "/api/claude_cli_profile"):
			writeJSON(w, http.StatusOK, claudeProfileResponse())
		case r.Method == http.MethodGet && path == "/api/claude_code/settings":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && path == "/api/claude_code/policy_limits":
			writeJSON(w, http.StatusOK, map[string]any{"restrictions": map[string]any{}})
		case r.Method == http.MethodGet && path == "/api/claude_code_penguin_mode":
			writeJSON(w, http.StatusOK, map[string]any{"enabled": true})
		default:
			if routeExists(path) {
				writeAnthropicError(w, http.StatusMethodNotAllowed, "invalid_request_error", "method not allowed")
				return
			}
			writeAnthropicError(w, http.StatusNotFound, "not_found_error", "route not found")
		}
	})
}

func (s *Server) logRequest(method string, path string, rawQuery string) {
	target := path
	if rawQuery != "" {
		target += "?" + rawQuery
	}
	s.appendProxyLog("%s %s %s\n", time.Now().UTC().Format(time.RFC3339Nano), method, target)
}

func (s *Server) appendProxyLog(format string, args ...any) {
	logPath := os.Getenv("CLAUDODEX_PROXY_LOG")
	if logPath == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(logPath), 0o700)
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, format, args...)
}

func authStatus(present bool) string {
	if present {
		return "present"
	}
	return "missing"
}

func claudeProfileResponse() map[string]any {
	return map[string]any{
		"account": map[string]any{
			"uuid":          "00000000-0000-4000-8000-000000000001",
			"email":         "claudodex@local",
			"email_address": "claudodex@local",
			"display_name":  "",
			"created_at":    "2026-01-01T00:00:00Z",
		},
		"organization": map[string]any{
			"uuid":                    "00000000-0000-4000-8000-000000000002",
			"name":                    "Claudodex",
			"organization_type":       "claude_max",
			"rate_limit_tier":         nil,
			"has_extra_usage_enabled": false,
			"billing_type":            nil,
			"subscription_created_at": "2026-01-01T00:00:00Z",
		},
	}
}

func estimateTokenCount(r *http.Request) int {
	const maxCountBody = 64 << 20
	data, err := io.ReadAll(io.LimitReader(r.Body, maxCountBody+1))
	if err != nil || len(data) == 0 {
		return 1
	}
	truncated := len(data) > maxCountBody
	if truncated {
		data = data[:maxCountBody]
	}
	return estimateTokenCountFromBytes(data, truncated)
}

func estimateTokenCountFromBytes(data []byte, truncated bool) int {
	tokens := (len(data) + 2) / 3 // deliberately conservative JSON chars/token estimate.
	tokens += estimateImagePadding(data)
	if truncated {
		tokens += 1_000_000
	}
	if tokens < 1 {
		return 1
	}
	return tokens
}

func estimateImagePadding(data []byte) int {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return 0
	}
	return imagePaddingValue(value)
}

func imagePaddingValue(value any) int {
	switch v := value.(type) {
	case map[string]any:
		padding := 0
		if typ, _ := v["type"].(string); isImageType(typ) {
			padding += 8500
		}
		if _, ok := v["image_url"]; ok {
			padding += 8500
		}
		if source, ok := v["source"].(map[string]any); ok {
			if typ, _ := source["type"].(string); typ == "base64" || typ == "url" {
				padding += 8500
			}
		}
		for _, child := range v {
			padding += imagePaddingValue(child)
		}
		return padding
	case []any:
		padding := 0
		for _, child := range v {
			padding += imagePaddingValue(child)
		}
		return padding
	default:
		return 0
	}
}

func isImageType(value string) bool {
	value = strings.ToLower(value)
	return value == "image" || value == "input_image"
}

func routeExists(path string) bool {
	switch path {
	case "/healthz", "/v1", "/v1/models", "/v1/mcp_servers", "/v1/messages/count_tokens", "/v1/messages/batches", "/v1/messages", "/api/claude_cli/bootstrap", "/api/oauth/usage", "/api/oauth/profile", "/api/claude_cli_profile", "/api/claude_code/settings", "/api/claude_code/policy_limits", "/api/claude_code_penguin_mode":
		return true
	default:
		return false
	}
}

func normalizePath(path string) string {
	for {
		switch {
		case strings.HasPrefix(path, "/v1/v1/"):
			path = "/v1/" + strings.TrimPrefix(path, "/v1/v1/")
		case strings.HasPrefix(path, "/api/v1/"):
			path = "/v1/" + strings.TrimPrefix(path, "/api/v1/")
		default:
			return path
		}
	}
}

func modelsResponse(models []codex.ModelInfo, modelCfg modelconfig.Config) (map[string]any, error) {
	modelCfg = modelCfg.Normalize()
	specs := append(modelconfig.ClaudeAliasSpecs(modelCfg), modelconfig.DirectModelSpecs(modelCfg)...)
	data := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		target := modelCfg.Target(spec.Family)
		contextWindow, ok := modelContextWindow(models, target)
		if !ok {
			return nil, fmt.Errorf("Codex model metadata missing context window for %s", target)
		}
		data = append(data, map[string]any{
			"id":               spec.ID,
			"type":             "model",
			"display_name":     spec.DisplayName,
			"max_input_tokens": contextWindow,
			"max_tokens":       128000,
		})
	}
	return map[string]any{
		"data": data,
	}, nil
}

func modelContextWindow(models []codex.ModelInfo, slug string) (int64, bool) {
	for _, model := range models {
		if !strings.EqualFold(strings.TrimSpace(model.Slug), slug) {
			continue
		}
		if model.ContextWindow > 0 {
			return model.ContextWindow, true
		}
		if model.MaxContextWindow > 0 {
			return model.MaxContextWindow, true
		}
	}
	return 0, false
}

func bootstrapResponse(modelCfg modelconfig.Config) map[string]any {
	modelCfg = modelCfg.Normalize()
	return map[string]any{
		"client_data": nil,
		"additional_model_options": []map[string]string{
			{"model": modelconfig.WithLongContext(modelCfg.Opus), "name": modelCfg.Opus, "description": modelCfg.Opus + " - default complex-work model"},
			{"model": modelconfig.WithLongContext(modelCfg.Sonnet), "name": modelCfg.Sonnet, "description": modelCfg.Sonnet + " - everyday coding model"},
			{"model": modelconfig.WithLongContext(modelCfg.Haiku), "name": modelCfg.Haiku, "description": modelCfg.Haiku + " - quick-answer model"},
		},
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeAnthropicError(w http.ResponseWriter, status int, typ, message string) {
	writeJSON(w, status, map[string]any{
		"type": "error",
		"error": map[string]string{
			"type":    typ,
			"message": message,
		},
	})
}
