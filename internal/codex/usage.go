package codex

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func (c Client) FetchUsage(ctx context.Context, credentials Credentials) (map[string]any, error) {
	client := CredentialHTTPClient(c.routingCookieHTTPClient(), credentials)
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if err := CheckCredentialDestination(credentials, baseURL); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/wham/usage", nil)
	if err != nil {
		return nil, err
	}
	for key, value := range c.headers(credentials, Route{}, nil, false) {
		req.Header.Set(key, value)
	}
	req.Header.Del("content-type")
	req.Header.Set("accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return nil, &UpstreamError{
			Status:     resp.StatusCode,
			Content:    body,
			Header:     resp.Header.Clone(),
			ContentTyp: resp.Header.Get("content-type"),
		}
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
