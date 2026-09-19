package codex

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gorilla/websocket"
)

func TestChatGPTRoutingCookiesFlowFromHTTPToWebSocket(t *testing.T) {
	useIsolatedRoutingCookieStore(t)

	upgrader := websocket.Upgrader{}
	var websocketCookie string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !websocket.IsWebSocketUpgrade(r) {
			w.Header().Add("Set-Cookie", "__oailb=west; Path=/backend-api; Secure; HttpOnly")
			w.Header().Add("Set-Cookie", "chatgpt_session=never-store; Path=/; Secure; HttpOnly")
			w.Header().Set("content-type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			return
		}
		if cookie, err := r.Cookie("__oailb"); err == nil {
			websocketCookie = cookie.Value
		}
		if _, err := r.Cookie("chatgpt_session"); !errors.Is(err, http.ErrNoCookie) {
			t.Errorf("WebSocket handshake received excluded session cookie: %v", err)
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()
		if _, _, err := conn.ReadMessage(); err != nil {
			t.Errorf("read response.create: %v", err)
			return
		}
		_ = conn.WriteJSON(map[string]any{"type": "response.completed", "response": map[string]any{}})
	}))
	defer server.Close()

	client := testChatGPTClient(t, server)
	resp, err := client.CreateResponse(context.Background(), Request{Model: "gpt-5.6-terra", Stream: true}, Credentials{}, Route{})
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	conversation := &WebSocketConversation{}
	resp, err = conversation.CreateResponse(context.Background(), client, Request{Model: "gpt-5.6-terra"}, Credentials{}, Route{}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatal(err)
	}
	if websocketCookie != "west" {
		t.Fatalf("WebSocket routing cookie = %q, want west", websocketCookie)
	}
}

func TestChatGPTRoutingCookiesFromRejectedUpgradeReachRetry(t *testing.T) {
	useIsolatedRoutingCookieStore(t)

	upgrader := websocket.Upgrader{}
	var attempts atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := attempts.Add(1)
		if attempt == 1 {
			w.Header().Add("Set-Cookie", "__oailb=east; Path=/backend-api; Secure; HttpOnly")
			w.Header().Add("Set-Cookie", "session=never-store; Path=/; Secure")
			http.Error(w, "retry", http.StatusForbidden)
			return
		}
		cookie, err := r.Cookie("__oailb")
		if err != nil || cookie.Value != "east" {
			t.Errorf("retry routing cookie = %v, %v", cookie, err)
		}
		if _, err := r.Cookie("session"); !errors.Is(err, http.ErrNoCookie) {
			t.Errorf("retry received excluded session cookie: %v", err)
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()
		if _, _, err := conn.ReadMessage(); err != nil {
			t.Errorf("read response.create: %v", err)
			return
		}
		_ = conn.WriteJSON(map[string]any{"type": "response.completed", "response": map[string]any{}})
	}))
	defer server.Close()

	client := testChatGPTClient(t, server)
	conversation := &WebSocketConversation{}
	_, err := conversation.CreateResponse(context.Background(), client, Request{Model: "gpt-5.6-sol"}, Credentials{}, Route{}, false)
	var upstream *UpstreamError
	if !errors.As(err, &upstream) || upstream.Status != http.StatusForbidden {
		t.Fatalf("first handshake error = %v, want HTTP 403", err)
	}

	resp, err := conversation.CreateResponse(context.Background(), client, Request{Model: "gpt-5.6-sol"}, Credentials{}, Route{}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatal(err)
	}
	if attempts.Load() != 2 {
		t.Fatalf("handshake attempts = %d, want 2", attempts.Load())
	}
}

func TestChatGPTRoutingCookiesRejectUnsafeHostsSchemesAndNames(t *testing.T) {
	store := newChatGPTRoutingCookieStore()
	headers := http.Header{}
	headers.Add("Set-Cookie", "__cf_bm=allowed; Path=/; Secure")
	headers.Add("Set-Cookie", "cf_chl_test=challenge; Path=/; Secure")
	headers.Add("Set-Cookie", "chatgpt_session=secret; Path=/; Secure")
	store.storeResponseCookies("https://chatgpt.com/backend-api/codex/responses", headers)

	requestHeaders := http.Header{}
	store.addRequestCookies("wss://chatgpt.com/backend-api/codex/responses", requestHeaders)
	got := requestHeaders.Get("Cookie")
	if !strings.Contains(got, "__cf_bm=allowed") || !strings.Contains(got, "cf_chl_test=challenge") {
		t.Fatalf("allowlisted Cookie header = %q", got)
	}
	if !isAllowedChatGPTInfrastructureCookie("cf_chl_test") {
		t.Fatal("Cloudflare challenge cookie prefix was not allowlisted")
	}
	if strings.Contains(got, "chatgpt_session") {
		t.Fatalf("Cookie header retained excluded session cookie: %q", got)
	}

	explicit := http.Header{"Cookie": []string{"explicit=keep"}}
	store.addRequestCookies("wss://chatgpt.com/backend-api/codex/responses", explicit)
	if got := explicit.Get("Cookie"); got != "explicit=keep" {
		t.Fatalf("explicit Cookie header = %q", got)
	}

	for _, rawURL := range []string{
		"ws://chatgpt.com/backend-api/codex/responses",
		"https://example.com/backend-api/codex/responses",
		"https://notchatgpt.com/backend-api/codex/responses",
	} {
		unsafe := http.Header{}
		store.addRequestCookies(rawURL, unsafe)
		if got := unsafe.Get("Cookie"); got != "" {
			t.Fatalf("unsafe destination %q received Cookie %q", rawURL, got)
		}
	}
}

func useIsolatedRoutingCookieStore(t *testing.T) {
	t.Helper()
	original := sharedChatGPTRoutingCookies
	sharedChatGPTRoutingCookies = newChatGPTRoutingCookieStore()
	t.Cleanup(func() { sharedChatGPTRoutingCookies = original })
}

func testChatGPTClient(t *testing.T, server *httptest.Server) Client {
	t.Helper()
	serverAddress := strings.TrimPrefix(server.URL, "https://")
	dial := (&net.Dialer{}).DialContext
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // test server certificate
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return dial(ctx, network, serverAddress)
		},
	}
	dialer := *websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // test server certificate
	dialer.NetDialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return dial(ctx, network, serverAddress)
	}
	return Client{
		BaseURL:         "https://chatgpt.com/backend-api",
		HTTPClient:      &http.Client{Transport: transport},
		WebSocketDialer: &dialer,
	}
}
