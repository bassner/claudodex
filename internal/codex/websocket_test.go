package codex

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketResponseHandshakeCarriesRoutingHintAndRetriesConnectFailures(t *testing.T) {
	upgrader := websocket.Upgrader{}
	requestSeen := make(chan struct{}, 1)
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/codex/accounts/check" {
			_ = json.NewEncoder(w).Encode(map[string]any{"accounts": []map[string]any{{
				"id":                       "account-1",
				"workspace_backend_origin": server.URL,
				"account_routing_override": "NO_CONSTRAINT",
			}}})
			return
		}
		if got := r.Header.Get("x-codex-routing-hint"); got != "model=gpt-5.6-terra;tier=priority" {
			t.Errorf("routing hint = %q", got)
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
		requestSeen <- struct{}{}
		if err := conn.WriteJSON(map[string]any{"type": "response.completed", "response": map[string]any{}}); err != nil {
			t.Errorf("write completion: %v", err)
		}
	}))
	defer server.Close()

	baseDialer := *websocket.DefaultDialer
	baseDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // test server certificate
	realDial := (&net.Dialer{}).DialContext
	var dialAttempts atomic.Int32
	baseDialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if dialAttempts.Add(1) <= 2 {
			return nil, &net.OpError{Op: "dial", Net: network, Err: errors.New("connection unavailable")}
		}
		return realDial(ctx, network, addr)
	}
	client := Client{
		BaseURL:             server.URL,
		ChatGPTBaseURL:      server.URL,
		HTTPClient:          server.Client(),
		WebSocketDialer:     &baseDialer,
		ConnectRetryInitial: time.Millisecond,
		ConnectRetryMax:     time.Millisecond,
	}
	conversation := &WebSocketConversation{}
	resp, err := conversation.CreateResponse(context.Background(), client, Request{
		Model:       "gpt-5.6-terra",
		ServiceTier: "priority",
	}, Credentials{AccountID: "account-1"}, Route{}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatal(err)
	}
	select {
	case <-requestSeen:
	case <-time.After(time.Second):
		t.Fatal("websocket response.create was not observed")
	}
	if got := dialAttempts.Load(); got != 3 {
		t.Fatalf("dial attempts = %d, want two pending connect retries plus one handshake", got)
	}
}

func TestWebSocketResponseReconnectsForTokenAndWorkspaceRoutingChanges(t *testing.T) {
	type handshake struct {
		routing string
		account string
		token   string
	}
	var mu sync.Mutex
	var handshakes []handshake
	upgrader := websocket.Upgrader{}
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/backend-api/wham/accounts/check" {
			override := "us"
			if r.Header.Get("chatgpt-account-id") == "account-2" {
				override = "us_cr"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"accounts": []map[string]any{{
				"id":                       r.Header.Get("chatgpt-account-id"),
				"workspace_backend_origin": server.URL,
				"account_routing_override": override,
			}}})
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()
		mu.Lock()
		handshakes = append(handshakes, handshake{
			routing: r.Header.Get(accountRoutingHeader),
			account: r.Header.Get("chatgpt-account-id"),
			token:   r.Header.Get("authorization"),
		})
		mu.Unlock()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
			if err := conn.WriteJSON(map[string]any{"type": "response.completed", "response": map[string]any{}}); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	dialer := *websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // test server certificate
	client := Client{
		BaseURL:         server.URL + "/backend-api",
		ChatGPTBaseURL:  server.URL,
		HTTPClient:      server.Client(),
		WebSocketDialer: &dialer,
	}
	conversation := &WebSocketConversation{}
	defer conversation.Close()
	for _, credentials := range []Credentials{
		{AccessToken: "token-1", AccountID: "account-1"},
		{AccessToken: "token-1", AccountID: "account-1"},
		{AccessToken: "token-2", AccountID: "account-1"},
		{AccessToken: "token-2", AccountID: "account-2"},
	} {
		resp, err := conversation.CreateResponse(context.Background(), client, Request{Model: "gpt-5.6-sol"}, credentials, Route{}, false)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.ReadAll(resp.Body); err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
	}

	mu.Lock()
	defer mu.Unlock()
	want := []handshake{
		{routing: "us", account: "account-1", token: "Bearer token-1"},
		{routing: "us", account: "account-1", token: "Bearer token-2"},
		{routing: "us_cr", account: "account-2", token: "Bearer token-2"},
	}
	if len(handshakes) != len(want) {
		t.Fatalf("handshakes = %#v, want %#v", handshakes, want)
	}
	for i := range want {
		if handshakes[i] != want[i] {
			t.Fatalf("handshake %d = %#v, want %#v", i, handshakes[i], want[i])
		}
	}
}
