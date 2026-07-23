package proxy

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bassner/claudodex/internal/auth"
	"github.com/bassner/claudodex/internal/codex"
	"github.com/gorilla/websocket"
)

func TestAnthropicMessageIDIsUniquePerProxyRequest(t *testing.T) {
	first := anthropicMessageID(map[string]any{"request_id": "123-1"})
	second := anthropicMessageID(map[string]any{"request_id": "123-2"})
	if first != "msg_claudodex_123_1" {
		t.Fatalf("first message ID = %q", first)
	}
	if second != "msg_claudodex_123_2" {
		t.Fatalf("second message ID = %q", second)
	}
	if first == second {
		t.Fatal("separate proxy requests must not reuse an Anthropic message ID")
	}
}

func TestResponseChainKeyIsolatesToollessAuxiliaryRequests(t *testing.T) {
	main := codex.Request{Tools: []codex.Tool{{Name: "Bash"}}}
	if got := responseChainKey("thread-1", "request-1", main); got != "thread-1" {
		t.Fatalf("main chain key = %q", got)
	}
	firstAux := responseChainKey("thread-1", "request-1", codex.Request{})
	secondAux := responseChainKey("thread-1", "request-2", codex.Request{})
	if firstAux == "thread-1" || firstAux == secondAux {
		t.Fatalf("auxiliary chain keys were not isolated: %q / %q", firstAux, secondAux)
	}
}

func TestMessagesStreamsCodexResponseAndBuildsUpstreamRequest(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	var captured map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/codex/responses" {
			t.Fatalf("unexpected upstream path %s", r.URL.Path)
		}
		if got := r.Header.Get("authorization"); got != "Bearer access-1" {
			t.Fatalf("authorization = %q", got)
		}
		if got := r.Header.Get("chatgpt-account-id"); got != "acc_123" {
			t.Fatalf("chatgpt-account-id = %q", got)
		}
		if got := r.Header.Get("x-anthropic-billing-header"); got != "" {
			t.Fatalf("forwarded billing header = %q", got)
		}
		if got := r.Header.Get("session-id"); got != "session-123" {
			t.Fatalf("session-id = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("content-type", "text/event-stream")
		w.Header().Set("x-codex-primary-used-percent", "42")
		w.Header().Set("x-codex-primary-window-minutes", "300")
		w.Header().Set("x-codex-primary-reset-at", "1770000000")
		w.Header().Set("x-codex-secondary-used-percent", "17")
		w.Header().Set("x-codex-secondary-window-minutes", "10080")
		w.Header().Set("x-codex-secondary-reset-at", "1770500000")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: response.created`,
			`data: {"type":"response.created","response":{"id":"resp_1"}}`,
			``,
			`event: response.output_item.added`,
			`data: {"type":"response.output_item.added","item":{"type":"message","id":"item_1"}}`,
			``,
			`event: response.output_text.delta`,
			`data: {"type":"response.output_text.delta","delta":"ok"}`,
			``,
			`event: response.output_item.done`,
			`data: {"type":"response.output_item.done","item":{"type":"message","id":"item_1","content":[{"type":"output_text","text":"ok"}]}}`,
			``,
			`event: response.completed`,
			`data: {"type":"response.completed","response":{"usage":{"input_tokens":4,"output_tokens":1}}}`,
			``,
			``,
		}, "\n")))
	}))
	defer upstream.Close()

	server := New(Config{Version: "test", Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	body := `{"model":"claude-sonnet-4-6","system":"hello\nx-anthropic-billing-header: secret\nworld","output_config":{"effort":"max"},"stream":true,"messages":[{"role":"user","content":"say ok"}]}`
	req, err := http.NewRequest(http.MethodPost, "http://"+addr+"/v1/messages?beta=true", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-claude-code-session-id", "session-123")
	req.Header.Set("x-anthropic-billing-header", "must-not-forward")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if got := resp.Header.Get("anthropic-ratelimit-unified-5h-utilization"); got != "0.42" {
		t.Fatalf("5h utilization header = %q", got)
	}
	if got := resp.Header.Get("anthropic-ratelimit-unified-7d-utilization"); got != "0.17" {
		t.Fatalf("7d utilization header = %q", got)
	}
	sse := readAllString(t, resp)
	if !strings.Contains(sse, "event: message_start") || !strings.Contains(sse, `"model":"claude-sonnet-4-6"`) {
		t.Fatalf("missing message_start/model in SSE:\n%s", sse)
	}
	if !strings.Contains(sse, `"text":"ok"`) || !strings.Contains(sse, "event: message_stop") {
		t.Fatalf("missing text/stop in SSE:\n%s", sse)
	}
	if captured["model"] != "gpt-5.6-terra" {
		t.Fatalf("upstream model = %#v", captured["model"])
	}
	instructions, _ := captured["instructions"].(string)
	if !strings.HasPrefix(instructions, "hello\nworld\n\nClaude Code compatibility:\n") {
		t.Fatalf("instructions = %#v", captured["instructions"])
	}
	if strings.Contains(instructions, "x-anthropic-billing-header") || strings.Contains(instructions, "must-not-forward") {
		t.Fatalf("billing header leaked into instructions: %q", instructions)
	}
	reasoning := captured["reasoning"].(map[string]any)
	if reasoning["effort"] != "max" {
		t.Fatalf("reasoning = %#v", reasoning)
	}
	if captured["stream"] != true || captured["store"] != false || captured["parallel_tool_calls"] != true {
		t.Fatalf("upstream stream/store/parallel = %#v", captured)
	}
}

func TestMessagesHTTPToolContinuationReplaysEncryptedReasoningLosslessly(t *testing.T) {
	t.Setenv("CLAUDODEX_DISABLE_CODEX_WEBSOCKET", "1")
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	var requests atomic.Int32
	var secondRequest map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestNumber := requests.Add(1)
		var captured map[string]any
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("content-type", "text/event-stream")
		if requestNumber == 1 {
			_, _ = w.Write([]byte(strings.Join([]string{
				`event: response.created`,
				`data: {"type":"response.created","response":{"id":"resp_state_1"}}`,
				``,
				`event: response.output_item.done`,
				`data: {"type":"response.output_item.done","output_index":0,"item":{"id":"rs_1","type":"reasoning","summary":[{"type":"summary_text","text":"**Inspecting state**"}],"encrypted_content":"opaque-reasoning-state","provider_extension":{"keep":true}}}`,
				``,
				`event: response.output_item.done`,
				`data: {"type":"response.output_item.done","output_index":1,"item":{"id":"msg_1","type":"message","role":"assistant","phase":"commentary","content":[{"type":"output_text","text":"Checking."}]}}`,
				``,
				`event: response.output_item.done`,
				`data: {"type":"response.output_item.done","output_index":2,"item":{"id":"fc_1","type":"function_call","call_id":"call_1","name":"Read","arguments":"{\"file_path\":\"a.go\"}"}}`,
				``,
				`event: response.completed`,
				`data: {"type":"response.completed","response":{"id":"resp_state_1","usage":{"input_tokens":10,"output_tokens":2}}}`,
				``,
				``,
			}, "\n")))
			return
		}
		secondRequest = captured
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: response.created`,
			`data: {"type":"response.created","response":{"id":"resp_state_2"}}`,
			``,
			`event: response.output_item.done`,
			`data: {"type":"response.output_item.done","output_index":0,"item":{"id":"msg_2","type":"message","role":"assistant","phase":"final_answer","content":[{"type":"output_text","text":"done"}]}}`,
			``,
			`event: response.completed`,
			`data: {"type":"response.completed","response":{"id":"resp_state_2","usage":{"input_tokens":12,"output_tokens":1}}}`,
			``,
			``,
		}, "\n")))
	}))
	defer upstream.Close()

	server := New(Config{
		Home:         home,
		CodexBaseURL: upstream.URL,
		HTTPClient:   upstream.Client(),
		AuthPresent:  true,
		Models:       []codex.ModelInfo{{Slug: "gpt-5.6-terra", SupportsReasoningSummaries: true}},
	})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	first := `{"model":"claude-sonnet-4-6","thinking":{"type":"adaptive"},"stream":true,"messages":[{"role":"user","content":"read a.go"}],"tools":[{"name":"Read","input_schema":{"type":"object","properties":{"file_path":{"type":"string"}}}}]}`
	request, err := http.NewRequest(http.MethodPost, "http://"+addr+"/v1/messages", strings.NewReader(first))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("content-type", "application/json")
	request.Header.Set("x-claude-code-session-id", "state-session")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	firstSSE := readAllString(t, response)
	_ = response.Body.Close()
	if got := thinkingTextFromSSE(t, firstSSE); got != "**Inspecting state**" {
		t.Fatalf("first response thinking = %q, want %q:\n%s", got, "**Inspecting state**", firstSSE)
	}

	second := `{"model":"claude-sonnet-4-6","thinking":{"type":"adaptive"},"stream":true,"messages":[{"role":"user","content":"read a.go"},{"role":"assistant","content":[{"type":"thinking","thinking":"**Inspecting state**","signature":"claudodex_openai_reasoning_summary"},{"type":"text","text":"Checking."},{"type":"tool_use","id":"call_1","name":"Read","input":{"file_path":"a.go"}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"file contents"}]}],"tools":[{"name":"Read","input_schema":{"type":"object","properties":{"file_path":{"type":"string"}}}}]}`
	request, err = http.NewRequest(http.MethodPost, "http://"+addr+"/v1/messages", strings.NewReader(second))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("content-type", "application/json")
	request.Header.Set("x-claude-code-session-id", "state-session")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	secondSSE := readAllString(t, response)
	_ = response.Body.Close()
	if !strings.Contains(secondSSE, `"text":"done"`) {
		t.Fatalf("second response missing final text:\n%s", secondSSE)
	}
	if requests.Load() != 2 {
		t.Fatalf("upstream requests = %d, want 2", requests.Load())
	}
	if secondRequest["previous_response_id"] != nil {
		t.Fatalf("stateless HTTP replay unexpectedly used previous_response_id: %#v", secondRequest)
	}
	input, _ := secondRequest["input"].([]any)
	if len(input) != 5 {
		t.Fatalf("replayed input = %#v", input)
	}
	for index, wantType := range []string{"message", "reasoning", "message", "function_call", "function_call_output"} {
		item, _ := input[index].(map[string]any)
		if item["type"] != wantType {
			t.Fatalf("input[%d] = %#v, want type %q", index, item, wantType)
		}
	}
	reasoning := input[1].(map[string]any)
	if reasoning["encrypted_content"] != "opaque-reasoning-state" {
		t.Fatalf("encrypted reasoning was not replayed: %#v", reasoning)
	}
	if reasoning["provider_extension"].(map[string]any)["keep"] != true {
		t.Fatalf("unknown reasoning fields were not replayed: %#v", reasoning)
	}
	commentary := input[2].(map[string]any)
	if commentary["phase"] != "commentary" {
		t.Fatalf("assistant phase was not replayed: %#v", commentary)
	}
	encoded, err := json.Marshal(secondRequest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "claudodex_openai_reasoning_summary") {
		t.Fatalf("synthetic Claude thinking leaked upstream: %s", encoded)
	}
}

func TestMessagesRoutesClaudeCodeSubagentAsCodexChildThread(t *testing.T) {
	t.Setenv("CLAUDODEX_DISABLE_CODEX_WEBSOCKET", "1")
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	var captured map[string]any
	var gotSessionID string
	var gotThreadID string
	var gotParentThreadID string
	var gotSubagent string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSessionID = r.Header.Get("session-id")
		gotThreadID = r.Header.Get("thread-id")
		gotParentThreadID = r.Header.Get("x-codex-parent-thread-id")
		gotSubagent = r.Header.Get("x-openai-subagent")
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte("event: response.completed\n"))
		_, _ = w.Write([]byte(`data: {"type":"response.completed","response":{"usage":{"input_tokens":1,"output_tokens":0}}}` + "\n\n"))
	}))
	defer upstream.Close()

	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	body := `{"model":"claude-opus-4-6","system":"You are an agent for Claude Code, Anthropic's official CLI for Claude. Complete the delegated task.","messages":[{"role":"user","content":"Read README.md."}]}`
	req, err := http.NewRequest(http.MethodPost, "http://"+addr+"/v1/messages", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-claude-code-session-id", "parent-123")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if !strings.HasPrefix(gotSessionID, "parent-123:agent:") || gotThreadID != gotSessionID {
		t.Fatalf("session/thread headers = %q / %q", gotSessionID, gotThreadID)
	}
	if gotParentThreadID != "parent-123" || gotSubagent != "collab_spawn" {
		t.Fatalf("subagent headers = parent %q subagent %q", gotParentThreadID, gotSubagent)
	}
	if captured["prompt_cache_key"] != gotSessionID {
		t.Fatalf("prompt_cache_key = %#v, want %q", captured["prompt_cache_key"], gotSessionID)
	}
}

func TestMessagesUsesWebSocketPreviousResponseForMainTurnSteering(t *testing.T) {
	t.Setenv("CLAUDODEX_FORCE_CODEX_WEBSOCKET", "1")
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	var wsHandshakes atomic.Int32
	var firstWSRequest map[string]any
	var secondWSRequest map[string]any
	upgrader := websocket.Upgrader{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !websocket.IsWebSocketUpgrade(r) {
			t.Fatalf("expected websocket upgrade, got %s", r.Header.Get("upgrade"))
		}
		wsHandshakes.Add(1)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		if err := conn.ReadJSON(&firstWSRequest); err != nil {
			t.Fatal(err)
		}
		writeWSJSON(t, conn, map[string]any{"type": "response.created", "response": map[string]any{"id": "resp-1"}})
		writeWSJSON(t, conn, map[string]any{
			"type": "response.output_item.done",
			"item": map[string]any{"type": "function_call", "call_id": "call_1", "name": "Read", "arguments": "{\"file_path\":\"a.go\"}"},
		})
		writeWSJSON(t, conn, map[string]any{
			"type":     "response.completed",
			"response": map[string]any{"id": "resp-1", "stop_reason": "tool_calls", "usage": map[string]any{"input_tokens": 10, "output_tokens": 1}},
		})
		if err := conn.ReadJSON(&secondWSRequest); err != nil {
			t.Fatal(err)
		}
		writeWSJSON(t, conn, map[string]any{"type": "response.created", "response": map[string]any{"id": "resp-2"}})
		writeWSJSON(t, conn, map[string]any{
			"type": "response.output_item.done",
			"item": map[string]any{
				"type":    "message",
				"role":    "assistant",
				"content": []any{map[string]any{"type": "output_text", "text": "done"}},
			},
		})
		writeWSJSON(t, conn, map[string]any{
			"type":     "response.completed",
			"response": map[string]any{"id": "resp-2", "usage": map[string]any{"input_tokens": 1, "output_tokens": 1}},
		})
	}))
	defer upstream.Close()

	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	system := `"system":"You are Claude Code, Anthropic's official CLI.",`
	first := `{"model":"claude-opus-4-6",` + system + `"stream":true,"messages":[{"role":"user","content":"read a.go"}],"tools":[{"name":"Read","input_schema":{"type":"object","properties":{"file_path":{"type":"string"}}}}]}`
	req, err := http.NewRequest(http.MethodPost, "http://"+addr+"/v1/messages", strings.NewReader(first))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-claude-code-session-id", "resume-session")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = readAllString(t, resp)
	_ = resp.Body.Close()

	second := `{"model":"claude-opus-4-6",` + system + `"stream":true,"messages":[{"role":"user","content":"read a.go"},{"role":"assistant","content":[{"type":"tool_use","id":"call_1","name":"Read","input":{"file_path":"a.go"}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"file contents"},{"type":"text","text":"Stop and answer me now."}]}],"tools":[{"name":"Read","input_schema":{"type":"object","properties":{"file_path":{"type":"string"}}}}]}`
	req, err = http.NewRequest(http.MethodPost, "http://"+addr+"/v1/messages", strings.NewReader(second))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-claude-code-session-id", "resume-session")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	sse := readAllString(t, resp)
	_ = resp.Body.Close()
	if !strings.Contains(sse, `"text":"done"`) {
		t.Fatalf("missing websocket response text:\n%s", sse)
	}
	if wsHandshakes.Load() != 1 {
		t.Fatalf("websocket handshakes = %d, want one persistent connection", wsHandshakes.Load())
	}
	if firstWSRequest["previous_response_id"] != nil {
		t.Fatalf("first previous_response_id = %#v, request %#v", firstWSRequest["previous_response_id"], firstWSRequest)
	}
	if secondWSRequest["previous_response_id"] != "resp-1" {
		t.Fatalf("previous_response_id = %#v, request %#v", secondWSRequest["previous_response_id"], secondWSRequest)
	}
	input, ok := secondWSRequest["input"].([]any)
	if !ok || len(input) != 2 {
		t.Fatalf("websocket input = %#v, want tool output plus steering", secondWSRequest["input"])
	}
	item, _ := input[0].(map[string]any)
	if item["type"] != "function_call_output" || item["call_id"] != "call_1" {
		t.Fatalf("incremental input item = %#v", item)
	}
	steering, _ := input[1].(map[string]any)
	if steering["type"] != "message" || steering["role"] != "user" {
		t.Fatalf("incremental steering item = %#v", steering)
	}
	content, _ := steering["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("incremental steering content = %#v", steering["content"])
	}
	text, _ := content[0].(map[string]any)
	if text["type"] != "input_text" || text["text"] != "Stop and answer me now." {
		t.Fatalf("incremental steering text = %#v", text)
	}
	usage := messageDeltaUsage(t, sse)
	fullRequestEstimate := estimateTokenCountFromBytes([]byte(second), false)
	gotInput := usage.InputTokens + usage.CacheCreationInputTokens + usage.CacheReadInputTokens
	if gotInput < fullRequestEstimate {
		t.Fatalf("reported resumed input = %d, want at least full request estimate %d", gotInput, fullRequestEstimate)
	}
}

func TestMessagesRetriesFullRequestWhenWebSocketPreviousResponseIsMissing(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	var wsHandshakes atomic.Int32
	var secondWSRequest map[string]any
	var retryWSRequest map[string]any
	upgrader := websocket.Upgrader{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !websocket.IsWebSocketUpgrade(r) {
			t.Fatalf("expected websocket upgrade, got %s", r.Header.Get("upgrade"))
		}
		handshake := wsHandshakes.Add(1)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		switch handshake {
		case 1:
			var firstWSRequest map[string]any
			if err := conn.ReadJSON(&firstWSRequest); err != nil {
				t.Fatal(err)
			}
			writeWSJSON(t, conn, map[string]any{"type": "response.created", "response": map[string]any{"id": "resp-1"}})
			writeWSJSON(t, conn, map[string]any{
				"type": "response.output_item.done",
				"item": map[string]any{"type": "function_call", "call_id": "call_1", "name": "Read", "arguments": "{\"file_path\":\"a.go\"}"},
			})
			writeWSJSON(t, conn, map[string]any{
				"type":     "response.completed",
				"response": map[string]any{"id": "resp-1", "stop_reason": "tool_calls", "usage": map[string]any{"input_tokens": 10, "output_tokens": 1}},
			})
			if err := conn.ReadJSON(&secondWSRequest); err != nil {
				t.Fatal(err)
			}
			writeWSJSON(t, conn, map[string]any{
				"type":  "error",
				"error": map[string]any{"type": "server_error", "message": "Previous response with id 'resp-1' not found"},
			})
		case 2:
			if err := conn.ReadJSON(&retryWSRequest); err != nil {
				t.Fatal(err)
			}
			writeWSJSON(t, conn, map[string]any{"type": "response.created", "response": map[string]any{"id": "resp-2"}})
			writeWSJSON(t, conn, map[string]any{
				"type": "response.output_item.done",
				"item": map[string]any{
					"type":    "message",
					"role":    "assistant",
					"content": []any{map[string]any{"type": "output_text", "text": "recovered"}},
				},
			})
			writeWSJSON(t, conn, map[string]any{
				"type":     "response.completed",
				"response": map[string]any{"id": "resp-2", "usage": map[string]any{"input_tokens": 20, "output_tokens": 1}},
			})
		default:
			t.Fatalf("unexpected websocket handshake %d", handshake)
		}
	}))
	defer upstream.Close()

	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	system := `"system":"You are an agent for Claude Code, Anthropic's official CLI for Claude. Complete the delegated task.",`
	first := `{"model":"claude-opus-4-6",` + system + `"stream":true,"messages":[{"role":"user","content":"read a.go"}],"tools":[{"name":"Read","input_schema":{"type":"object","properties":{"file_path":{"type":"string"}}}}]}`
	req, err := http.NewRequest(http.MethodPost, "http://"+addr+"/v1/messages", strings.NewReader(first))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-claude-code-session-id", "recover-session")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = readAllString(t, resp)
	_ = resp.Body.Close()

	second := `{"model":"claude-opus-4-6",` + system + `"stream":true,"messages":[{"role":"user","content":"read a.go"},{"role":"assistant","content":[{"type":"tool_use","id":"call_1","name":"Read","input":{"file_path":"a.go"}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"file contents"}]}],"tools":[{"name":"Read","input_schema":{"type":"object","properties":{"file_path":{"type":"string"}}}}]}`
	req, err = http.NewRequest(http.MethodPost, "http://"+addr+"/v1/messages", strings.NewReader(second))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-claude-code-session-id", "recover-session")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	sse := readAllString(t, resp)
	_ = resp.Body.Close()
	if !strings.Contains(sse, `"text":"recovered"`) {
		t.Fatalf("missing recovered websocket response text:\n%s", sse)
	}
	if wsHandshakes.Load() != 2 {
		t.Fatalf("websocket handshakes = %d, want retry on a fresh connection", wsHandshakes.Load())
	}
	if secondWSRequest["previous_response_id"] != "resp-1" {
		t.Fatalf("previous_response_id = %#v, request %#v", secondWSRequest["previous_response_id"], secondWSRequest)
	}
	if retryWSRequest["previous_response_id"] != nil {
		t.Fatalf("retry previous_response_id = %#v, request %#v", retryWSRequest["previous_response_id"], retryWSRequest)
	}
	input, ok := retryWSRequest["input"].([]any)
	if !ok || len(input) <= 1 {
		t.Fatalf("retry input = %#v, want full replay", retryWSRequest["input"])
	}
}

func TestMessagesRetriesFullRequestWhenWebSocketPreviousResponseErrorArrivesAfterCreated(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	var wsHandshakes atomic.Int32
	var secondWSRequest map[string]any
	var retryWSRequest map[string]any
	upgrader := websocket.Upgrader{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !websocket.IsWebSocketUpgrade(r) {
			t.Fatalf("expected websocket upgrade, got %s", r.Header.Get("upgrade"))
		}
		handshake := wsHandshakes.Add(1)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		switch handshake {
		case 1:
			var firstWSRequest map[string]any
			if err := conn.ReadJSON(&firstWSRequest); err != nil {
				t.Fatal(err)
			}
			writeWSJSON(t, conn, map[string]any{"type": "response.created", "response": map[string]any{"id": "resp-1"}})
			writeWSJSON(t, conn, map[string]any{
				"type": "response.output_item.done",
				"item": map[string]any{"type": "function_call", "call_id": "call_1", "name": "Read", "arguments": "{\"file_path\":\"a.go\"}"},
			})
			writeWSJSON(t, conn, map[string]any{
				"type":     "response.completed",
				"response": map[string]any{"id": "resp-1", "stop_reason": "tool_calls", "usage": map[string]any{"input_tokens": 10, "output_tokens": 1}},
			})
			if err := conn.ReadJSON(&secondWSRequest); err != nil {
				t.Fatal(err)
			}
			writeWSJSON(t, conn, map[string]any{"type": "response.created", "response": map[string]any{"id": "resp-poison"}})
			writeWSJSON(t, conn, map[string]any{
				"type":  "error",
				"error": map[string]any{"type": "server_error", "message": "Previous response with id 'resp-1' not found"},
			})
		case 2:
			if err := conn.ReadJSON(&retryWSRequest); err != nil {
				t.Fatal(err)
			}
			writeWSJSON(t, conn, map[string]any{"type": "response.created", "response": map[string]any{"id": "resp-2"}})
			writeWSJSON(t, conn, map[string]any{
				"type": "response.output_item.done",
				"item": map[string]any{
					"type":    "message",
					"role":    "assistant",
					"content": []any{map[string]any{"type": "output_text", "text": "recovered after created"}},
				},
			})
			writeWSJSON(t, conn, map[string]any{
				"type":     "response.completed",
				"response": map[string]any{"id": "resp-2", "usage": map[string]any{"input_tokens": 20, "output_tokens": 1}},
			})
		default:
			t.Fatalf("unexpected websocket handshake %d", handshake)
		}
	}))
	defer upstream.Close()

	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	system := `"system":"You are an agent for Claude Code, Anthropic's official CLI for Claude. Complete the delegated task.",`
	first := `{"model":"claude-opus-4-6",` + system + `"stream":true,"messages":[{"role":"user","content":"read a.go"}],"tools":[{"name":"Read","input_schema":{"type":"object","properties":{"file_path":{"type":"string"}}}}]}`
	req, err := http.NewRequest(http.MethodPost, "http://"+addr+"/v1/messages", strings.NewReader(first))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-claude-code-session-id", "created-error-session")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = readAllString(t, resp)
	_ = resp.Body.Close()

	second := `{"model":"claude-opus-4-6",` + system + `"stream":true,"messages":[{"role":"user","content":"read a.go"},{"role":"assistant","content":[{"type":"tool_use","id":"call_1","name":"Read","input":{"file_path":"a.go"}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"file contents"}]}],"tools":[{"name":"Read","input_schema":{"type":"object","properties":{"file_path":{"type":"string"}}}}]}`
	req, err = http.NewRequest(http.MethodPost, "http://"+addr+"/v1/messages", strings.NewReader(second))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-claude-code-session-id", "created-error-session")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	sse := readAllString(t, resp)
	_ = resp.Body.Close()
	if !strings.Contains(sse, `"text":"recovered after created"`) {
		t.Fatalf("missing recovered websocket response text:\n%s", sse)
	}
	if wsHandshakes.Load() != 2 {
		t.Fatalf("websocket handshakes = %d, want retry on a fresh connection", wsHandshakes.Load())
	}
	if secondWSRequest["previous_response_id"] != "resp-1" {
		t.Fatalf("previous_response_id = %#v, request %#v", secondWSRequest["previous_response_id"], secondWSRequest)
	}
	if retryWSRequest["previous_response_id"] != nil {
		t.Fatalf("retry previous_response_id = %#v, request %#v", retryWSRequest["previous_response_id"], retryWSRequest)
	}
	input, ok := retryWSRequest["input"].([]any)
	if !ok || len(input) <= 1 {
		t.Fatalf("retry input = %#v, want full replay", retryWSRequest["input"])
	}
}

func TestMessagesFullSubagentRequestFallsBackToHTTPWhenWebSocketIsBusy(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	wsReady := make(chan struct{})
	releaseWS := make(chan struct{})
	firstDone := make(chan error, 1)
	var httpRequests atomic.Int32
	upgrader := websocket.Upgrader{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if websocket.IsWebSocketUpgrade(r) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			var firstWSRequest map[string]any
			if err := conn.ReadJSON(&firstWSRequest); err != nil {
				t.Fatal(err)
			}
			close(wsReady)
			<-releaseWS
			writeWSJSON(t, conn, map[string]any{"type": "response.created", "response": map[string]any{"id": "resp-1"}})
			writeWSJSON(t, conn, map[string]any{
				"type":     "response.completed",
				"response": map[string]any{"id": "resp-1", "usage": map[string]any{"input_tokens": 1, "output_tokens": 0}},
			})
			return
		}

		httpRequests.Add(1)
		var captured map[string]any
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		if captured["previous_response_id"] != nil {
			t.Fatalf("HTTP fallback previous_response_id = %#v", captured["previous_response_id"])
		}
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: response.output_item.done`,
			`data: {"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"http fallback"}]}}`,
			``,
			`event: response.completed`,
			`data: {"type":"response.completed","response":{"id":"resp-http","usage":{"input_tokens":1,"output_tokens":1}}}`,
			``,
			``,
		}, "\n")))
	}))
	defer upstream.Close()

	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	system := `"system":"You are an agent for Claude Code, Anthropic's official CLI for Claude. cwd: /repo/.claude/worktrees/agent-busy12345",`
	first := `{"model":"claude-opus-4-6",` + system + `"stream":true,"messages":[{"role":"user","content":"first"}]}`
	go func() {
		req, err := http.NewRequest(http.MethodPost, "http://"+addr+"/v1/messages", strings.NewReader(first))
		if err != nil {
			firstDone <- err
			return
		}
		req.Header.Set("content-type", "application/json")
		req.Header.Set("x-claude-code-session-id", "busy-session")
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_, _ = io.ReadAll(resp.Body)
			_ = resp.Body.Close()
		}
		firstDone <- err
	}()

	select {
	case <-wsReady:
	case <-time.After(2 * time.Second):
		t.Fatal("first websocket request did not start")
	}

	second := `{"model":"claude-opus-4-6",` + system + `"stream":false,"messages":[{"role":"user","content":"second"}]}`
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "http://"+addr+"/v1/messages", strings.NewReader(second))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-claude-code-session-id", "busy-session")
	resp, err := client.Do(req)
	if err != nil {
		close(releaseWS)
		t.Fatal(err)
	}
	body := readAllString(t, resp)
	_ = resp.Body.Close()
	close(releaseWS)
	if !strings.Contains(body, "http fallback") {
		t.Fatalf("missing HTTP fallback response:\n%s", body)
	}
	if httpRequests.Load() != 1 {
		t.Fatalf("HTTP requests = %d, want one fallback request", httpRequests.Load())
	}
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
}

func TestMessagesRetriesFullRequestWhenWebSocketContinuationClosesBeforeFirstEvent(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	var wsHandshakes atomic.Int32
	var secondWSRequest map[string]any
	var retryWSRequest map[string]any
	upgrader := websocket.Upgrader{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !websocket.IsWebSocketUpgrade(r) {
			t.Fatalf("expected websocket upgrade, got %s", r.Header.Get("upgrade"))
		}
		handshake := wsHandshakes.Add(1)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		switch handshake {
		case 1:
			var firstWSRequest map[string]any
			if err := conn.ReadJSON(&firstWSRequest); err != nil {
				t.Fatal(err)
			}
			writeWSJSON(t, conn, map[string]any{"type": "response.created", "response": map[string]any{"id": "resp-1"}})
			writeWSJSON(t, conn, map[string]any{
				"type": "response.output_item.done",
				"item": map[string]any{"type": "function_call", "call_id": "call_1", "name": "Read", "arguments": "{\"file_path\":\"a.go\"}"},
			})
			writeWSJSON(t, conn, map[string]any{
				"type":     "response.completed",
				"response": map[string]any{"id": "resp-1", "stop_reason": "tool_calls", "usage": map[string]any{"input_tokens": 10, "output_tokens": 1}},
			})
			if err := conn.ReadJSON(&secondWSRequest); err != nil {
				t.Fatal(err)
			}
			return
		case 2:
			if err := conn.ReadJSON(&retryWSRequest); err != nil {
				t.Fatal(err)
			}
			writeWSJSON(t, conn, map[string]any{"type": "response.created", "response": map[string]any{"id": "resp-2"}})
			writeWSJSON(t, conn, map[string]any{
				"type": "response.output_item.done",
				"item": map[string]any{
					"type":    "message",
					"role":    "assistant",
					"content": []any{map[string]any{"type": "output_text", "text": "recovered"}},
				},
			})
			writeWSJSON(t, conn, map[string]any{
				"type":     "response.completed",
				"response": map[string]any{"id": "resp-2", "usage": map[string]any{"input_tokens": 20, "output_tokens": 1}},
			})
		default:
			t.Fatalf("unexpected websocket handshake %d", handshake)
		}
	}))
	defer upstream.Close()

	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	system := `"system":"You are an agent for Claude Code, Anthropic's official CLI for Claude. Complete the delegated task.",`
	first := `{"model":"claude-opus-4-6",` + system + `"stream":true,"messages":[{"role":"user","content":"read a.go"}],"tools":[{"name":"Read","input_schema":{"type":"object","properties":{"file_path":{"type":"string"}}}}]}`
	req, err := http.NewRequest(http.MethodPost, "http://"+addr+"/v1/messages", strings.NewReader(first))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-claude-code-session-id", "closed-session")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = readAllString(t, resp)
	_ = resp.Body.Close()

	second := `{"model":"claude-opus-4-6",` + system + `"stream":true,"messages":[{"role":"user","content":"read a.go"},{"role":"assistant","content":[{"type":"tool_use","id":"call_1","name":"Read","input":{"file_path":"a.go"}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"file contents"}]}],"tools":[{"name":"Read","input_schema":{"type":"object","properties":{"file_path":{"type":"string"}}}}]}`
	req, err = http.NewRequest(http.MethodPost, "http://"+addr+"/v1/messages", strings.NewReader(second))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-claude-code-session-id", "closed-session")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	sse := readAllString(t, resp)
	_ = resp.Body.Close()
	if !strings.Contains(sse, `"text":"recovered"`) {
		t.Fatalf("missing recovered websocket response text:\n%s", sse)
	}
	if wsHandshakes.Load() != 2 {
		t.Fatalf("websocket handshakes = %d, want retry on a fresh connection", wsHandshakes.Load())
	}
	if secondWSRequest["previous_response_id"] != "resp-1" {
		t.Fatalf("previous_response_id = %#v, request %#v", secondWSRequest["previous_response_id"], secondWSRequest)
	}
	if retryWSRequest["previous_response_id"] != nil {
		t.Fatalf("retry previous_response_id = %#v, request %#v", retryWSRequest["previous_response_id"], retryWSRequest)
	}
	input, ok := retryWSRequest["input"].([]any)
	if !ok || len(input) <= 1 {
		t.Fatalf("retry input = %#v, want full replay", retryWSRequest["input"])
	}
}

func TestMessagesNonStreamingAssemblesMessage(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte("event: response.output_item.done\n"))
		_, _ = w.Write([]byte(`data: {"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"done"}]}}` + "\n\n"))
		_, _ = w.Write([]byte("event: response.completed\n"))
		_, _ = w.Write([]byte(`data: {"type":"response.completed","response":{"stop_reason":"stop","usage":{"input_tokens":1,"output_tokens":1}}}` + "\n\n"))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if got := resp.Header.Get("content-type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("content-type = %q, want JSON for omitted stream flag", got)
	}
	var body struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Content) != 1 || body.Content[0].Text != "done" || body.StopReason != "end_turn" {
		t.Fatalf("body = %#v", body)
	}
}

func TestMessagesStreamingMalformedUpstreamSSEEmitsAnthropicErrorEvent(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte("event: response.output_text.delta\n"))
		_, _ = w.Write([]byte("data: {not-json}\n\n"))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","stream":true,"messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	sse := readAllString(t, resp)
	if !strings.Contains(sse, "event: error") || !strings.Contains(sse, "malformed SSE") {
		t.Fatalf("missing Anthropic error event in SSE:\n%s", sse)
	}
}

func TestMessagesWritesMetadataTraceWithIdleMarker(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	tracePath := home + "/trace.jsonl"
	t.Setenv("CLAUDODEX_PROXY_TRACE", tracePath)
	t.Setenv("CLAUDODEX_PROXY_TRACE_IDLE_INTERVAL", "10ms")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		time.Sleep(35 * time.Millisecond)
		_, _ = w.Write([]byte("event: response.created\n"))
		_, _ = w.Write([]byte(`data: {"type":"response.created","response":{"id":"resp_trace"}}` + "\n\n"))
		_, _ = w.Write([]byte("event: response.completed\n"))
		_, _ = w.Write([]byte(`data: {"type":"response.completed","response":{"id":"resp_trace","usage":{"input_tokens":1,"output_tokens":0}}}` + "\n\n"))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","stream":true,"messages":[{"role":"user","content":"trace me"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	_ = readAllString(t, resp)
	traceData, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	trace := string(traceData)
	for _, want := range []string{`"event":"messages.request"`, `"event":"upstream.waiting_headers"`, `"event":"upstream.opened"`, `"response_header_retries":"0"`, `"event":"stream.idle"`, `"event":"stream.first_event"`, `"event":"stream.completed"`} {
		if !strings.Contains(trace, want) {
			t.Fatalf("trace missing %s:\n%s", want, trace)
		}
	}
	if strings.Contains(trace, "trace me") {
		t.Fatalf("trace leaked message content:\n%s", trace)
	}
}

func TestMessagesKeepSchemaBackedToolStreamAliveWhileArgumentsAreBuffered(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	tracePath := home + "/trace.jsonl"
	t.Setenv("CLAUDODEX_PROXY_TRACE", tracePath)
	t.Setenv(streamKeepaliveIntervalEnv, "5ms")

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		writeUpstreamEvent := func(event, data string) {
			_, _ = w.Write([]byte("event: " + event + "\ndata: " + data + "\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
		}

		writeUpstreamEvent("response.created", `{"type":"response.created","response":{"id":"resp_keepalive"}}`)
		writeUpstreamEvent("response.output_item.added", `{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","call_id":"call_write","name":"Write"}}`)
		writeUpstreamEvent("response.function_call_arguments.delta", `{"type":"response.function_call_arguments.delta","output_index":0,"delta":"{\"file_path\":\"a.go\","}`)
		time.Sleep(15 * time.Millisecond)
		writeUpstreamEvent("response.function_call_arguments.delta", `{"type":"response.function_call_arguments.delta","output_index":0,"delta":"\"content\":\"package main\"}"}`)
		time.Sleep(15 * time.Millisecond)
		writeUpstreamEvent("response.output_item.done", `{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","call_id":"call_write","name":"Write"}}`)
		writeUpstreamEvent("response.completed", `{"type":"response.completed","response":{"id":"resp_keepalive","usage":{"input_tokens":1,"output_tokens":8}}}`)
	}))
	defer upstream.Close()

	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	body := `{"model":"claude-opus-4-6","stream":true,"messages":[{"role":"user","content":"write a.go"}],"tools":[{"name":"Write","input_schema":{"type":"object","required":["file_path","content"],"properties":{"file_path":{"type":"string"},"content":{"type":"string"}}}}]}`
	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	sse := readAllString(t, resp)
	pingAt := strings.Index(sse, "event: ping")
	toolStartAt := strings.Index(sse, `"type":"tool_use"`)
	if pingAt < 0 || toolStartAt < 0 || pingAt > toolStartAt {
		t.Fatalf("keepalive was not emitted before the completed tool block:\n%s", sse)
	}
	if got := strings.Count(sse, "event: content_block_delta"); got != 1 {
		t.Fatalf("tool arguments were not emitted atomically: got %d content deltas\n%s", got, sse)
	}
	if !strings.Contains(sse, `\"content\":\"package main\"`) {
		t.Fatalf("completed tool arguments missing from stream:\n%s", sse)
	}

	traceData, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(traceData), `"keepalive_events":1`) {
		t.Fatalf("trace did not record the keepalive:\n%s", traceData)
	}
}

func TestMessagesHeaderTimeoutRetryConsumesEarlyStreamRetryBudget(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	t.Setenv("CLAUDODEX_CODEX_RESPONSE_HEADER_TIMEOUT", "10ms")
	var attempts atomic.Int32
	upstreamClient := &http.Client{Transport: proxyRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempt := attempts.Add(1)
		if trace := httptrace.ContextClientTrace(r.Context()); trace != nil && trace.WroteRequest != nil {
			trace.WroteRequest(httptrace.WroteRequestInfo{})
		}
		if attempt == 1 {
			<-r.Context().Done()
			return nil, r.Context().Err()
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"content-type": []string{"text/event-stream"}},
			Body: io.NopCloser(strings.NewReader(strings.Join([]string{
				"event: response.created",
				`data: {"type":"response.created","response":{"id":"resp_partial"}}`,
				"",
			}, "\n"))),
			Request: r,
		}, nil
	})}
	server := New(Config{Home: home, CodexBaseURL: "https://codex.invalid", HTTPClient: upstreamClient, AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","stream":true,"messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_ = readAllString(t, resp)
	if got := attempts.Load(); got != 2 {
		t.Fatalf("generation attempts = %d, want exactly 2 after header timeout plus early stream failure", got)
	}
}

func TestUpstreamCreateErrorTraceFieldsIncludesResponseHeaderTimeout(t *testing.T) {
	fields := upstreamCreateErrorTraceFields(&codex.ResponseHeaderTimeoutError{Timeout: 125 * time.Millisecond, Attempts: 2}, map[string]any{"attempt": 1})
	if fields["response_header_timeout"] != true {
		t.Fatalf("response_header_timeout = %#v", fields["response_header_timeout"])
	}
	if fields["response_header_timeout_ms"] != int64(125) {
		t.Fatalf("response_header_timeout_ms = %#v", fields["response_header_timeout_ms"])
	}
	if fields["response_header_attempts"] != 2 {
		t.Fatalf("response_header_attempts = %#v", fields["response_header_attempts"])
	}
}

func TestMessagesStreamingBackfillsMissingUsage(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: response.output_item.added`,
			`data: {"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","call_id":"call_1","name":"Read"}}`,
			``,
			`event: response.function_call_arguments.delta`,
			`data: {"type":"response.function_call_arguments.delta","output_index":0,"delta":"{\"file_path\":\"a.go\"}"}`,
			``,
			`event: response.output_item.done`,
			`data: {"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","call_id":"call_1","name":"Read"}}`,
			``,
			`event: response.completed`,
			`data: {"type":"response.completed","response":{"stop_reason":"tool_calls"}}`,
			``,
			``,
		}, "\n")))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	body := `{"model":"claude-opus-4-6","stream":true,"messages":[{"role":"user","content":"read a.go"}],"tools":[{"name":"Read","input_schema":{"type":"object","properties":{"file_path":{"type":"string"}}}}]}`
	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	sse := readAllString(t, resp)
	usage := messageDeltaUsage(t, sse)
	if usage.InputTokens <= 0 || usage.OutputTokens <= 0 {
		t.Fatalf("usage = %#v\nSSE:\n%s", usage, sse)
	}
}

func TestMessagesStreamingForwardsNamedUpstreamErrorEvent(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte("event: error\n"))
		_, _ = w.Write([]byte(`data: {"error":{"type":"api_error","message":"upstream boom"}}` + "\n\n"))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","stream":true,"messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	sse := readAllString(t, resp)
	if !strings.Contains(sse, "event: error") || !strings.Contains(sse, "upstream boom") {
		t.Fatalf("missing upstream error event in SSE:\n%s", sse)
	}
}

func TestShouldRetryStreamRetriesTransientTransportErrors(t *testing.T) {
	if !shouldRetryStream(nil, errors.New("stream error: stream ID 11; INTERNAL_ERROR; received from peer"), false) {
		t.Fatal("expected transient stream reset to be retryable")
	}
	if shouldRetryStream(nil, upstreamStreamEventError{typ: "api_error", message: "quota exhausted"}, false) {
		t.Fatal("upstream event errors should not be retried without implicit resume")
	}
	if !shouldRetryStream(nil, upstreamStreamEventError{typ: "api_error", message: "previous response not found"}, true) {
		t.Fatal("implicit resume errors should be retryable")
	}
}

func TestMessagesStreamingCleanCloseWithoutTerminalEventEmitsError(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte("event: response.created\n"))
		_, _ = w.Write([]byte(`data: {"type":"response.created","response":{"id":"resp_1"}}` + "\n\n"))
		_, _ = w.Write([]byte("event: response.output_text.delta\n"))
		_, _ = w.Write([]byte(`data: {"type":"response.output_text.delta","delta":"partial"}` + "\n\n"))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","stream":true,"messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	sse := readAllString(t, resp)
	if !strings.Contains(sse, "event: error") || !strings.Contains(sse, "ended before completion") {
		t.Fatalf("missing premature-close error event in SSE:\n%s", sse)
	}
}

func TestMessagesNonStreamingMalformedUpstreamSSEReturns502(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte("event: response.output_text.delta\n"))
		_, _ = w.Write([]byte("data: {not-json}\n\n"))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","stream":false,"messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body struct {
		Type  string `json:"type"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Type != "error" || body.Error.Type != "api_error" || !strings.Contains(body.Error.Message, "malformed SSE") {
		t.Fatalf("body = %#v", body)
	}
}

func TestMessagesNonStreamingCleanCloseWithoutTerminalEventReturns502(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		_, _ = w.Write([]byte("event: response.output_text.delta\n"))
		_, _ = w.Write([]byte(`data: {"type":"response.output_text.delta","delta":"partial"}` + "\n\n"))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","stream":false,"messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body.Error.Message, "ended before completion") {
		t.Fatalf("body = %#v", body)
	}
}

func TestMessagesRefreshesAndRetriesOnceOn401(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "expired-access")
	var attempts atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/codex/responses":
			attempt := attempts.Add(1)
			if attempt == 1 {
				http.Error(w, `{"error":{"message":"expired"}}`, http.StatusUnauthorized)
				return
			}
			if got := r.Header.Get("authorization"); got != "Bearer fresh-access" {
				t.Fatalf("retry authorization = %q", got)
			}
			w.Header().Set("content-type", "text/event-stream")
			_, _ = w.Write([]byte("event: response.completed\n"))
			_, _ = w.Write([]byte(`data: {"type":"response.completed","response":{"usage":{"input_tokens":0,"output_tokens":0}}}` + "\n\n"))
		case "/oauth/token":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"fresh-access","refresh_token":"refresh-2"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	server := New(Config{Home: home, CodexBaseURL: upstream.URL, TokenEndpoint: upstream.URL + "/oauth/token", HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.StatusCode, readAllString(t, resp))
	}
	if attempts.Load() != 2 {
		t.Fatalf("attempts = %d, want 2", attempts.Load())
	}
}

func TestMessages401ThenHeaderTimeoutUsesOnlyTwoGenerationAttempts(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "expired-access")
	t.Setenv("CLAUDODEX_CODEX_RESPONSE_HEADER_TIMEOUT", "10ms")
	var generationAttempts atomic.Int32
	upstreamClient := &http.Client{Transport: proxyRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/oauth/token":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"content-type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"fresh-access","refresh_token":"refresh-2"}`)),
				Request:    r,
			}, nil
		case "/codex/responses":
			attempt := generationAttempts.Add(1)
			if attempt == 1 {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"expired"}}`)),
					Request:    r,
				}, nil
			}
			if trace := httptrace.ContextClientTrace(r.Context()); trace != nil && trace.WroteRequest != nil {
				trace.WroteRequest(httptrace.WroteRequestInfo{})
			}
			<-r.Context().Done()
			return nil, r.Context().Err()
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Header: make(http.Header), Body: http.NoBody, Request: r}, nil
		}
	})}
	server := New(Config{Home: home, CodexBaseURL: "https://codex.invalid", TokenEndpoint: "https://codex.invalid/oauth/token", HTTPClient: upstreamClient, AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("status = %d body=%s", resp.StatusCode, readAllString(t, resp))
	}
	if got := generationAttempts.Load(); got != 2 {
		t.Fatalf("generation attempts = %d, want 401 plus one timed-out refreshed request", got)
	}
}

func TestMessages401ThenSuccessConsumesEarlyStreamRetryBudget(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "expired-access")
	var generationAttempts atomic.Int32
	upstreamClient := &http.Client{Transport: proxyRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/oauth/token":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"content-type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"fresh-access","refresh_token":"refresh-2"}`)),
				Request:    r,
			}, nil
		case "/codex/responses":
			attempt := generationAttempts.Add(1)
			if attempt == 1 {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"expired"}}`)),
					Request:    r,
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"content-type": []string{"text/event-stream"}},
				Body: io.NopCloser(strings.NewReader(strings.Join([]string{
					"event: response.created",
					`data: {"type":"response.created","response":{"id":"resp_partial"}}`,
					"",
				}, "\n"))),
				Request: r,
			}, nil
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Header: make(http.Header), Body: http.NoBody, Request: r}, nil
		}
	})}
	server := New(Config{Home: home, CodexBaseURL: "https://codex.invalid", TokenEndpoint: "https://codex.invalid/oauth/token", HTTPClient: upstreamClient, AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","stream":true,"messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_ = readAllString(t, resp)
	if got := generationAttempts.Load(); got != 2 {
		t.Fatalf("generation attempts = %d, want no stream retry after 401 plus refreshed generation", got)
	}
}

func TestMessagesGeneratedRouteIDsStayStableAcrossAuthRetry(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "expired-access")
	var routeHeaders [][3]string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/codex/responses":
			routeHeaders = append(routeHeaders, [3]string{
				r.Header.Get("x-client-request-id"),
				r.Header.Get("session-id"),
				r.Header.Get("thread-id"),
			})
			if len(routeHeaders) == 1 {
				http.Error(w, `{"error":{"message":"expired"}}`, http.StatusUnauthorized)
				return
			}
			w.Header().Set("content-type", "text/event-stream")
			_, _ = io.WriteString(w, "event: response.completed\n")
			_, _ = io.WriteString(w, `data: {"type":"response.completed","response":{"usage":{"input_tokens":0,"output_tokens":0}}}`+"\n\n")
		case "/oauth/token":
			w.Header().Set("content-type", "application/json")
			_, _ = io.WriteString(w, `{"access_token":"fresh-access","refresh_token":"refresh-2"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, TokenEndpoint: upstream.URL + "/oauth/token", HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.StatusCode, readAllString(t, resp))
	}
	if len(routeHeaders) != 2 || routeHeaders[0][0] == "" || routeHeaders[0] != routeHeaders[1] {
		t.Fatalf("generated route headers changed across auth retry: %#v", routeHeaders)
	}
	if routeHeaders[0][0] != routeHeaders[0][1] || routeHeaders[0][1] != routeHeaders[0][2] {
		t.Fatalf("generated route headers are inconsistent: %#v", routeHeaders[0])
	}
}

func TestMessagesGeneratedRouteIDsStayStableAcrossEarlyStreamRetry(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	var attempts atomic.Int32
	var routeHeaders [][3]string
	upstreamClient := &http.Client{Transport: proxyRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		routeHeaders = append(routeHeaders, [3]string{
			r.Header.Get("x-client-request-id"),
			r.Header.Get("session-id"),
			r.Header.Get("thread-id"),
		})
		attempt := attempts.Add(1)
		if attempt == 1 {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"content-type": []string{"text/event-stream"}},
				Body:       errorReadCloser{err: io.ErrUnexpectedEOF},
				Request:    r,
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"content-type": []string{"text/event-stream"}},
			Body: io.NopCloser(strings.NewReader(strings.Join([]string{
				"event: response.output_item.done",
				`data: {"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"recovered"}]}}`,
				"",
				"event: response.completed",
				`data: {"type":"response.completed","response":{"usage":{"input_tokens":1,"output_tokens":1}}}`,
				"",
			}, "\n"))),
			Request: r,
		}, nil
	})}
	server := New(Config{Home: home, CodexBaseURL: "https://codex.invalid", HTTPClient: upstreamClient, AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","stream":true,"messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if body := readAllString(t, resp); !strings.Contains(body, "recovered") {
		t.Fatalf("missing recovered response: %s", body)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("generation attempts = %d, want one early stream retry", got)
	}
	if len(routeHeaders) != 2 || routeHeaders[0][0] == "" || routeHeaders[0] != routeHeaders[1] {
		t.Fatalf("generated route headers changed across stream retry: %#v", routeHeaders)
	}
	if routeHeaders[0][0] != routeHeaders[0][1] || routeHeaders[0][1] != routeHeaders[0][2] {
		t.Fatalf("generated route headers are inconsistent: %#v", routeHeaders[0])
	}
}

func TestMessagesMapsNonJSONUpstreamError(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html>nope</html>"))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Type != "permission_error" || body.Error.Message != "Codex upstream returned HTTP 403" {
		t.Fatalf("body = %#v", body)
	}
}

func TestMessagesMaps429RateLimitHeaders(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-codex-primary-used-percent", "100")
		w.Header().Set("x-codex-primary-window-minutes", "300")
		w.Header().Set("x-codex-primary-reset-at", "1770000000")
		w.Header().Set("retry-after", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"quota exhausted"}}`))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Post("http://"+addr+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4-6","messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if got := resp.Header.Get("anthropic-ratelimit-unified-status"); got != "rejected" {
		t.Fatalf("status header = %q", got)
	}
	if got := resp.Header.Get("anthropic-ratelimit-unified-representative-claim"); got != "five_hour" {
		t.Fatalf("claim header = %q", got)
	}
	var body struct {
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Type != "rate_limit_error" {
		t.Fatalf("body = %#v", body)
	}
}

func TestUsageFetchesWhamUsageAndMapsResponse(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wham/usage" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("authorization"); got != "Bearer access-1" {
			t.Fatalf("authorization = %q", got)
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"account_id":"hidden",
			"email":"hidden@example.com",
			"rate_limit":{
				"primary_window":{"used_percent":12,"limit_window_seconds":18000,"reset_at":1770000000},
				"secondary_window":{"used_percent":34,"limit_window_seconds":604800,"reset_at":1770500000}
			},
			"credits":{"has_credits":true,"unlimited":false},
			"spend_control":{"individual_limit":null}
		}`))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Get("http://" + addr + "/api/oauth/usage")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["account_id"]; ok {
		t.Fatalf("account_id leaked: %#v", body)
	}
	five := body["five_hour"].(map[string]any)
	if five["utilization"] != float64(12) {
		t.Fatalf("five_hour = %#v", five)
	}
	extra := body["extra_usage"].(map[string]any)
	if extra["is_enabled"] != true {
		t.Fatalf("extra_usage = %#v", extra)
	}
}

func TestUsageAllowsMissingFiveHourWindow(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "access-1")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wham/usage" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"rate_limit":{"secondary_window":{"used_percent":34,"limit_window_seconds":604800}}}`))
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Get("http://" + addr + "/api/oauth/usage")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if fiveHour, ok := body["five_hour"]; !ok || fiveHour != nil {
		t.Fatalf("five_hour = %#v, present=%v; want present null", fiveHour, ok)
	}
	sevenDay, _ := body["seven_day"].(map[string]any)
	if sevenDay == nil || sevenDay["utilization"] != float64(34) {
		t.Fatalf("seven_day = %#v, want mapped secondary window", body["seven_day"])
	}
}

func TestUsageRefreshesAndRetriesOnceOn401(t *testing.T) {
	home := t.TempDir()
	saveTestAuth(t, home, "expired-access")
	var attempts atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wham/usage":
			attempt := attempts.Add(1)
			if attempt == 1 {
				http.Error(w, `{"error":{"message":"expired"}}`, http.StatusUnauthorized)
				return
			}
			if got := r.Header.Get("authorization"); got != "Bearer fresh-access" {
				t.Fatalf("retry authorization = %q", got)
			}
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{
				"rate_limit":{
					"primary_window":{"used_percent":1,"limit_window_seconds":18000,"reset_at":1770000000},
					"secondary_window":{"used_percent":2,"limit_window_seconds":604800,"reset_at":1770500000}
				}
			}`))
		case "/oauth/token":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"fresh-access","refresh_token":"refresh-2"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()
	server := New(Config{Home: home, CodexBaseURL: upstream.URL, TokenEndpoint: upstream.URL + "/oauth/token", HTTPClient: upstream.Client(), AuthPresent: true})
	addr, err := server.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Get("http://" + addr + "/api/oauth/usage")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.StatusCode, readAllString(t, resp))
	}
	if attempts.Load() != 2 {
		t.Fatalf("attempts = %d, want 2", attempts.Load())
	}
}

type proxyRoundTripFunc func(*http.Request) (*http.Response, error)

func (f proxyRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type errorReadCloser struct {
	err error
}

func (r errorReadCloser) Read([]byte) (int, error) { return 0, r.err }
func (errorReadCloser) Close() error               { return nil }

func saveTestAuth(t *testing.T, home, accessToken string) {
	t.Helper()
	err := auth.NewStore(home).Save(auth.File{
		AuthMode: "chatgpt",
		Issuer:   auth.Issuer,
		ClientID: auth.ClientID,
		Tokens: auth.Tokens{
			AccessToken:  accessToken,
			RefreshToken: "refresh-1",
			AccountID:    "acc_123",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func readAllString(t *testing.T, resp *http.Response) string {
	t.Helper()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func thinkingTextFromSSE(t *testing.T, sse string) string {
	t.Helper()
	var thinking strings.Builder
	for _, line := range strings.Split(sse, "\n") {
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			t.Fatalf("decode SSE data %q: %v", data, err)
		}
		delta, _ := event["delta"].(map[string]any)
		if delta["type"] == "thinking_delta" {
			text, _ := delta["thinking"].(string)
			thinking.WriteString(text)
		}
	}
	return thinking.String()
}

func writeWSJSON(t *testing.T, conn *websocket.Conn, value any) {
	t.Helper()
	if err := conn.WriteJSON(value); err != nil {
		t.Fatal(err)
	}
}

func messageDeltaUsage(t *testing.T, sse string) struct {
	InputTokens              int `json:"input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	OutputTokens             int `json:"output_tokens"`
} {
	t.Helper()
	for _, line := range strings.Split(sse, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") || !strings.Contains(line, `"type":"message_delta"`) {
			continue
		}
		var event struct {
			Usage struct {
				InputTokens              int `json:"input_tokens"`
				CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
				CacheReadInputTokens     int `json:"cache_read_input_tokens"`
				OutputTokens             int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
			t.Fatal(err)
		}
		return event.Usage
	}
	t.Fatalf("missing message_delta usage in SSE:\n%s", sse)
	return struct {
		InputTokens              int `json:"input_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		OutputTokens             int `json:"output_tokens"`
	}{}
}
