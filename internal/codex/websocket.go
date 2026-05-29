package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var ErrWebSocketBusy = errors.New("websocket conversation is busy")

type wsCreateRequest struct {
	Type string `json:"type"`
	Request
}

type WebSocketConversation struct {
	mu     sync.Mutex
	conn   *websocket.Conn
	header http.Header
}

func (w *WebSocketConversation) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.conn == nil {
		return nil
	}
	err := w.conn.Close()
	w.conn = nil
	w.header = nil
	return err
}

func (w *WebSocketConversation) CreateResponse(ctx context.Context, client Client, request Request, credentials Credentials, route Route, wait bool) (*http.Response, error) {
	if wait {
		w.mu.Lock()
	} else if !w.mu.TryLock() {
		return nil, ErrWebSocketBusy
	}
	conn, header, err := w.connection(ctx, client, credentials, route)
	if err != nil {
		w.mu.Unlock()
		return nil, err
	}

	payload, err := json.Marshal(wsCreateRequest{Type: "response.create", Request: request})
	if err != nil {
		w.mu.Unlock()
		return nil, err
	}
	if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
		w.closeLocked()
		w.mu.Unlock()
		return nil, err
	}

	reader, writer := io.Pipe()
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     header.Clone(),
		Body:       reader,
	}
	resp.Header.Set("content-type", "text/event-stream")

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()
	go w.streamAsSSE(conn, writer, done)
	return resp, nil
}

func (w *WebSocketConversation) connection(ctx context.Context, c Client, credentials Credentials, route Route) (*websocket.Conn, http.Header, error) {
	if w.conn != nil {
		return w.conn, w.header.Clone(), nil
	}
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	wsURL, err := websocketResponseURL(baseURL)
	if err != nil {
		return nil, nil, err
	}
	headers := http.Header{}
	for key, value := range c.headers(credentials, route, true) {
		headers.Set(key, value)
	}
	headers.Del("accept")
	headers.Del("content-type")

	dialer := websocket.DefaultDialer
	conn, handshake, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		if handshake != nil {
			defer handshake.Body.Close()
			body, _ := io.ReadAll(io.LimitReader(handshake.Body, 1<<20))
			return nil, nil, &UpstreamError{
				Status:     handshake.StatusCode,
				Content:    body,
				Header:     handshake.Header.Clone(),
				ContentTyp: handshake.Header.Get("content-type"),
			}
		}
		return nil, nil, err
	}
	w.conn = conn
	w.header = http.Header{}
	if handshake != nil {
		w.header = handshake.Header.Clone()
	}
	return w.conn, w.header.Clone(), nil
}

func (w *WebSocketConversation) closeLocked() {
	if w.conn != nil {
		_ = w.conn.Close()
	}
	w.conn = nil
	w.header = nil
}

func (w *WebSocketConversation) streamAsSSE(conn *websocket.Conn, writer *io.PipeWriter, done chan<- struct{}) {
	defer close(done)
	defer w.mu.Unlock()
	defer writer.Close()
	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			w.closeLocked()
			_ = writer.CloseWithError(err)
			return
		}
		if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
			continue
		}
		event := websocketEventName(data)
		if _, err := fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", event, bytes.TrimSpace(data)); err != nil {
			w.closeLocked()
			return
		}
		if isTerminalWebSocketError(event) {
			w.closeLocked()
			return
		}
		if isTerminalWebSocketEvent(event) {
			return
		}
	}
}

func websocketResponseURL(baseURL string) (string, error) {
	switch {
	case strings.HasPrefix(baseURL, "https://"):
		return "wss://" + strings.TrimPrefix(baseURL, "https://") + "/codex/responses", nil
	case strings.HasPrefix(baseURL, "http://"):
		return "ws://" + strings.TrimPrefix(baseURL, "http://") + "/codex/responses", nil
	case strings.HasPrefix(baseURL, "wss://"), strings.HasPrefix(baseURL, "ws://"):
		return baseURL + "/codex/responses", nil
	default:
		return "", fmt.Errorf("unsupported Codex base URL %q", baseURL)
	}
}

func websocketEventName(data []byte) string {
	var event struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &event); err != nil || strings.TrimSpace(event.Type) == "" {
		return "message"
	}
	return strings.TrimSpace(event.Type)
}

func isTerminalWebSocketEvent(event string) bool {
	switch event {
	case "response.completed", "response.failed", "error":
		return true
	default:
		return false
	}
}

func isTerminalWebSocketError(event string) bool {
	switch event {
	case "response.failed", "error":
		return true
	default:
		return false
	}
}
