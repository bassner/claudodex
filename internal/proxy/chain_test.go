package proxy

import (
	"encoding/json"
	"strings"
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

func TestStatelessReplayPreservesEncryptedReasoningPhaseAndOrder(t *testing.T) {
	server := New(Config{})
	previous := codex.Request{
		Model:   "gpt-5.6-terra",
		Input:   []codex.InputItem{{Type: "message", Role: "user", Content: []codex.ContentPart{{Type: "input_text", Text: "inspect"}}}},
		Include: []string{"reasoning.encrypted_content"},
		Stream:  true,
		Store:   false,
	}
	var reasoning codex.InputItem
	if err := json.Unmarshal([]byte(`{"id":"rs_1","type":"reasoning","summary":[{"type":"summary_text","text":"display only"}],"encrypted_content":"opaque-reasoning"}`), &reasoning); err != nil {
		t.Fatal(err)
	}
	var commentary codex.InputItem
	if err := json.Unmarshal([]byte(`{"id":"msg_1","type":"message","role":"assistant","phase":"commentary","content":[{"type":"output_text","text":"Checking."}]}`), &commentary); err != nil {
		t.Fatal(err)
	}
	call := codex.InputItem{Type: "function_call", CallID: "call_1", Name: "Bash", Arguments: `{"command":"pwd"}`}
	server.recordResponseChain("main-session", previous, responseTrace{
		ResponseID: "resp_1",
		Output:     []codex.InputItem{reasoning, commentary, call},
	})

	current := previous
	current.Input = append(append([]codex.InputItem(nil), previous.Input...),
		codex.InputItem{Type: "message", Role: "assistant", Content: []codex.ContentPart{{Type: "output_text", Text: "Checking."}}},
		call,
		codex.InputItem{Type: "function_call_output", CallID: "call_1", Output: "/repo"},
	)
	used, reason, _, _ := server.applyStatelessReplayDetailed("main-session", &current)
	if !used || reason != "applied" {
		t.Fatalf("stateless replay = %v reason %q", used, reason)
	}
	if len(current.Input) != 5 {
		t.Fatalf("replayed input = %#v", current.Input)
	}
	for index, wantType := range []string{"message", "reasoning", "message", "function_call", "function_call_output"} {
		if current.Input[index].Type != wantType {
			t.Fatalf("input[%d].type = %q, want %q", index, current.Input[index].Type, wantType)
		}
	}
	encoded, err := json.Marshal(current)
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(string(encoded), `"encrypted_content":"opaque-reasoning"`, `"phase":"commentary"`) {
		t.Fatalf("replay lost opaque reasoning or phase: %s", encoded)
	}
	if strings.Contains(string(encoded), openAIReasoningSummarySignatureForTest) {
		t.Fatalf("synthetic Claude thinking signature leaked into OpenAI replay: %s", encoded)
	}
}

const openAIReasoningSummarySignatureForTest = "claudodex_openai_reasoning_summary"

func containsAll(value string, wants ...string) bool {
	for _, want := range wants {
		if !strings.Contains(value, want) {
			return false
		}
	}
	return true
}
