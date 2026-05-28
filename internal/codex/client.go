package codex

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Version    string
}

type UpstreamError struct {
	Status     int
	Content    []byte
	Header     http.Header
	ContentTyp string
}

func (e *UpstreamError) Error() string {
	if e.Status == 0 {
		return "Codex upstream request failed"
	}
	return fmt.Sprintf("Codex upstream returned HTTP %d", e.Status)
}

func (c Client) CreateResponse(ctx context.Context, request Request, credentials Credentials, route Route) (*http.Response, error) {
	includeHTTPBeta := httpBetaHeaderEnabled()
	resp, err := c.createResponse(ctx, request, credentials, route, includeHTTPBeta)
	if err == nil {
		return resp, nil
	}
	var upstream *UpstreamError
	if includeHTTPBeta && errors.As(err, &upstream) && upstream.Status == http.StatusBadRequest {
		return c.createResponse(ctx, request, credentials, route, false)
	}
	return nil, err
}

func (c Client) createResponse(ctx context.Context, request Request, credentials Credentials, route Route, includeHTTPBeta bool) (*http.Response, error) {
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	data, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/codex/responses", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	for key, value := range c.headers(credentials, route, includeHTTPBeta) {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return nil, &UpstreamError{
			Status:     resp.StatusCode,
			Content:    body,
			Header:     resp.Header.Clone(),
			ContentTyp: resp.Header.Get("content-type"),
		}
	}
	return resp, nil
}

func (c Client) headers(credentials Credentials, route Route, includeHTTPBeta bool) map[string]string {
	version := c.Version
	if version == "" {
		version = "dev"
	}
	sessionID := route.SessionID
	if sessionID == "" {
		sessionID = mustUUID()
	}
	threadID := route.ThreadID
	if threadID == "" {
		threadID = sessionID
	}
	headers := map[string]string{
		"authorization":           "Bearer " + credentials.AccessToken,
		"content-type":            "application/json",
		"accept":                  "text/event-stream",
		"originator":              "codex_cli_rs",
		"user-agent":              fmt.Sprintf("codex_cli_rs/0.0.0 (%s; %s) claudodex/%s", runtime.GOOS, runtime.GOARCH, version),
		"x-codex-installation-id": credentials.InstallationID,
		"x-codex-turn-state":      "null",
		"x-client-request-id":     sessionID,
		"session-id":              sessionID,
		"thread-id":               threadID,
		"chatgpt-account-id":      credentials.AccountID,
	}
	if includeHTTPBeta {
		headers["openai-beta"] = "responses_websockets=2026-02-06"
	}
	if credentials.FedRAMP {
		headers["x-openai-internal-codex-residency"] = "us"
		headers["x-openai-fedramp"] = "true"
	}
	if strings.TrimSpace(credentials.AccountID) == "" {
		delete(headers, "chatgpt-account-id")
	}
	if strings.TrimSpace(credentials.InstallationID) == "" {
		delete(headers, "x-codex-installation-id")
	}
	if strings.TrimSpace(credentials.AccessToken) == "" {
		delete(headers, "authorization")
	}
	if strings.TrimSpace(sessionID) == "" {
		delete(headers, "x-client-request-id")
		delete(headers, "session-id")
	}
	if strings.TrimSpace(threadID) == "" {
		delete(headers, "thread-id")
	}
	return headers
}

func httpBetaHeaderEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("CLAUDODEX_HTTP_BETA_HEADER")))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func mustUUID() string {
	id, err := uuidV4()
	if err != nil {
		return "claudodex-session"
	}
	return id
}

func uuidV4() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
