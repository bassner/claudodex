package codex

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const DefaultModelsClientVersion = "0.133.0"

type ModelsResponse struct {
	Models []ModelInfo `json:"models"`
}

type ModelInfo struct {
	Slug                          string `json:"slug"`
	DisplayName                   string `json:"display_name"`
	Description                   string `json:"description,omitempty"`
	ContextWindow                 int64  `json:"context_window,omitempty"`
	MaxContextWindow              int64  `json:"max_context_window,omitempty"`
	AutoCompactTokenLimit         int64  `json:"auto_compact_token_limit,omitempty"`
	EffectiveContextWindowPercent int64  `json:"effective_context_window_percent,omitempty"`
	SupportedInAPI                bool   `json:"supported_in_api"`
	Visibility                    string `json:"visibility"`
}

func (c Client) FetchModels(ctx context.Context, credentials Credentials) ([]ModelInfo, error) {
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	endpoint, err := url.Parse(baseURL + "/codex/models")
	if err != nil {
		return nil, err
	}
	query := endpoint.Query()
	query.Set("client_version", modelsClientVersion(c.Version))
	endpoint.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	for key, value := range c.headers(credentials, Route{}, false) {
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
	var out ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Models, nil
}

func validClientVersion(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}
	return true
}

func modelsClientVersion(version string) string {
	if validClientVersion(version) {
		return version
	}
	return DefaultModelsClientVersion
}
