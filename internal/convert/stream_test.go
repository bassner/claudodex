package convert

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/codex"
)

func TestStreamReducerStreamsTextAndIgnoresReasoning(t *testing.T) {
	reducer := NewStreamReducer("msg_1", "claude-sonnet-4-6")
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.created","response":{"id":"resp_1"}}`,
		`{"type":"response.reasoning_summary_text.delta","delta":"hidden"}`,
		`{"type":"response.output_item.added","item":{"type":"message","id":"msg"}}`,
		`{"type":"response.output_text.delta","delta":"hel"}`,
		`{"type":"response.output_text.delta","delta":"lo"}`,
		`{"type":"response.output_item.done","item":{"type":"message","id":"msg","content":[{"type":"output_text","text":"hello"}]}}`,
		`{"type":"response.completed","response":{"usage":{"input_tokens":10,"input_tokens_details":{"cached_tokens":3},"output_tokens":2}}}`,
	} {
		next, err := reducer.Reduce(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, next...)
	}
	message, errEvent := AssembleMessage(events, "", "claude-sonnet-4-6")
	if errEvent != nil {
		t.Fatalf("unexpected error event: %#v", errEvent)
	}
	content := message["content"].([]map[string]any)
	if len(content) != 1 || content[0]["text"] != "hello" {
		t.Fatalf("content = %#v", content)
	}
	usage := message["usage"].(Usage)
	if usage.InputTokens != 7 || usage.CacheReadInputTokens != 3 || usage.OutputTokens != 2 {
		t.Fatalf("usage = %#v", usage)
	}
	for _, event := range events {
		if event.Event == "content_block_start" {
			block := event.Data["content_block"].(map[string]any)
			if block["type"] == "thinking" {
				t.Fatalf("reasoning leaked as thinking block: %#v", event)
			}
		}
	}
}

func TestStreamReducerStreamsToolArgumentsAsDeltas(t *testing.T) {
	reducer := NewStreamReducer("msg_1", "claude-opus-4-6")
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","call_id":"call_1","name":"read_file"}}`,
		`{"type":"response.function_call_arguments.delta","output_index":0,"delta":"{\"path\""}`,
		`{"type":"response.function_call_arguments.delta","output_index":0,"delta":":\"a.go\"}"}`,
		`{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","call_id":"call_1","name":"read_file","arguments":"{\"path\":\"a.go\"}"}}`,
		`{"type":"response.completed","response":{"stop_reason":"tool_calls"}}`,
	} {
		next, err := reducer.Reduce(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, next...)
	}
	message, errEvent := AssembleMessage(events, "", "claude-opus-4-6")
	if errEvent != nil {
		t.Fatalf("unexpected error event: %#v", errEvent)
	}
	content := message["content"].([]map[string]any)
	if len(content) != 1 {
		t.Fatalf("content = %#v", content)
	}
	input := content[0]["input"].(map[string]any)
	if content[0]["type"] != "tool_use" || content[0]["id"] != "call_1" || input["path"] != "a.go" {
		t.Fatalf("tool content = %#v", content[0])
	}
	if message["stop_reason"] != "tool_use" {
		t.Fatalf("stop_reason = %#v", message["stop_reason"])
	}
}

func TestStreamReducerBackfillsMissingUsageForVisibleToolCall(t *testing.T) {
	reducer := NewStreamReducerWithOptions("msg_1", "claude-opus-4-6", StreamReducerOptions{
		FallbackInputTokens: 123,
	})
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","call_id":"call_1","name":"Read"}}`,
		`{"type":"response.function_call_arguments.delta","output_index":0,"delta":"{\"file_path\":\"a.go\"}"}`,
		`{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","call_id":"call_1","name":"Read"}}`,
		`{"type":"response.completed","response":{"stop_reason":"tool_calls"}}`,
	} {
		next, err := reducer.Reduce(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, next...)
	}
	message, errEvent := AssembleMessage(events, "", "claude-opus-4-6")
	if errEvent != nil {
		t.Fatalf("unexpected error event: %#v", errEvent)
	}
	usage := message["usage"].(Usage)
	if usage.InputTokens != 123 || usage.OutputTokens <= 0 {
		t.Fatalf("usage = %#v", usage)
	}
}

func TestStreamReducerAddsFallbackInputUsageToMessageStart(t *testing.T) {
	reducer := NewStreamReducerWithOptions("msg_1", "claude-opus-4-6", StreamReducerOptions{
		FallbackInputTokens: 123,
	})
	events, err := reducer.Reduce(json.RawMessage(`{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","call_id":"call_1","name":"Read"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 || events[0].Event != "message_start" {
		t.Fatalf("events = %#v", events)
	}
	message := events[0].Data["message"].(map[string]any)
	usage := message["usage"].(Usage)
	if usage.InputTokens != 123 || usage.OutputTokens != 0 {
		t.Fatalf("usage = %#v", usage)
	}
}

func TestStreamReducerUsesFullRequestEstimateAsInputFloor(t *testing.T) {
	tests := []struct {
		name           string
		fallback       int
		input          int
		cacheRead      int
		wantInput      int
		wantTotalInput int
	}{
		{
			name:           "resumed input below full request",
			fallback:       61_962,
			input:          1_000,
			wantInput:      61_962,
			wantTotalInput: 61_962,
		},
		{
			name:           "cache accounting is preserved while filling the gap",
			fallback:       258_400,
			input:          35_614,
			cacheRead:      185_344,
			wantInput:      73_056,
			wantTotalInput: 258_400,
		},
		{
			name:           "upstream total above floor is unchanged",
			fallback:       100_000,
			input:          90_000,
			cacheRead:      30_000,
			wantInput:      90_000,
			wantTotalInput: 120_000,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := map[string]any{
				"response": map[string]any{
					"usage": map[string]any{
						"input_tokens":            test.input,
						"cache_read_input_tokens": test.cacheRead,
						"output_tokens":           7,
					},
				},
			}
			reducer := NewStreamReducerWithOptions("msg_1", "claude-opus-4-6", StreamReducerOptions{
				FallbackInputTokens: test.fallback,
			})

			for phase, usage := range map[string]Usage{
				"start":  reducer.usageForStart(event),
				"finish": reducer.usageForFinish(event),
			} {
				if usage.InputTokens != test.wantInput || usage.CacheReadInputTokens != test.cacheRead || usageInputTokens(usage) != test.wantTotalInput {
					t.Fatalf("%s usage = %#v, want input=%d cache_read=%d total=%d", phase, usage, test.wantInput, test.cacheRead, test.wantTotalInput)
				}
			}
		})
	}
}

func TestStreamReducerSupplementsOutputOnlyUsage(t *testing.T) {
	reducer := NewStreamReducerWithOptions("msg_1", "claude-opus-4-6", StreamReducerOptions{
		FallbackInputTokens: 456,
	})
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.output_item.added","item":{"type":"message","id":"msg"}}`,
		`{"type":"response.output_text.delta","delta":"done"}`,
		`{"type":"response.output_item.done","item":{"type":"message","id":"msg","content":[{"type":"output_text","text":"done"}]}}`,
		`{"type":"response.completed","response":{"usage":{"output_tokens":7}}}`,
	} {
		next, err := reducer.Reduce(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, next...)
	}
	message, errEvent := AssembleMessage(events, "", "claude-opus-4-6")
	if errEvent != nil {
		t.Fatalf("unexpected error event: %#v", errEvent)
	}
	usage := message["usage"].(Usage)
	if usage.InputTokens != 456 || usage.OutputTokens != 7 {
		t.Fatalf("usage = %#v", usage)
	}
}

func TestStreamReducerBackfillsInputUsageForEmptyCompletion(t *testing.T) {
	reducer := NewStreamReducerWithOptions("msg_1", "claude-opus-4-6", StreamReducerOptions{
		FallbackInputTokens: 789,
	})
	events, err := reducer.Reduce(json.RawMessage(`{"type":"response.completed","response":{"usage":{"input_tokens":0,"output_tokens":0}}}`))
	if err != nil {
		t.Fatal(err)
	}
	message, errEvent := AssembleMessage(events, "", "claude-opus-4-6")
	if errEvent != nil {
		t.Fatalf("unexpected error event: %#v", errEvent)
	}
	usage := message["usage"].(Usage)
	if usage.InputTokens != 789 || usage.OutputTokens != 0 {
		t.Fatalf("usage = %#v", usage)
	}
}

func TestUsageFromEventAcceptsPromptTokenAliases(t *testing.T) {
	var event map[string]any
	if err := json.Unmarshal([]byte(`{"response":{"usage":{"prompt_tokens":10,"prompt_tokens_details":{"cached_tokens":2},"completion_tokens":3}}}`), &event); err != nil {
		t.Fatal(err)
	}
	usage := usageFromEvent(event)
	if usage.InputTokens != 8 || usage.CacheReadInputTokens != 2 || usage.OutputTokens != 3 {
		t.Fatalf("usage = %#v", usage)
	}
}

func TestUsageFromEventDerivesInputFromTotalTokens(t *testing.T) {
	var event map[string]any
	if err := json.Unmarshal([]byte(`{"usage":{"total_tokens":25,"output_tokens":5}}`), &event); err != nil {
		t.Fatal(err)
	}
	usage := usageFromEvent(event)
	if usage.InputTokens != 20 || usage.OutputTokens != 5 {
		t.Fatalf("usage = %#v", usage)
	}
}

func TestStreamReducerUsesFunctionArgumentsDoneWhenNoDeltasArrive(t *testing.T) {
	reducer := NewStreamReducer("msg_1", "claude-opus-4-6")
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","call_id":"call_1","name":"read_file"}}`,
		`{"type":"response.function_call_arguments.done","output_index":0,"arguments":"{\"path\":\"a.go\"}"}`,
		`{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","call_id":"call_1","name":"read_file"}}`,
		`{"type":"response.completed","response":{"stop_reason":"tool_calls"}}`,
	} {
		next, err := reducer.Reduce(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, next...)
	}
	message, errEvent := AssembleMessage(events, "", "claude-opus-4-6")
	if errEvent != nil {
		t.Fatalf("unexpected error event: %#v", errEvent)
	}
	content := message["content"].([]map[string]any)
	input := content[0]["input"].(map[string]any)
	if input["path"] != "a.go" {
		t.Fatalf("tool input = %#v", input)
	}
}

func TestStreamReducerPrunesEmptyOptionalToolArguments(t *testing.T) {
	reducer := NewStreamReducerWithOptions("msg_1", "claude-opus-4-6", StreamReducerOptions{
		ToolSchemas: map[string]map[string]any{
			"Read": {
				"type":     "object",
				"required": []any{"file_path"},
				"properties": map[string]any{
					"file_path": map[string]any{"type": "string"},
					"limit":     map[string]any{"type": "number"},
					"offset":    map[string]any{"type": "number"},
					"pages":     map[string]any{"type": "string"},
				},
			},
		},
	})
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","call_id":"call_1","name":"Read"}}`,
		`{"type":"response.function_call_arguments.delta","output_index":0,"delta":"{\"file_path\":\"/tmp/meeting_today.txt\",\"limit\":1,\"offset\":0,\"pages\":\"\"}"}`,
		`{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","call_id":"call_1","name":"Read"}}`,
		`{"type":"response.completed","response":{"stop_reason":"tool_calls"}}`,
	} {
		next, err := reducer.Reduce(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, next...)
	}
	message, errEvent := AssembleMessage(events, "", "claude-opus-4-6")
	if errEvent != nil {
		t.Fatalf("unexpected error event: %#v", errEvent)
	}
	content := message["content"].([]map[string]any)
	input := content[0]["input"].(map[string]any)
	if input["file_path"] != "/tmp/meeting_today.txt" || input["limit"] != float64(1) || input["offset"] != float64(0) {
		t.Fatalf("tool input lost required/non-empty fields: %#v", input)
	}
	if _, ok := input["pages"]; ok {
		t.Fatalf("empty optional pages was not pruned: %#v", input)
	}
}

func TestStreamReducerBuffersReadArgumentsForSanitization(t *testing.T) {
	reducer := NewStreamReducerWithOptions("msg_1", "claude-opus-4-6", StreamReducerOptions{
		ToolSchemas: map[string]map[string]any{
			"Read": {
				"type":     "object",
				"required": []any{"file_path"},
				"properties": map[string]any{
					"file_path": map[string]any{"type": "string"},
					"pages":     map[string]any{"type": "string"},
				},
			},
		},
	})
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","call_id":"call_1","name":"Read"}}`,
		`{"type":"response.function_call_arguments.delta","output_index":0,"delta":"{\"file_path\":\"/tmp/meeting_today.txt\",\"pages\":\"\"}"}`,
	} {
		next, err := reducer.Reduce(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, next...)
	}
	for _, event := range events {
		if event.Event == "content_block_start" || event.Event == "content_block_delta" {
			t.Fatalf("Read tool block was emitted before sanitization: %#v", event)
		}
	}
	next, err := reducer.Reduce(json.RawMessage(`{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","call_id":"call_1","name":"Read"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !containsInputJSONDelta(next) {
		t.Fatalf("sanitized Read arguments were not emitted on tool stop: %#v", next)
	}
}

func TestStreamReducerBuffersWriteArgumentsWithToolSchemas(t *testing.T) {
	reducer := NewStreamReducerWithOptions("msg_1", "claude-opus-4-6", StreamReducerOptions{
		ToolSchemas: map[string]map[string]any{
			"Write": {
				"type":     "object",
				"required": []any{"file_path", "content"},
				"properties": map[string]any{
					"file_path": map[string]any{"type": "string"},
					"content":   map[string]any{"type": "string"},
				},
			},
		},
	})
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","call_id":"call_1","name":"Write"}}`,
		`{"type":"response.function_call_arguments.delta","output_index":0,"delta":"{\"file_path\":\"a.go\","}`,
		`{"type":"response.function_call_arguments.delta","output_index":0,"delta":"\"content\":\"package main\"}"}`,
	} {
		next, err := reducer.Reduce(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, next...)
	}
	for _, event := range events {
		if event.Event == "content_block_start" || event.Event == "content_block_delta" {
			t.Fatalf("Write tool block was emitted before completed arguments: %#v", event)
		}
	}
	next, err := reducer.Reduce(json.RawMessage(`{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","call_id":"call_1","name":"Write"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !containsInputJSONDelta(next) {
		t.Fatalf("Write arguments were not emitted on tool stop: %#v", next)
	}
}

func TestStreamReducerLeavesAgentModelAliasForClaudeCodeValidation(t *testing.T) {
	reducer := NewStreamReducerWithOptions("msg_1", "gpt-5.5", StreamReducerOptions{
		ToolSchemas: map[string]map[string]any{
			"Agent": {
				"type":     "object",
				"required": []any{"description", "prompt"},
				"properties": map[string]any{
					"description": map[string]any{"type": "string"},
					"prompt":      map[string]any{"type": "string"},
					"model":       map[string]any{"type": "string"},
				},
			},
		},
	})
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","call_id":"call_agent","name":"Agent"}}`,
		`{"type":"response.function_call_arguments.delta","output_index":0,"delta":"{\"description\":\"Retry cluster\",\"prompt\":\"do it\",\"model\":\"sonnet\"}"}`,
		`{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","call_id":"call_agent","name":"Agent"}}`,
		`{"type":"response.completed","response":{"stop_reason":"tool_calls"}}`,
	} {
		next, err := reducer.Reduce(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, next...)
	}
	message, errEvent := AssembleMessage(events, "", "gpt-5.5")
	if errEvent != nil {
		t.Fatalf("unexpected error event: %#v", errEvent)
	}
	content := message["content"].([]map[string]any)
	input := content[0]["input"].(map[string]any)
	if input["model"] != "sonnet" {
		t.Fatalf("agent model = %#v, input = %#v", input["model"], input)
	}
}

func TestStreamReducerPrunesEmptyAgentModel(t *testing.T) {
	reducer := NewStreamReducerWithOptions("msg_1", "gpt-5.5", StreamReducerOptions{
		ToolSchemas: map[string]map[string]any{
			"Agent": {
				"type": "object",
				"properties": map[string]any{
					"description": map[string]any{"type": "string"},
					"prompt":      map[string]any{"type": "string"},
					"model":       map[string]any{"type": "string"},
				},
			},
		},
	})
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","call_id":"call_agent","name":"Agent"}}`,
		`{"type":"response.function_call_arguments.done","output_index":0,"arguments":"{\"description\":\"Retry cluster\",\"prompt\":\"do it\",\"model\":\"\"}"}`,
		`{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","call_id":"call_agent","name":"Agent"}}`,
		`{"type":"response.completed","response":{"stop_reason":"tool_calls"}}`,
	} {
		next, err := reducer.Reduce(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, next...)
	}
	message, errEvent := AssembleMessage(events, "", "gpt-5.5")
	if errEvent != nil {
		t.Fatalf("unexpected error event: %#v", errEvent)
	}
	content := message["content"].([]map[string]any)
	input := content[0]["input"].(map[string]any)
	if _, ok := input["model"]; ok {
		t.Fatalf("empty agent model was not pruned: %#v", input)
	}
}

func TestStreamReducerUsesSSEEventNameForError(t *testing.T) {
	reducer := NewStreamReducer("msg_1", "claude-opus-4-6")
	events, err := reducer.ReduceNamed("error", json.RawMessage(`{"error":{"type":"api_error","message":"boom"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Event != "error" {
		t.Fatalf("events = %#v", events)
	}
	errObj := events[0].Data["error"].(map[string]any)
	if errObj["type"] != "api_error" || errObj["message"] != "boom" {
		t.Fatalf("error = %#v", errObj)
	}
}

func TestStreamReducerGoldenCodexSSEToAnthropicSSE(t *testing.T) {
	input, err := os.Open("testdata/codex_text_tool.sse")
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()

	reducer := NewStreamReducer("", "claude-sonnet-4-6")
	var events []AnthropicSSE
	if err := codex.ReadSSE(input, func(event codex.SSEEvent) error {
		next, err := reducer.Reduce(event.Data)
		if err != nil {
			return err
		}
		events = append(events, next...)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	got := renderGoldenSSE(t, events)
	wantBytes, err := os.ReadFile("testdata/anthropic_text_tool.golden.sse")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.ReplaceAll(string(wantBytes), "\r\n", "\n")
	if got != want {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func renderGoldenSSE(t *testing.T, events []AnthropicSSE) string {
	t.Helper()
	var out strings.Builder
	for _, event := range events {
		data, err := json.Marshal(event.Data)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&out, "event: %s\ndata: %s\n\n", event.Event, data)
	}
	return out.String()
}

func containsInputJSONDelta(events []AnthropicSSE) bool {
	for _, event := range events {
		if event.Event != "content_block_delta" {
			continue
		}
		delta, _ := event.Data["delta"].(map[string]any)
		if delta["type"] == "input_json_delta" {
			return true
		}
	}
	return false
}
