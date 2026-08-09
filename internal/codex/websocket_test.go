package codex

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketResponseHandshakeCarriesRoutingHintAndRetriesConnectFailures(t *testing.T) {
	upgrader := websocket.Upgrader{}
	requestSeen := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
