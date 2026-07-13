package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/bassner/claudodex/internal/auth"
	"github.com/bassner/claudodex/internal/codex"
)

func writeMappedUpstreamError(w http.ResponseWriter, err error) {
	if errors.Is(err, auth.ErrNotLoggedIn) || errors.Is(err, auth.ErrInvalidAuth) || errors.Is(err, auth.ErrPermanentRefreshFailure) {
		writeAnthropicError(w, http.StatusUnauthorized, "authentication_error", "Codex authentication is missing or expired; run claudodex clx:auth-login")
		return
	}
	var timeoutErr *codex.ResponseHeaderTimeoutError
	if errors.As(err, &timeoutErr) {
		writeAnthropicError(w, http.StatusGatewayTimeout, "api_error", "Codex upstream timed out waiting for response headers")
		return
	}
	var upstream *codex.UpstreamError
	if errors.As(err, &upstream) {
		status, typ := mapUpstreamStatus(upstream.Status)
		applyRateLimitHeaders(w.Header(), upstream.Header, status == http.StatusTooManyRequests)
		writeAnthropicError(w, status, typ, upstreamMessage(upstream))
		return
	}
	writeAnthropicError(w, http.StatusBadGateway, "api_error", "Codex upstream request failed")
}

func mapUpstreamStatus(status int) (int, string) {
	switch status {
	case http.StatusBadRequest:
		return status, "invalid_request_error"
	case http.StatusUnauthorized:
		return status, "authentication_error"
	case http.StatusForbidden:
		return status, "permission_error"
	case http.StatusNotFound:
		return status, "not_found_error"
	case http.StatusRequestTimeout:
		return status, "api_error"
	case http.StatusRequestEntityTooLarge:
		return status, "invalid_request_error"
	case http.StatusTooManyRequests:
		return status, "rate_limit_error"
	case http.StatusBadGateway, http.StatusServiceUnavailable, 529:
		return status, "overloaded_error"
	default:
		if status >= 500 {
			return status, "api_error"
		}
		if status == 0 {
			return http.StatusBadGateway, "api_error"
		}
		return status, "api_error"
	}
}

func upstreamMessage(err *codex.UpstreamError) string {
	var parsed struct {
		Error   any    `json:"error"`
		Detail  any    `json:"detail"`
		Message string `json:"message"`
	}
	if json.Unmarshal(err.Content, &parsed) == nil {
		if msg := nestedMessage(parsed.Error); msg != "" {
			return msg
		}
		if msg := nestedMessage(parsed.Detail); msg != "" {
			return msg
		}
		if parsed.Message != "" {
			return parsed.Message
		}
	}
	if err.Status == http.StatusRequestEntityTooLarge {
		return "Codex upstream rejected the prompt as too large"
	}
	if err.Status == http.StatusTooManyRequests {
		return "Codex upstream rate limit reached"
	}
	return fmt.Sprintf("Codex upstream returned HTTP %d", err.Status)
}

func nestedMessage(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case map[string]any:
		if msg, _ := v["message"].(string); msg != "" {
			return msg
		}
		if msg, _ := v["detail"].(string); msg != "" {
			return msg
		}
	}
	return ""
}
