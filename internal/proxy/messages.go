package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bassner/claudodex/internal/auth"
	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/convert"
	"github.com/bassner/claudodex/internal/ratelimit"
)

const maxMessagesBody = 64 << 20

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxMessagesBody+1))
	if err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "failed to read request body")
		return
	}
	if len(body) > maxMessagesBody {
		writeAnthropicError(w, http.StatusRequestEntityTooLarge, "invalid_request_error", "request body is too large")
		return
	}
	var anthropicReq convert.AnthropicRequest
	if err := json.Unmarshal(body, &anthropicReq); err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "request body is not valid JSON")
		return
	}

	sessionID := sessionIDFromRequest(r)
	result, err := convert.AnthropicToCodex(anthropicReq, convert.ConvertOptions{SessionID: sessionID, Models: s.cfg.ModelConfig})
	if err != nil {
		var bad convert.BadRequestError
		if errors.As(err, &bad) {
			writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", bad.Message)
			return
		}
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	upstream, err := s.createCodexResponse(r, result.Request, sessionID)
	if err != nil {
		writeMappedUpstreamError(w, err)
		return
	}
	defer upstream.Body.Close()

	if result.Stream {
		applyRateLimitHeaders(w.Header(), upstream.Header, false)
		s.streamAnthropicWithSchemas(w, upstream.Body, result.OriginalModel, result.ToolSchemas)
		return
	}
	applyRateLimitHeaders(w.Header(), upstream.Header, false)
	s.writeNonStreamingMessageWithSchemas(w, upstream.Body, result.OriginalModel, result.ToolSchemas)
}

func (s *Server) createCodexResponse(r *http.Request, req codex.Request, sessionID string) (*http.Response, error) {
	store := auth.NewStore(s.cfg.Home)
	file, err := auth.NewRefresher(store, s.cfg.HTTPClient, s.cfg.TokenEndpoint).EnsureFresh(r.Context(), 5*time.Minute)
	if err != nil {
		return nil, err
	}
	installationID, err := auth.InstallationID(s.cfg.Home)
	if err != nil {
		return nil, err
	}
	req.ClientMetadata = map[string]string{"x-codex-installation-id": installationID}

	client := codex.Client{BaseURL: s.cfg.CodexBaseURL, HTTPClient: s.cfg.HTTPClient, Version: s.cfg.Version}
	credentials := codex.Credentials{
		AccessToken:    file.Tokens.AccessToken,
		AccountID:      file.Tokens.AccountID,
		InstallationID: installationID,
		FedRAMP:        file.Tokens.ChatGPTAccountIsFedRAMP,
	}
	route := codex.Route{SessionID: sessionID, ThreadID: sessionID}
	resp, err := client.CreateResponse(r.Context(), req, credentials, route)
	if err == nil {
		return resp, nil
	}
	var upstream *codex.UpstreamError
	if !errors.As(err, &upstream) || upstream.Status != http.StatusUnauthorized {
		return nil, err
	}

	file, refreshErr := auth.NewRefresher(store, s.cfg.HTTPClient, s.cfg.TokenEndpoint).Refresh(r.Context())
	if refreshErr != nil {
		return nil, refreshErr
	}
	credentials.AccessToken = file.Tokens.AccessToken
	credentials.AccountID = file.Tokens.AccountID
	credentials.FedRAMP = file.Tokens.ChatGPTAccountIsFedRAMP
	return client.CreateResponse(r.Context(), req, credentials, route)
}

func (s *Server) streamAnthropic(w http.ResponseWriter, body io.Reader, model string) {
	s.streamAnthropicWithSchemas(w, body, model, nil)
}

func (s *Server) streamAnthropicWithSchemas(w http.ResponseWriter, body io.Reader, model string, toolSchemas map[string]map[string]any) {
	w.Header().Set("content-type", "text/event-stream")
	w.Header().Set("cache-control", "no-cache")
	w.Header().Set("connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	reducer := convert.NewStreamReducerWithOptions("", model, convert.StreamReducerOptions{
		ToolSchemas: toolSchemas,
	})
	err := codex.ReadSSE(body, func(event codex.SSEEvent) error {
		events, err := reducer.ReduceNamed(event.Event, event.Data)
		if err != nil {
			return err
		}
		for _, anthropicEvent := range events {
			if err := writeSSE(w, anthropicEvent); err != nil {
				return err
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		return nil
	})
	if err != nil {
		_ = writeSSE(w, streamErrorEvent("Codex upstream stream failed"))
		if flusher != nil {
			flusher.Flush()
		}
		return
	}
	if !reducer.Done() {
		_ = writeSSE(w, streamErrorEvent("Codex upstream stream ended before completion"))
		if flusher != nil {
			flusher.Flush()
		}
	}
}

func (s *Server) writeNonStreamingMessage(w http.ResponseWriter, body io.Reader, model string) {
	s.writeNonStreamingMessageWithSchemas(w, body, model, nil)
}

func (s *Server) writeNonStreamingMessageWithSchemas(w http.ResponseWriter, body io.Reader, model string, toolSchemas map[string]map[string]any) {
	reducer := convert.NewStreamReducerWithOptions("", model, convert.StreamReducerOptions{
		ToolSchemas: toolSchemas,
	})
	var events []convert.AnthropicSSE
	err := codex.ReadSSE(body, func(event codex.SSEEvent) error {
		next, err := reducer.ReduceNamed(event.Event, event.Data)
		if err != nil {
			return err
		}
		events = append(events, next...)
		return nil
	})
	if err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "api_error", "Codex upstream stream failed")
		return
	}
	if !reducer.Done() {
		writeAnthropicError(w, http.StatusBadGateway, "api_error", "Codex upstream stream ended before completion")
		return
	}
	message, errEvent := convert.AssembleMessage(events, "", model)
	if errEvent != nil {
		writeJSON(w, http.StatusBadGateway, errEvent.Data)
		return
	}
	writeJSON(w, http.StatusOK, message)
}

func streamErrorEvent(message string) convert.AnthropicSSE {
	return convert.AnthropicSSE{
		Event: "error",
		Data: map[string]any{
			"type": "error",
			"error": map[string]any{
				"type":    "api_error",
				"message": message,
			},
		},
	}
}

func writeSSE(w io.Writer, event convert.AnthropicSSE) error {
	data, err := json.Marshal(event.Data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Event, data)
	return err
}

func applyRateLimitHeaders(dst http.Header, upstream http.Header, forceRejected bool) {
	snapshot := ratelimit.FromCodexHeaders(upstream)
	if snapshot == nil && forceRejected {
		snapshot = ratelimit.SnapshotFromRetryAfter(upstream, time.Now())
	}
	ratelimit.ApplyAnthropicHeaders(dst, snapshot, forceRejected)
}

func sessionIDFromRequest(r *http.Request) string {
	for _, key := range []string{"X-Claude-Code-Session-Id", "x-claude-code-session-id", "x-client-request-id"} {
		if value := strings.TrimSpace(r.Header.Get(key)); value != "" {
			return sanitizeRouteID(value)
		}
	}
	return ""
}

func sanitizeRouteID(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 128 {
		return value[:128]
	}
	return value
}
