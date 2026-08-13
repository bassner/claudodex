package proxy

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/convert"
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

func TestImplicitResumeTreatsCompactionAsNewReasoningChain(t *testing.T) {
	server := New(Config{})
	previous := codex.Request{
		Model:  "gpt-5.6-terra",
		Input:  []codex.InputItem{{Type: "message", Role: "user", Content: []codex.ContentPart{{Type: "input_text", Text: "original request"}}}},
		Stream: true,
		Store:  false,
	}
	server.recordResponseChain("compacted-session", previous, responseTrace{
		ResponseID: "resp_pre_compaction",
		Output: []codex.InputItem{{
			Type:    "message",
			Role:    "assistant",
			Content: []codex.ContentPart{{Type: "output_text", Text: "pre-compaction answer"}},
		}},
	})

	current := previous
	current.Input = []codex.InputItem{{
		Type:    "message",
		Role:    "user",
		Content: []codex.ContentPart{{Type: "input_text", Text: "The conversation was compacted into this new summary."}},
	}}
	original := append([]codex.InputItem(nil), current.Input...)

	used, reason, _, _ := server.applyImplicitResumeDetailed("compacted-session", &current)
	if used || reason != "input_prefix_mismatch" {
		t.Fatalf("compacted chain resume = %v reason %q", used, reason)
	}
	if current.PreviousResponseID != "" {
		t.Fatalf("compacted chain reused previous_response_id %q", current.PreviousResponseID)
	}
	if !inputHasPrefix(current.Input, original) || len(current.Input) != len(original) {
		t.Fatalf("compacted input was rewritten with pre-compaction state: %#v", current.Input)
	}
}

func TestResponseTurnIdentityContinuesToolsAndPropagatesToSubagents(t *testing.T) {
	server := New(Config{})
	parentRequest := codex.Request{
		Input: []codex.InputItem{{Type: "message", Role: "user", Content: []codex.ContentPart{{Type: "input_text", Text: "delegate"}}}},
		ClientMetadata: map[string]string{
			"turn_id":      "turn-parent",
			"root_turn_id": "turn-root",
		},
	}
	server.recordResponseChain("parent-thread", parentRequest, responseTrace{
		ResponseID: "resp-parent",
		Output:     []codex.InputItem{{Type: "function_call", CallID: "call-agent", Name: "Agent", Arguments: `{}`}},
	})

	continuation := codex.Request{Input: []codex.InputItem{
		parentRequest.Input[0],
		{Type: "function_call", CallID: "call-agent", Name: "Agent", Arguments: `{}`},
		{Type: "function_call_output", CallID: "call-agent", Output: "done"},
	}}
	continuedRoute := codex.MaterializeRoute(codex.Route{SessionID: "parent-thread", ThreadID: "parent-thread"})
	server.inheritResponseTurnIdentity("parent-thread", &continuedRoute, continuation)
	if continuedRoute.TurnID != "turn-parent" || continuedRoute.RootTurnID != "turn-root" {
		t.Fatalf("tool continuation identity = turn %q root %q", continuedRoute.TurnID, continuedRoute.RootTurnID)
	}

	childRoute := codex.MaterializeRoute(codex.Route{SessionID: "child-thread", ThreadID: "child-thread", ParentThreadID: "parent-thread", Subagent: "collab_spawn"})
	childTurnID := childRoute.TurnID
	server.inheritResponseTurnIdentity("child-thread", &childRoute, codex.Request{})
	if childRoute.TurnID != childTurnID || childRoute.RootTurnID != "turn-root" {
		t.Fatalf("child identity = turn %q root %q", childRoute.TurnID, childRoute.RootTurnID)
	}
}

func TestResponseTurnIdentityStartsNewRootAfterFinalMessage(t *testing.T) {
	server := New(Config{})
	server.recordResponseChain("thread-1", codex.Request{ClientMetadata: map[string]string{
		"turn_id": "turn-old", "root_turn_id": "root-old",
	}}, responseTrace{ResponseID: "resp-old", Output: []codex.InputItem{{Type: "message", Role: "assistant", Content: []codex.ContentPart{{Type: "output_text", Text: "done"}}}}})
	route := codex.MaterializeRoute(codex.Route{SessionID: "thread-1", ThreadID: "thread-1"})
	newRoot := route.RootTurnID
	server.inheritResponseTurnIdentity("thread-1", &route, codex.Request{Input: []codex.InputItem{{Type: "message", Role: "user"}}})
	if route.RootTurnID != newRoot || route.RootTurnID == "root-old" {
		t.Fatalf("new top-level turn root = %q", route.RootTurnID)
	}
}

func TestPreviousResponseUsageSurvivesChainReset(t *testing.T) {
	server := New(Config{})
	want := convert.Usage{
		InputTokens:          32_000,
		CacheReadInputTokens: 18_000,
		OutputTokens:         500,
	}
	server.recordResponseUsage("session-1", want)
	server.clearImplicitResume("session-1")
	if got := server.previousResponseUsage("session-1"); got != want {
		t.Fatalf("previous response usage = %#v, want %#v", got, want)
	}
}

func TestEstimatedNextInputUsageAddsOnlyAppendedInput(t *testing.T) {
	server := New(Config{})
	server.recordResponseUsage("session-1", convert.Usage{
		InputTokens:          32_000,
		CacheReadInputTokens: 18_000,
		OutputTokens:         500,
	})
	appended := []codex.InputItem{{
		Type:    "message",
		Role:    "user",
		Content: []codex.ContentPart{{Type: "input_text", Text: "new user input"}},
	}}
	increment := estimateCodexInputItems(appended)
	got := server.estimatedNextInputUsage("session-1", appended, true)
	if got.InputTokens != 32_000+500+increment ||
		got.CacheReadInputTokens != 18_000 ||
		got.OutputTokens != 0 {
		t.Fatalf("next input usage = %#v, want prior input plus prior output and appended estimate %d", got, increment)
	}

	if unknown := server.estimatedNextInputUsage("session-1", appended, false); unknown != (convert.Usage{}) {
		t.Fatalf("unknown incremental suffix usage = %#v, want full-request fallback", unknown)
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
