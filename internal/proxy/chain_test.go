package proxy

import (
	"encoding/json"
	"testing"

	"github.com/bassner/claudodex/internal/codex"
)

func TestResponseTraceBackfillsFunctionCallArguments(t *testing.T) {
	var trace responseTrace
	trace.observe(codex.SSEEvent{Event: "response.created", Data: json.RawMessage(`{"response":{"id":"resp_1"}}`)})
	trace.observe(codex.SSEEvent{Event: "response.output_item.done", Data: json.RawMessage(`{"output_index":0,"item":{"type":"function_call","call_id":"call_1","name":"Read"}}`)})
	trace.observe(codex.SSEEvent{Event: "response.function_call_arguments.done", Data: json.RawMessage(`{"output_index":0,"arguments":"{\"file_path\":\"README.md\"}"}`)})

	if trace.ResponseID != "resp_1" {
		t.Fatalf("response id = %q", trace.ResponseID)
	}
	if len(trace.Output) != 1 {
		t.Fatalf("output len = %d, output = %#v", len(trace.Output), trace.Output)
	}
	if trace.Output[0].Arguments != `{"file_path":"README.md"}` {
		t.Fatalf("arguments = %q", trace.Output[0].Arguments)
	}
}

func TestImplicitResumeUsesRecordedFunctionCallArguments(t *testing.T) {
	server := New(Config{})
	previous := codex.Request{
		Model:  "gpt-5.5",
		Input:  []codex.InputItem{{Type: "message", Role: "user", Content: []codex.ContentPart{{Type: "input_text", Text: "read"}}}},
		Stream: true,
		Store:  false,
	}
	trace := responseTrace{
		ResponseID: "resp_1",
		Output: []codex.InputItem{{
			Type:      "function_call",
			CallID:    "call_1",
			Name:      "Read",
			Arguments: `{"file_path":"README.md"}`,
		}},
	}
	server.recordResponseChain("chain-1", previous, trace)

	current := previous
	current.Input = append(append([]codex.InputItem(nil), previous.Input...), codex.InputItem{
		Type:      "function_call",
		CallID:    "call_1",
		Name:      "Read",
		Arguments: `{"file_path":"README.md"}`,
	}, codex.InputItem{
		Type:   "function_call_output",
		CallID: "call_1",
		Output: "contents",
	})

	used, reason, prefixItems, inputItems := server.applyImplicitResumeDetailed("chain-1", &current)
	if !used || reason != "applied" {
		t.Fatalf("resume = %v reason %q prefix %d input %d", used, reason, prefixItems, inputItems)
	}
	if current.PreviousResponseID != "resp_1" {
		t.Fatalf("previous_response_id = %q", current.PreviousResponseID)
	}
	if len(current.Input) != 1 || current.Input[0].Type != "function_call_output" {
		t.Fatalf("incremental input = %#v", current.Input)
	}
}

func TestImplicitResumeAllowsToolSetChangesAndTrimsAfterRecordedCalls(t *testing.T) {
	server := New(Config{})
	previous := codex.Request{
		Model:        "gpt-5.5",
		Instructions: "initial instructions",
		Input: []codex.InputItem{
			{Type: "message", Role: "user", Content: []codex.ContentPart{{Type: "input_text", Text: "read"}}},
			{Type: "function_call", CallID: "call_old", Name: "Bash", Arguments: `{"command":"pwd"}`},
			{Type: "function_call_output", CallID: "call_old", Output: "root"},
		},
		Tools:  []codex.Tool{{Type: "function", Name: "Bash"}},
		Stream: true,
		Store:  false,
	}
	trace := responseTrace{
		ResponseID: "resp_2",
		Output: []codex.InputItem{{
			Type:      "function_call",
			CallID:    "call_glob",
			Name:      "Glob",
			Arguments: `{"pattern":"*.md"}`,
		}},
	}
	server.recordResponseChain("chain-2", previous, trace)

	current := previous
	current.Instructions = "updated instructions after skill load"
	current.Tools = []codex.Tool{{Type: "function", Name: "Bash"}, {Type: "function", Name: "Glob"}}
	current.Input = append(append([]codex.InputItem(nil), previous.Input...), codex.InputItem{
		Type:      "function_call",
		CallID:    "call_glob",
		Name:      "Glob",
		Arguments: `{"pattern":"README.md"}`,
	}, codex.InputItem{
		Type:   "function_call_output",
		CallID: "call_glob",
		Output: "README.md",
	})

	used, reason, _, _ := server.applyImplicitResumeDetailed("chain-2", &current)
	if !used || reason != "applied_by_output_calls" {
		t.Fatalf("resume = %v reason %q", used, reason)
	}
	if current.PreviousResponseID != "resp_2" {
		t.Fatalf("previous_response_id = %q", current.PreviousResponseID)
	}
	if len(current.Input) != 1 || current.Input[0].Type != "function_call_output" || current.Input[0].CallID != "call_glob" {
		t.Fatalf("incremental input = %#v", current.Input)
	}
}
