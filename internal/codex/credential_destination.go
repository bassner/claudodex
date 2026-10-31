package codex

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// CheckCredentialDestination checks a credential's source constraint before any
// network operation. Direct Claudodex credentials have no source constraint.
func CheckCredentialDestination(credentials Credentials, destination string) error {
	if credentials.AllowedOrigin == "" {
		return nil
	}
	allowed, err := credentialOrigin(credentials.AllowedOrigin)
	if err != nil || allowed != "https://chatgpt.com" {
		return fmt.Errorf("borrowed Codex credentials have an invalid allowed origin")
	}
	actual, err := credentialOrigin(destination)
	if err != nil || actual != allowed {
		return fmt.Errorf("local Codex login can only be used with the official ChatGPT backend; use a direct Claudodex login for a custom backend")
	}
	return nil
}

func credentialOrigin(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Hostname() == "" || u.Opaque != "" {
		return "", fmt.Errorf("invalid credential destination")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme == "wss" {
		scheme = "https"
	}
	if scheme != "https" || (u.Port() != "" && u.Port() != "443") {
		return "", fmt.Errorf("invalid credential destination")
	}
	return scheme + "://" + strings.ToLower(u.Hostname()), nil
}

// CredentialHTTPClient preserves the caller's client and redirect policy while
// enforcing the source constraint on both redirects and each transport request.
func CredentialHTTPClient(client *http.Client, credentials Credentials) *http.Client {
	if client == nil {
		client = http.DefaultClient
	}
	if credentials.AllowedOrigin == "" {
		return client
	}
	clone := *client
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	clone.Transport = credentialDestinationTransport{base: base, credentials: credentials}
	previous := client.CheckRedirect
	clone.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if err := CheckCredentialDestination(credentials, req.URL.String()); err != nil {
			return err
		}
		if previous != nil {
			return previous(req, via)
		}
		if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		return nil
	}
	return &clone
}

type credentialDestinationTransport struct {
	base        http.RoundTripper
	credentials Credentials
}

func (t credentialDestinationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := CheckCredentialDestination(t.credentials, req.URL.String()); err != nil {
		return nil, err
	}
	return t.base.RoundTrip(req)
}
