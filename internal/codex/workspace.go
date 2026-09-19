package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const accountRoutingHeader = "x-openai-account-routing-override"

type responsesDestination struct {
	baseURL       string
	routingHeader string
	routed        bool
}

type accountsCheckResponse struct {
	Accounts []workspaceAccount `json:"accounts"`
}

type workspaceAccount struct {
	ID                     string `json:"id"`
	AccountID              string `json:"account_id"`
	WorkspaceBackendOrigin string `json:"workspace_backend_origin"`
	AccountRoutingOverride string `json:"account_routing_override"`
}

func (c Client) resolveResponsesDestination(ctx context.Context, credentials Credentials) (responsesDestination, error) {
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	destination := responsesDestination{baseURL: baseURL}
	accountID := strings.TrimSpace(credentials.AccountID)
	if accountID == "" || !c.workspaceRoutingEligible(baseURL) {
		return destination, nil
	}

	endpoint, err := accountsCheckURL(baseURL)
	if err != nil {
		return responsesDestination{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return responsesDestination{}, err
	}
	for key, value := range c.headers(credentials, Route{}, nil, false) {
		req.Header.Set(key, value)
	}
	req.Header.Del("content-type")
	req.Header.Set("accept", "application/json")

	resp, err := c.noRedirectHTTPClient().Do(req)
	if err != nil {
		return responsesDestination{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return responsesDestination{}, &UpstreamError{
			Status:     resp.StatusCode,
			Content:    body,
			Header:     resp.Header.Clone(),
			ContentTyp: resp.Header.Get("content-type"),
		}
	}
	var payload accountsCheckResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return responsesDestination{}, fmt.Errorf("decode accounts check response: %w", err)
	}
	for _, account := range payload.Accounts {
		id := strings.TrimSpace(account.ID)
		if id == "" {
			id = strings.TrimSpace(account.AccountID)
		}
		if id != accountID {
			continue
		}
		resolvedBaseURL, routingHeader, err := applyWorkspaceRouting(baseURL, account.WorkspaceBackendOrigin, account.AccountRoutingOverride)
		if err != nil {
			return responsesDestination{}, err
		}
		return responsesDestination{
			baseURL:       resolvedBaseURL,
			routingHeader: routingHeader,
			routed:        true,
		}, nil
	}
	return responsesDestination{}, fmt.Errorf("selected ChatGPT workspace %q was not returned by accounts check", accountID)
}

// workspaceRoutingEligible keeps independent or explicitly configured providers
// off ChatGPT's account-discovery path. Tests and embedded callers can replace
// the ChatGPT bootstrap URL without changing the Responses provider URL.
func (c Client) workspaceRoutingEligible(baseURL string) bool {
	chatGPTBaseURL := strings.TrimSpace(c.ChatGPTBaseURL)
	if chatGPTBaseURL == "" {
		chatGPTBaseURL = DefaultBaseURL
	}
	provider, providerErr := url.Parse(baseURL)
	bootstrap, bootstrapErr := url.Parse(chatGPTBaseURL)
	if providerErr != nil || bootstrapErr != nil {
		return false
	}
	return strings.EqualFold(provider.Scheme, bootstrap.Scheme) && strings.EqualFold(provider.Host, bootstrap.Host)
}

func accountsCheckURL(baseURL string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	path := strings.TrimRight(parsed.Path, "/")
	if strings.Contains(path, "/backend-api") {
		parsed.Path = path + "/wham/accounts/check"
	} else if strings.HasSuffix(path, "/api/codex") {
		parsed.Path = path + "/accounts/check"
	} else {
		parsed.Path = path + "/api/codex/accounts/check"
	}
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func applyWorkspaceRouting(baseURL, backendOrigin, routingOverride string) (string, string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", "", err
	}
	backendOrigin = strings.TrimSpace(backendOrigin)
	if backendOrigin != "NO_CONSTRAINT" {
		backend, err := url.Parse(backendOrigin)
		if err != nil {
			return "", "", fmt.Errorf("invalid workspace backend origin: %w", err)
		}
		if backend.Scheme != "https" || backend.Hostname() == "" || backend.User != nil ||
			(backend.Path != "" && backend.Path != "/") || backend.RawQuery != "" || backend.Fragment != "" {
			return "", "", fmt.Errorf("invalid workspace backend origin")
		}
		base.Scheme = backend.Scheme
		base.Host = backend.Host
	}

	routingOverride = strings.TrimSpace(routingOverride)
	routingHeader := ""
	switch routingOverride {
	case "NO_CONSTRAINT":
	case "us", "us_cr":
		routingHeader = routingOverride
	default:
		return "", "", fmt.Errorf("invalid workspace routing override")
	}

	return strings.TrimRight(base.String(), "/"), routingHeader, nil
}

func (c Client) noRedirectHTTPClient() *http.Client {
	client := c.routingCookieHTTPClient()
	clone := *client
	clone.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &clone
}
