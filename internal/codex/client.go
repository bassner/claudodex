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
	"net/http/httptrace"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	responseHeaderTimeoutEnv     = "CLAUDODEX_CODEX_RESPONSE_HEADER_TIMEOUT"
	defaultResponseHeaderTimeout = 45 * time.Second
	responseHeaderRetriesHeader  = "x-claudodex-response-header-retries"
)

type Client struct {
	BaseURL                string
	HTTPClient             *http.Client
	Version                string
	ResponseHeaderTimeout  time.Duration
	ResponseHeaderAttempts int
}

type ResponseHeaderTimeoutError struct {
	Timeout  time.Duration
	Attempts int
}

type createResponseAttemptError struct {
	err      error
	attempts int
}

func (e *createResponseAttemptError) Error() string { return e.err.Error() }
func (e *createResponseAttemptError) Unwrap() error { return e.err }

func (e *ResponseHeaderTimeoutError) Error() string {
	if e.Attempts > 1 {
		return fmt.Sprintf("timed out waiting %s for Codex upstream response headers after %d attempts", e.Timeout, e.Attempts)
	}
	return fmt.Sprintf("timed out waiting %s for Codex upstream response headers", e.Timeout)
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
	route = MaterializeRoute(route)
	maxAttempts := c.responseHeaderAttempts()
	includeHTTPBeta := httpBetaHeaderEnabled()
	resp, attempts, err := c.createResponseWithHeaderTimeoutRetry(ctx, request, credentials, route, includeHTTPBeta, maxAttempts)
	if err == nil {
		SetCreateResponseAttempts(resp, attempts)
		return resp, nil
	}
	var upstream *UpstreamError
	if includeHTTPBeta && attempts < maxAttempts && errors.As(err, &upstream) && upstream.Status == http.StatusBadRequest {
		fallbackResp, fallbackAttempts, fallbackErr := c.createResponseWithHeaderTimeoutRetry(ctx, request, credentials, route, false, maxAttempts-attempts)
		attempts += fallbackAttempts
		if fallbackErr == nil {
			SetCreateResponseAttempts(fallbackResp, attempts)
			return fallbackResp, nil
		}
		var timeoutErr *ResponseHeaderTimeoutError
		if errors.As(fallbackErr, &timeoutErr) {
			timeoutErr.Attempts = attempts
			return nil, fallbackErr
		}
		return nil, WithCreateResponseAttempts(fallbackErr, attempts)
	}
	return nil, err
}

func (c Client) createResponseWithHeaderTimeoutRetry(ctx context.Context, request Request, credentials Credentials, route Route, includeHTTPBeta bool, maxAttempts int) (*http.Response, int, error) {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := c.createResponse(ctx, request, credentials, route, includeHTTPBeta)
		if err == nil {
			return resp, attempt, nil
		}
		var timeoutErr *ResponseHeaderTimeoutError
		if !errors.As(err, &timeoutErr) || ctx.Err() != nil {
			return nil, attempt, err
		}
		timeoutErr.Attempts = attempt
		if attempt == maxAttempts {
			return nil, attempt, err
		}
	}
	panic("unreachable")
}

// SetCreateResponseAttempts records the total generation attempts represented by resp.
func SetCreateResponseAttempts(resp *http.Response, attempts int) {
	if resp == nil {
		return
	}
	if attempts < 1 {
		attempts = 1
	}
	if resp.Header == nil {
		resp.Header = make(http.Header)
	}
	resp.Header.Set(responseHeaderRetriesHeader, strconv.Itoa(attempts-1))
}

func (c Client) responseHeaderAttempts() int {
	if c.ResponseHeaderAttempts > 0 {
		return c.ResponseHeaderAttempts
	}
	return 2
}

func CreateResponseAttempts(resp *http.Response, err error) int {
	var attemptErr *createResponseAttemptError
	if errors.As(err, &attemptErr) && attemptErr.attempts > 0 {
		return attemptErr.attempts
	}
	var timeoutErr *ResponseHeaderTimeoutError
	if errors.As(err, &timeoutErr) && timeoutErr.Attempts > 0 {
		return timeoutErr.Attempts
	}
	if resp != nil {
		if retries, parseErr := strconv.Atoi(resp.Header.Get(responseHeaderRetriesHeader)); parseErr == nil && retries >= 0 {
			return retries + 1
		}
	}
	return 1
}

// WithCreateResponseAttempts records the total generation attempts represented by err.
func WithCreateResponseAttempts(err error, attempts int) error {
	if err == nil || attempts <= 1 {
		return err
	}
	var timeoutErr *ResponseHeaderTimeoutError
	if errors.As(err, &timeoutErr) {
		timeoutErr.Attempts = attempts
		return err
	}
	return &createResponseAttemptError{err: err, attempts: attempts}
}

// MaterializeRoute fills missing session and thread IDs once for reuse across retries.
func MaterializeRoute(route Route) Route {
	if strings.TrimSpace(route.SessionID) == "" {
		route.SessionID = mustUUID()
	}
	if strings.TrimSpace(route.ThreadID) == "" {
		route.ThreadID = route.SessionID
	}
	return route
}

func (c Client) createResponse(ctx context.Context, request Request, credentials Credentials, route Route, includeHTTPBeta bool) (*http.Response, error) {
	client := c.responseHTTPClient()
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

func (c Client) responseHTTPClient() *http.Client {
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	clone := *client
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	clone.Transport = responseHeaderTimeoutRoundTripper{
		base:    transport,
		timeout: c.responseHeaderTimeout(),
	}
	return &clone
}

func (c Client) responseHeaderTimeout() time.Duration {
	if c.ResponseHeaderTimeout != 0 {
		if c.ResponseHeaderTimeout < 0 {
			return 0
		}
		return c.ResponseHeaderTimeout
	}
	value := strings.TrimSpace(os.Getenv(responseHeaderTimeoutEnv))
	if value == "" {
		return defaultResponseHeaderTimeout
	}
	switch strings.ToLower(value) {
	case "0", "false", "no", "off":
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	timeout, err := time.ParseDuration(value)
	if err != nil || timeout <= 0 {
		return defaultResponseHeaderTimeout
	}
	return timeout
}

type responseHeaderTimeoutRoundTripper struct {
	base    http.RoundTripper
	timeout time.Duration
}

func (t responseHeaderTimeoutRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.timeout <= 0 {
		return t.base.RoundTrip(req)
	}

	parentCtx := req.Context()
	ctx, cancel := context.WithCancel(parentCtx)
	wait := responseHeaderWait{timeout: t.timeout, cancel: cancel}
	wait.reset()
	trace := &httptrace.ClientTrace{
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			if info.Err == nil {
				wait.reset()
			}
		},
	}
	req = req.Clone(httptrace.WithClientTrace(ctx, trace))
	resp, err := t.base.RoundTrip(req)
	timedOut := wait.finish()
	if timedOut {
		cancel()
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if parentCtx.Err() != nil {
			return nil, parentCtx.Err()
		}
		return nil, &ResponseHeaderTimeoutError{Timeout: t.timeout, Attempts: 1}
	}
	if err != nil {
		cancel()
		return resp, err
	}
	if resp.Body == nil {
		cancel()
		return resp, nil
	}
	resp.Body = &cancelOnCloseReadCloser{ReadCloser: resp.Body, cancel: cancel}
	return resp, nil
}

type responseHeaderWait struct {
	mu       sync.Mutex
	timeout  time.Duration
	cancel   context.CancelFunc
	timer    *time.Timer
	finished bool
	timedOut bool
}

func (w *responseHeaderWait) reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.finished || w.timedOut {
		return
	}
	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(w.timeout, func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		if w.finished {
			return
		}
		w.timedOut = true
		w.cancel()
	})
}

func (w *responseHeaderWait) finish() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.finished = true
	if w.timer != nil {
		w.timer.Stop()
	}
	return w.timedOut
}

type cancelOnCloseReadCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (r *cancelOnCloseReadCloser) Close() error {
	err := r.ReadCloser.Close()
	r.cancel()
	return err
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
	if parentThreadID := strings.TrimSpace(route.ParentThreadID); parentThreadID != "" {
		headers["x-codex-parent-thread-id"] = parentThreadID
	}
	if subagent := strings.TrimSpace(route.Subagent); subagent != "" {
		headers["x-openai-subagent"] = subagent
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
