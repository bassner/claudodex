package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bassner/claudodex/internal/codex"
)

func TestWriteMappedUpstreamErrorStatusTable(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       []byte
		wantStatus int
		wantType   string
		wantMsg    string
	}{
		{
			name:       "bad request",
			status:     http.StatusBadRequest,
			body:       []byte(`{"error":{"message":"invalid model"}}`),
			wantStatus: http.StatusBadRequest,
			wantType:   "invalid_request_error",
			wantMsg:    "invalid model",
		},
		{
			name:       "unauthorized",
			status:     http.StatusUnauthorized,
			body:       []byte(`{"detail":"expired"}`),
			wantStatus: http.StatusUnauthorized,
			wantType:   "authentication_error",
			wantMsg:    "expired",
		},
		{
			name:       "forbidden non json",
			status:     http.StatusForbidden,
			body:       []byte(`<html>forbidden</html>`),
			wantStatus: http.StatusForbidden,
			wantType:   "permission_error",
			wantMsg:    "Codex upstream returned HTTP 403",
		},
		{
			name:       "not found",
			status:     http.StatusNotFound,
			body:       []byte(`{"message":"missing"}`),
			wantStatus: http.StatusNotFound,
			wantType:   "not_found_error",
			wantMsg:    "missing",
		},
		{
			name:       "too large",
			status:     http.StatusRequestEntityTooLarge,
			body:       nil,
			wantStatus: http.StatusRequestEntityTooLarge,
			wantType:   "invalid_request_error",
			wantMsg:    "Codex upstream rejected the prompt as too large",
		},
		{
			name:       "rate limit",
			status:     http.StatusTooManyRequests,
			body:       nil,
			wantStatus: http.StatusTooManyRequests,
			wantType:   "rate_limit_error",
			wantMsg:    "Codex upstream rate limit reached",
		},
		{
			name:       "bad gateway",
			status:     http.StatusBadGateway,
			body:       []byte(`{"error":"overloaded"}`),
			wantStatus: http.StatusBadGateway,
			wantType:   "overloaded_error",
			wantMsg:    "overloaded",
		},
		{
			name:       "server error",
			status:     http.StatusInternalServerError,
			body:       nil,
			wantStatus: http.StatusInternalServerError,
			wantType:   "api_error",
			wantMsg:    "Codex upstream returned HTTP 500",
		},
		{
			name:       "overloaded custom status",
			status:     529,
			body:       []byte(`{"error":{"message":"overloaded"}}`),
			wantStatus: 529,
			wantType:   "overloaded_error",
			wantMsg:    "overloaded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeMappedUpstreamError(rec, &codex.UpstreamError{
				Status:  tt.status,
				Content: tt.body,
				Header:  http.Header{},
			})
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			var got struct {
				Type  string `json:"type"`
				Error struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			if got.Type != "error" || got.Error.Type != tt.wantType || got.Error.Message != tt.wantMsg {
				t.Fatalf("body = %#v", got)
			}
		})
	}
}

func TestWriteMappedUpstreamErrorMapsResponseHeaderTimeoutToSafe504(t *testing.T) {
	rec := httptest.NewRecorder()
	writeMappedUpstreamError(rec, &codex.ResponseHeaderTimeoutError{Timeout: 45 * time.Second, Attempts: 2})
	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusGatewayTimeout)
	}
	var got struct {
		Type  string `json:"type"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Type != "error" || got.Error.Type != "api_error" || got.Error.Message != "Codex upstream timed out waiting for response headers" {
		t.Fatalf("body = %#v", got)
	}
	if strings.Contains(got.Error.Message, "45s") || strings.Contains(got.Error.Message, "2 attempts") {
		t.Fatalf("response exposed internal timeout details: %q", got.Error.Message)
	}
}
