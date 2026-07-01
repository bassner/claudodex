package launcher

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestOAuthProxyInterceptsAnthropicUsage(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/oauth/usage" {
			t.Fatalf("path = %q, want /api/oauth/usage", r.URL.Path)
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"five_hour":{"utilization":12}}`))
	}))
	defer target.Close()

	proxy, err := StartOAuthProxy(target.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer proxy.Close()

	ca, err := os.ReadFile(proxy.CAPath())
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca) {
		t.Fatal("failed to append proxy CA")
	}
	proxyURL, err := url.Parse(proxy.ProxyURL())
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: &http.Transport{
		Proxy:           http.ProxyURL(proxyURL),
		TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
	}}
	resp, err := client.Get("https://api.anthropic.com/api/oauth/usage")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || string(body) != `{"five_hour":{"utilization":12}}` {
		t.Fatalf("status/body = %d %s", resp.StatusCode, body)
	}
}

func TestOAuthProxyForwardsAnthropicPostBodyAndSSE(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("path = %q, want /v1/messages", r.URL.Path)
		}
		if got := r.Header.Get("anthropic-version"); got != "2023-06-01" {
			t.Fatalf("anthropic-version = %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != `{"stream":true}` {
			t.Fatalf("body = %q", body)
		}
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte("event: message_start\ndata: {\"type\":\"message_start\"}\n\n"))
	}))
	defer target.Close()

	client := oauthProxyTestClient(t, target.URL)
	req, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", strings.NewReader(`{"stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body %s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("content-type"); got != "text/event-stream" {
		t.Fatalf("content-type = %q", got)
	}
	if !strings.Contains(string(body), "message_start") {
		t.Fatalf("body missing SSE event: %s", body)
	}
}

func TestOAuthProxyRejectsUnknownAnthropicRoute(t *testing.T) {
	client := oauthProxyTestClient(t, "http://127.0.0.1:1")
	resp, err := client.Get("https://api.anthropic.com/api/oauth/claude_cli/create_api_key")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestOAuthProxyRoutesRemoteControlOnlyToAnthropic(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   oauthProxyRoute
	}{
		{http.MethodPost, "/v1/messages", oauthProxyRouteLocal},
		{http.MethodGet, "/v1/models", oauthProxyRouteLocal},
		{http.MethodGet, "/api/oauth/profile", oauthProxyRouteLocal},
		{http.MethodPost, "/v1/sessions", oauthProxyRouteAnthropic},
		{http.MethodGet, "/v1/sessions/session_123/events", oauthProxyRouteAnthropic},
		{http.MethodGet, "/v1/sessions/ws/session_123/subscribe", oauthProxyRouteAnthropic},
		{http.MethodPost, "/v1/code/sessions", oauthProxyRouteAnthropic},
		{http.MethodPost, "/v1/code/sessions/cse_123/bridge", oauthProxyRouteAnthropic},
		{http.MethodGet, "/v1/code/sessions/cse_123/worker/events/stream", oauthProxyRouteAnthropic},
		{http.MethodPost, "/v1/environments/bridge", oauthProxyRouteAnthropic},
		{http.MethodPut, "/v1/session_ingress/session/session_123", oauthProxyRouteAnthropic},
		{http.MethodGet, "/api/oauth/files/file_123/content", oauthProxyRouteAnthropic},
		{http.MethodPost, "/api/oauth/files/file_123/content", oauthProxyRouteNone},
		{http.MethodGet, "/api/oauth/claude_cli/create_api_key", oauthProxyRouteNone},
	}
	for _, tt := range tests {
		if got := oauthProxyRouteFor(tt.method, tt.path); got != tt.want {
			t.Fatalf("%s %s route = %v, want %v", tt.method, tt.path, got, tt.want)
		}
	}
}

func TestOAuthProxyCertificateValidityCoversLongSessions(t *testing.T) {
	cert, caPEM, err := generateOAuthProxyCertificate()
	if err != nil {
		t.Fatal(err)
	}
	if len(cert.Certificate) == 0 {
		t.Fatal("missing server certificate")
	}
	serverCert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	caBlock, _ := pem.Decode(caPEM)
	if caBlock == nil {
		t.Fatal("missing CA PEM block")
	}
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	wantMin := 300 * 24 * time.Hour
	if remaining := time.Until(serverCert.NotAfter); remaining < wantMin {
		t.Fatalf("server certificate lifetime remaining = %s, want at least %s", remaining, wantMin)
	}
	if remaining := time.Until(caCert.NotAfter); remaining < wantMin {
		t.Fatalf("CA certificate lifetime remaining = %s, want at least %s", remaining, wantMin)
	}
}

func oauthProxyTestClient(t *testing.T, targetURL string) *http.Client {
	t.Helper()
	proxy, err := StartOAuthProxy(targetURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = proxy.Close() })

	ca, err := os.ReadFile(proxy.CAPath())
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca) {
		t.Fatal("failed to append proxy CA")
	}
	proxyURL, err := url.Parse(proxy.ProxyURL())
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Transport: &http.Transport{
		Proxy:           http.ProxyURL(proxyURL),
		TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
	}}
}
