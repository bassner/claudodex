package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/convert"
	"github.com/gorilla/websocket"
)

const isolationTimeout = 5 * time.Second

type isolationResult struct {
	status int
	body   string
	err    error
}

type isolationRequest struct {
	thread     string
	session    string
	parent     string
	subagent   string
	connection int32
	request    codex.Request
}

func (r isolationRequest) String() string {
	return fmt.Sprintf("thread=%q session=%q parent=%q subagent=%q connection=%d model=%q cache=%q previous=%q", r.thread, r.session, r.parent, r.subagent, r.connection, r.request.Model, r.request.PromptCacheKey, r.request.PreviousResponseID)
}

func newIsolationProxy(t *testing.T, upstreamHandler http.Handler) (*Server, string, <-chan struct{}) {
	t.Helper()
	home := t.TempDir()
	saveTestAuth(t, home, "fixture-token")
	upstream := httptest.NewServer(upstreamHandler)
	server := New(Config{
		Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true,
		Models: []codex.ModelInfo{inlineModelInfo("gpt-6.1-sol"), inlineModelInfo("gpt-6-astra")},
	})
	mux := http.NewServeMux()
	server.routes(mux)
	finished := make(chan struct{}, 32)
	frontend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
		finished <- struct{}{}
	}))
	t.Cleanup(func() {
		_ = server.Close()
		frontend.Close()
		upstream.Close()
	})
	return server, frontend.URL, finished
}

func isolationBody(model, system string, turn int) string {
	messages := []any{map[string]any{"role": "user", "content": "Read shared.go."}}
	if turn >= 2 {
		messages = append(messages,
			map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "tool_use", "id": "fixture-call", "name": "Read", "input": map[string]any{"file_path": "shared.go"}}}},
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "tool_result", "tool_use_id": "fixture-call", "content": "file contents"}}},
		)
	}
	if turn >= 3 {
		messages = append(messages, map[string]any{"role": "assistant", "content": "worker done"}, map[string]any{"role": "user", "content": "Continue."})
	}
	body, _ := json.Marshal(map[string]any{
		"model": model, "system": system, "stream": true, "messages": messages,
		"tools": []any{map[string]any{"name": "Read", "input_schema": map[string]any{"type": "object", "properties": map[string]any{"file_path": map[string]any{"type": "string"}}}}},
	})
	return string(body)
}

func isolationPost(ctx context.Context, url, body, session, agent string) <-chan isolationResult {
	finished := make(chan isolationResult, 1)
	go func() {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url+"/v1/messages", strings.NewReader(body))
		if err != nil {
			finished <- isolationResult{err: err}
			return
		}
		req.Header.Set("content-type", "application/json")
		if session != "" {
			req.Header.Set("X-Claude-Code-Session-Id", session)
		}
		if agent != "" {
			req.Header.Set("X-Claude-Code-Agent-Id", agent)
		}
		resp, err := (&http.Client{Timeout: isolationTimeout}).Do(req)
		if err != nil {
			finished <- isolationResult{err: err}
			return
		}
		data, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		finished <- isolationResult{status: resp.StatusCode, body: string(data), err: err}
	}()
	return finished
}

func isolationReceive[T any](t *testing.T, values <-chan T) T {
	t.Helper()
	select {
	case value := <-values:
		return value
	case <-time.After(isolationTimeout):
		t.Fatal("timed out waiting for isolation fixture")
		var zero T
		return zero
	}
}

func isolationOK(t *testing.T, result <-chan isolationResult, finished <-chan struct{}) string {
	t.Helper()
	got := isolationReceive(t, result)
	isolationReceive(t, finished)
	if got.err != nil || got.status != http.StatusOK {
		t.Fatalf("proxy response: status=%d err=%v body=%s", got.status, got.err, got.body)
	}
	return got.body
}

func isolationToolEvents(responseID string) []any {
	return []any{
		map[string]any{"type": "response.created", "response": map[string]any{"id": responseID}},
		map[string]any{"type": "response.output_item.done", "item": map[string]any{"type": "function_call", "call_id": "fixture-call", "name": "Read", "arguments": `{"file_path":"shared.go"}`}},
		map[string]any{"type": "response.completed", "response": map[string]any{"id": responseID, "usage": map[string]any{"input_tokens": 7, "output_tokens": 1}}},
	}
}

func isolationTextEvents(responseID string) []any {
	return []any{
		map[string]any{"type": "response.created", "response": map[string]any{"id": responseID}},
		map[string]any{"type": "response.output_item.done", "item": map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": "worker done"}}}},
		map[string]any{"type": "response.completed", "response": map[string]any{"id": responseID, "usage": map[string]any{"input_tokens": 7, "output_tokens": 1}}},
	}
}

func TestMessagesNativeAgentIDsIsolateConcurrentWebSocketWorkers(t *testing.T) {
	t.Setenv("CLAUDODEX_FORCE_CODEX_WEBSOCKET", "1")
	t.Setenv("CLAUDODEX_DISABLE_CODEX_WEBSOCKET", "0")
	const root = "fixture-session"
	const threadA = root + ":agent:worker-a"
	const threadB = root + ":agent:worker-b"
	requests := make(chan isolationRequest, 16)
	startedA, startedB, closedA := make(chan struct{}), make(chan struct{}), make(chan struct{})
	releaseB := make(chan struct{})
	var releaseOnce sync.Once
	var connections atomic.Int32
	upgrader := websocket.Upgrader{}
	server, url, finished := newIsolationProxy(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !websocket.IsWebSocketUpgrade(r) {
			t.Errorf("identified worker or main turn fell back to HTTP: path=%q thread=%q", r.URL.Path, r.Header.Get("thread-id"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		connection := connections.Add(1)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		thread := r.Header.Get("thread-id")
		for turn := 1; ; turn++ {
			_ = conn.SetReadDeadline(time.Now().Add(isolationTimeout))
			var request codex.Request
			if err := conn.ReadJSON(&request); err != nil {
				return
			}
			requests <- isolationRequest{thread: thread, session: r.Header.Get("session-id"), parent: r.Header.Get("x-codex-parent-thread-id"), subagent: r.Header.Get("x-openai-subagent"), connection: connection, request: request}
			if turn == 2 && thread == threadA {
				close(startedA)
				_, _, _ = conn.ReadMessage()
				close(closedA)
				return
			}
			if turn == 2 && thread == threadB {
				close(startedB)
				select {
				case <-releaseB:
				case <-time.After(isolationTimeout):
					return
				}
			}
			events := isolationTextEvents(fmt.Sprintf("%s-response-%d", thread, turn))
			if turn == 1 {
				events = isolationToolEvents(fmt.Sprintf("%s-response-%d", thread, turn))
			}
			for _, event := range events {
				_ = conn.SetWriteDeadline(time.Now().Add(isolationTimeout))
				if err := conn.WriteJSON(event); err != nil {
					t.Error(err)
					return
				}
			}
		}
	}))
	t.Cleanup(func() { releaseOnce.Do(func() { close(releaseB) }) })

	mainSystem := "You are Claude Code, Anthropic's official CLI. Shared reference: /repo/.claude/worktrees/agent-deadbeef."
	workerSystem := "You are an agent for Claude Code, Anthropic's official CLI. Shared reference: /repo/.claude/worktrees/agent-deadbeef."
	mainFirst := isolationPost(context.Background(), url, isolationBody("gpt-6.1-sol", mainSystem, 1), root, "")
	isolationOK(t, mainFirst, finished)
	mainRequest := isolationReceive(t, requests)
	if mainRequest.thread != root || mainRequest.parent != "" || mainRequest.subagent != "" {
		t.Fatalf("main route changed: %+v", mainRequest)
	}

	firstA := isolationPost(context.Background(), url, isolationBody("gpt-6.1-sol", workerSystem, 1), root, "worker-a")
	firstB := isolationPost(context.Background(), url, isolationBody("gpt-6-astra", workerSystem, 1), root, "worker-b")
	isolationOK(t, firstA, finished)
	isolationOK(t, firstB, finished)
	initial := map[string]isolationRequest{}
	for range 2 {
		got := isolationReceive(t, requests)
		initial[got.thread] = got
	}
	for thread, model := range map[string]string{threadA: "gpt-6.1-sol", threadB: "gpt-6-astra"} {
		got, ok := initial[thread]
		if !ok || got.session != thread || got.parent != root || got.subagent != "collab_spawn" || got.request.Model != model || got.request.PromptCacheKey != thread || got.request.PreviousResponseID != "" {
			t.Fatalf("worker route for %s: %+v", thread, got)
		}
	}
	if initial[threadA].connection == initial[threadB].connection || initial[threadA].connection == mainRequest.connection || initial[threadB].connection == mainRequest.connection {
		t.Fatalf("main and workers shared a connection: main=%d a=%d b=%d", mainRequest.connection, initial[threadA].connection, initial[threadB].connection)
	}

	ctxA, cancelA := context.WithCancel(context.Background())
	defer cancelA()
	secondA := isolationPost(ctxA, url, isolationBody("gpt-6.1-sol", workerSystem, 2), root, "worker-a")
	secondB := isolationPost(context.Background(), url, isolationBody("gpt-6-astra", workerSystem, 2), root, "worker-b")
	isolationReceive(t, startedA)
	isolationReceive(t, startedB)
	for range 2 {
		got := isolationReceive(t, requests)
		if got.connection != initial[got.thread].connection || got.request.PreviousResponseID != got.thread+"-response-1" {
			t.Fatalf("worker continuation changed identity: %+v", got)
		}
	}
	cancelA()
	if got := isolationReceive(t, secondA); got.err == nil {
		t.Fatalf("cancelled worker returned success: %+v", got)
	}
	isolationReceive(t, finished)
	isolationReceive(t, closedA)
	if !server.hasWebSocket(threadB) || !server.hasWebSocket(root) {
		t.Fatal("cancelling one worker removed sibling or main connection")
	}
	releaseOnce.Do(func() { close(releaseB) })
	if body := isolationOK(t, secondB, finished); !strings.Contains(body, "worker done") {
		t.Fatalf("sibling did not complete: %s", body)
	}

	isolationOK(t, isolationPost(context.Background(), url, isolationBody("gpt-6-astra", workerSystem, 3), root, "worker-b"), finished)
	thirdB := isolationReceive(t, requests)
	if thirdB.thread != threadB || thirdB.connection != initial[threadB].connection || thirdB.request.PreviousResponseID != threadB+"-response-2" {
		t.Fatalf("sibling follow-up lost its conversation: %+v", thirdB)
	}
	isolationOK(t, isolationPost(context.Background(), url, isolationBody("gpt-6.1-sol", mainSystem, 2), root, ""), finished)
	mainFollowup := isolationReceive(t, requests)
	if mainFollowup.thread != root || mainFollowup.connection != mainRequest.connection || mainFollowup.request.PreviousResponseID != root+"-response-1" || mainFollowup.parent != "" || mainFollowup.subagent != "" {
		t.Fatalf("main follow-up changed: %+v", mainFollowup)
	}
	if got := connections.Load(); got != 3 {
		t.Fatalf("WebSocket handshakes = %d, want main plus two workers", got)
	}
}

func TestMessagesUnidentifiedWorkersAreStatelessAndCleanUp(t *testing.T) {
	for _, identity := range []struct{ name, session, agent string }{
		{name: "headerless", session: "fixture-session"},
		{name: "agent without root session", agent: "worker-a"},
	} {
		t.Run(identity.name, func(t *testing.T) {
			t.Setenv("CLAUDODEX_FORCE_CODEX_WEBSOCKET", "1")
			t.Setenv("CLAUDODEX_DISABLE_CODEX_WEBSOCKET", "0")
			requests := make(chan isolationRequest, 8)
			cancelled := make(chan struct{}, 1)
			var mode atomic.Int32
			var upgrades atomic.Int32
			server, url, finished := newIsolationProxy(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if websocket.IsWebSocketUpgrade(r) {
					upgrades.Add(1)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				var request codex.Request
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				requests <- isolationRequest{thread: r.Header.Get("thread-id"), session: r.Header.Get("session-id"), parent: r.Header.Get("x-codex-parent-thread-id"), subagent: r.Header.Get("x-openai-subagent"), request: request}
				switch mode.Load() {
				case 1:
					w.WriteHeader(http.StatusBadRequest)
					_, _ = io.WriteString(w, `{"error":{"message":"fixture rejection"}}`)
				case 2:
					<-r.Context().Done()
					cancelled <- struct{}{}
				default:
					w.Header().Set("content-type", "text/event-stream")
					for _, event := range isolationTextEvents("fixture-response") {
						data, _ := json.Marshal(event)
						_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
					}
				}
			}))
			preservedChains := map[string]responseChain{"fixture-session": {ResponseID: "main-response"}, "fixture-session:agent:sibling": {ResponseID: "sibling-response"}}
			preservedUsage := map[string]convert.Usage{"fixture-session": {InputTokens: 5}, "fixture-session:agent:sibling": {InputTokens: 6}}
			preservedWS := map[string]*codex.WebSocketConversation{"fixture-session": {}, "fixture-session:agent:sibling": {}}
			for key, value := range preservedChains {
				server.chains[key] = value
			}
			for key, value := range preservedUsage {
				server.usage[key] = value
			}
			for key, value := range preservedWS {
				server.ws[key] = value
			}
			body := isolationBody("gpt-6.1-sol", "You are an agent for Claude Code. Shared reference: /repo/.claude/worktrees/agent-deadbeef.", 1)
			seen := map[string]bool{}
			for _, outcome := range []string{"success", "success", "error", "cancel"} {
				mode.Store(0)
				if outcome == "error" {
					mode.Store(1)
				} else if outcome == "cancel" {
					mode.Store(2)
				}
				ctx, cancel := context.WithCancel(context.Background())
				result := isolationPost(ctx, url, body, identity.session, identity.agent)
				got := isolationReceive(t, requests)
				if got.thread == "" || got.thread != got.session || got.request.PromptCacheKey != got.thread || got.subagent != "collab_spawn" || got.parent != identity.session || got.request.PreviousResponseID != "" {
					t.Fatalf("%s fallback route: %+v", outcome, got)
				}
				if seen[got.thread] {
					t.Fatalf("identical unidentified request reused thread %q", got.thread)
				}
				seen[got.thread] = true
				if outcome == "cancel" {
					cancel()
				}
				response := isolationReceive(t, result)
				isolationReceive(t, finished)
				cancel()
				switch outcome {
				case "success":
					if response.err != nil || response.status != http.StatusOK || !strings.Contains(response.body, "worker done") {
						t.Fatalf("success response: %+v", response)
					}
				case "error":
					if response.err != nil || response.status != http.StatusBadRequest {
						t.Fatalf("error response: %+v", response)
					}
				case "cancel":
					if response.err == nil {
						t.Fatalf("cancel response: %+v", response)
					}
					isolationReceive(t, cancelled)
				}
				server.chainsMu.Lock()
				chainsMatch := reflect.DeepEqual(server.chains, preservedChains)
				usageMatches := reflect.DeepEqual(server.usage, preservedUsage)
				server.chainsMu.Unlock()
				server.wsMu.Lock()
				wsMatch := reflect.DeepEqual(server.ws, preservedWS)
				server.wsMu.Unlock()
				if !chainsMatch || !usageMatches || !wsMatch {
					t.Fatalf("%s retained fallback state or changed parent/sibling: chains=%v usage=%v websockets=%v", outcome, chainsMatch, usageMatches, wsMatch)
				}
			}
			if got := upgrades.Load(); got != 0 {
				t.Fatalf("unidentified workers attempted %d WebSocket upgrades", got)
			}
		})
	}
}

func TestMessagesRejectsInvalidNativeAgentIDBeforeUpstream(t *testing.T) {
	var calls atomic.Int32
	_, url, finished := newIsolationProxy(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	body := isolationBody("gpt-6.1-sol", "You are Claude Code.", 1)
	for _, agent := range []string{"agent/child", "agent child", "agent:child", "agent.123", "agént", strings.Repeat("a", 81)} {
		t.Run(agent, func(t *testing.T) {
			got := isolationReceive(t, isolationPost(context.Background(), url, body, "fixture-session", agent))
			isolationReceive(t, finished)
			if got.err != nil || got.status != http.StatusBadRequest || !strings.Contains(got.body, "invalid_request_error") {
				t.Fatalf("invalid agent response: %+v", got)
			}
		})
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("invalid agent IDs reached upstream %d times", got)
	}
}
