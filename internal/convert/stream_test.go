package convert

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
)

func TestStreamReducerStreamsReasoningSummaryAsThinking(t *testing.T) {
	reducer := NewStreamReducer("msg_1", "claude-sonnet-4-6")
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.created","response":{"id":"resp_1"}}`,
		`{"type":"response.reasoning_summary_text.delta","summary_index":0,"delta":"summary"}`,
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
	if len(content) != 2 || content[0]["type"] != "thinking" || content[0]["thinking"] != "summary" || content[0]["signature"] != openAIReasoningSummarySignature || content[1]["text"] != "hello" {
		t.Fatalf("content = %#v", content)
	}
	usage := message["usage"].(Usage)
	if usage.InputTokens != 7 || usage.CacheReadInputTokens != 3 || usage.OutputTokens != 2 {
		t.Fatalf("usage = %#v", usage)
	}
}

func TestStreamReducerStreamsEveryIncrementalTextDeltaBeforeDone(t *testing.T) {
	reducer := NewStreamReducer("msg_incremental", "claude-sonnet-4-6")
	inputs := []string{
		`{"type":"response.created","response":{"id":"resp_incremental"}}`,
		`{"type":"response.output_item.added","item":{"type":"message","id":"msg_item"}}`,
		`{"type":"response.output_text.delta","delta":"first "}`,
		`{"type":"response.output_text.delta","delta":"second "}`,
		`{"type":"response.output_text.delta","delta":"third"}`,
	}
	var textDeltas []string
	for _, input := range inputs {
		events, err := reducer.Reduce(json.RawMessage(input))
		if err != nil {
			t.Fatal(err)
		}
		for _, event := range events {
			delta, _ := event.Data["delta"].(map[string]any)
			if delta["type"] == "text_delta" {
				textDeltas = append(textDeltas, delta["text"].(string))
			}
			if event.Event == "message_delta" || event.Event == "message_stop" {
				t.Fatalf("incremental text became terminal before response.completed: %#v", event)
			}
		}
	}
	if got := strings.Join(textDeltas, "|"); got != "first |second |third" {
		t.Fatalf("incremental text deltas = %q", got)
	}
	if reducer.Done() {
		t.Fatal("reducer completed before terminal upstream event")
	}
}

func TestStreamReducerKeepsDelayedSafetyBufferingNonterminal(t *testing.T) {
	reducer := NewStreamReducer("msg_safety", "claude-sonnet-4-6")
	inputs := []string{
		`{"type":"response.in_progress","response":{"id":"resp_safety","status":"in_progress"}}`,
		`{"type":"response.safety_buffering","status":"in_progress","safety":{"status":"buffering"}}`,
		`{"type":"response.status","status":"in_progress","message":"delayed safety check"}`,
	}
	for _, input := range inputs {
		events, err := reducer.Reduce(json.RawMessage(input))
		if err != nil {
			t.Fatal(err)
		}
		for _, event := range events {
			if event.Event == "content_block_start" || event.Event == "content_block_delta" ||
				event.Event == "content_block_stop" || event.Event == "message_delta" ||
				event.Event == "message_stop" || event.Event == "error" {
				t.Fatalf("nonterminal safety metadata became visible or terminal: %#v", event)
			}
		}
		if reducer.Done() || reducer.Failed() {
			t.Fatalf("nonterminal safety metadata ended request: done=%v failed=%v", reducer.Done(), reducer.Failed())
		}
	}

	events, err := reducer.Reduce(json.RawMessage(`{"type":"response.output_text.delta","delta":"safe output"}`))
	if err != nil {
		t.Fatal(err)
	}
	var gotText string
	for _, event := range events {
		delta, _ := event.Data["delta"].(map[string]any)
		if delta["type"] == "text_delta" {
			gotText, _ = delta["text"].(string)
		}
	}
	if gotText != "safe output" {
		t.Fatalf("post-safety text delta = %q", gotText)
	}
	if reducer.Done() {
		t.Fatal("text delta completed delayed safety response")
	}
}

func TestStreamReducerSeparatesAnnouncedReasoningSummaryParts(t *testing.T) {
	reducer := NewStreamReducer("msg_1", "claude-sonnet-4-6")
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.created","response":{"id":"resp_1"}}`,
		`{"type":"response.reasoning_summary_part.added","summary_index":0}`,
		`{"type":"response.reasoning_summary_text.delta","summary_index":0,"delta":"Planning installer security"}`,
		`{"type":"response.reasoning_summary_part.added","summary_index":1}`,
		`{"type":"response.reasoning_summary_text.delta","summary_index":1,"delta":"Evaluating filesystem protections"}`,
		`{"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"answer"}]}}`,
		`{"type":"response.completed","response":{"usage":{"input_tokens":1,"output_tokens":2}}}`,
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
	if len(content) != 3 || content[0]["thinking"] != "Planning installer security" || content[1]["thinking"] != "Evaluating filesystem protections" || content[2]["text"] != "answer" {
		t.Fatalf("content = %#v, want two thinking blocks followed by text", content)
	}
}

func TestStreamReducerSeparatesAdjacentBoldReasoningSections(t *testing.T) {
	reducer := NewStreamReducer("msg_1", "claude-sonnet-4-6")
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.created","response":{"id":"resp_1"}}`,
		`{"type":"response.reasoning_summary_text.delta","summary_index":0,"delta":"**First section**"}`,
		`{"type":"response.reasoning_summary_text.delta","summary_index":0,"delta":"**Second section****Third section**"}`,
		`{"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"answer"}]}}`,
		`{"type":"response.completed","response":{"usage":{"input_tokens":1,"output_tokens":2}}}`,
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
	if len(content) != 4 || content[0]["thinking"] != "**First section**" || content[1]["thinking"] != "**Second section**" || content[2]["thinking"] != "**Third section**" || content[3]["text"] != "answer" {
		t.Fatalf("content = %#v, want three thinking blocks followed by text", content)
	}
}

func TestStreamReducerSeparatesTitledSectionsWithinOneSummaryPart(t *testing.T) {
	reducer := NewStreamReducer("msg_1", "claude-sonnet-4-6")
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.created","response":{"id":"resp_1"}}`,
		`{"type":"response.reasoning_summary_text.delta","summary_index":0,"delta":"**First section**\n\nDetails for the first section.\n\n**Second section**\n\nDetails for the second section."}`,
		`{"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"answer"}]}}`,
		`{"type":"response.completed","response":{"usage":{"input_tokens":1,"output_tokens":2}}}`,
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
	if len(content) != 3 || content[0]["thinking"] != "**First section**\n\nDetails for the first section.\n\n" || content[1]["thinking"] != "**Second section**\n\nDetails for the second section." || content[2]["text"] != "answer" {
		t.Fatalf("content = %#v, want titled sections in separate thinking blocks", content)
	}
}

func TestStreamReducerSeparatesReasoningSectionsAtEveryDeltaBoundary(t *testing.T) {
	for _, summary := range []string{
		"**First section****Second section**",
		"**First section**\n\n**Second section**",
	} {
		for split := 1; split < len(summary); split++ {
			t.Run(fmt.Sprintf("%q/byte_%d", summary, split), func(t *testing.T) {
				reducer := NewStreamReducer("msg_1", "claude-sonnet-4-6")
				var events []AnthropicSSE
				for _, raw := range []string{
					`{"type":"response.created","response":{"id":"resp_1"}}`,
					fmt.Sprintf(`{"type":"response.reasoning_summary_text.delta","summary_index":0,"delta":%q}`, summary[:split]),
					fmt.Sprintf(`{"type":"response.reasoning_summary_text.delta","summary_index":0,"delta":%q}`, summary[split:]),
					`{"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"answer"}]}}`,
					`{"type":"response.completed","response":{"usage":{"input_tokens":1,"output_tokens":2}}}`,
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
				if len(content) != 3 {
					t.Fatalf("content = %#v, want two thinking blocks followed by text", content)
				}
				got := content[0]["thinking"].(string) + content[1]["thinking"].(string)
				if got != summary || content[2]["text"] != "answer" {
					t.Fatalf("content = %#v, reconstructed summary %q, want %q", content, got, summary)
				}
			})
		}
	}
}

func TestStreamReducerUsesCompletedReasoningItemWhenSummaryEventsAreAbsent(t *testing.T) {
	reducer := NewStreamReducer("msg_1", "claude-sonnet-4-6")
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.created","response":{"id":"resp_1"}}`,
		`{"type":"response.output_item.done","item":{"type":"reasoning","summary":[{"type":"summary_text","text":"first"},{"type":"summary_text","text":"second"}]}}`,
		`{"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"answer"}]}}`,
		`{"type":"response.completed","response":{"usage":{"input_tokens":1,"output_tokens":2}}}`,
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
	if len(content) != 3 || content[0]["thinking"] != "first" || content[1]["thinking"] != "second" || content[2]["text"] != "answer" {
		t.Fatalf("content = %#v", content)
	}
}

func TestStreamReducerDoesNotDuplicateReasoningSummaryDoneFromCompletedItem(t *testing.T) {
	reducer := NewStreamReducer("msg_1", "claude-sonnet-4-6")
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.created","response":{"id":"resp_1"}}`,
		`{"type":"response.reasoning_summary_text.done","item_id":"reasoning_1","summary_index":0,"text":"summary"}`,
		`{"type":"response.output_item.done","item":{"type":"reasoning","summary":[{"type":"summary_text","text":"summary"}]}}`,
		`{"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"answer"}]}}`,
		`{"type":"response.completed","response":{"usage":{"input_tokens":1,"output_tokens":2}}}`,
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
	if len(content) != 2 || content[0]["thinking"] != "summary" || content[1]["text"] != "answer" {
		t.Fatalf("content = %#v", content)
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

func TestStreamReducerConvertsWebSearchCallAndCitations(t *testing.T) {
	reducer := NewStreamReducer("msg_1", "claude-haiku-4-5")
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"web_search_call","id":"ws_1","status":"in_progress"}}`,
		`{"type":"response.output_item.done","output_index":0,"item":{"type":"web_search_call","id":"ws_1","status":"completed","action":{"type":"search","query":"official Go website","sources":[{"type":"url","title":"The Go Programming Language","url":"https://go.dev/"}]}}}`,
		`{"type":"response.output_item.added","output_index":1,"item":{"type":"message","id":"msg"}}`,
		`{"type":"response.output_text.delta","output_index":1,"delta":"The official site is go.dev."}`,
		`{"type":"response.output_text.annotation.added","output_index":1,"annotation":{"type":"url_citation","title":"The Go Programming Language","url":"https://go.dev/"}}`,
		`{"type":"response.output_item.done","output_index":1,"item":{"type":"message","id":"msg","content":[{"type":"output_text","text":"The official site is go.dev.","annotations":[{"type":"url_citation","title":"The Go Programming Language","url":"https://go.dev/"}]}]}}`,
		`{"type":"response.completed","response":{"stop_reason":"completed","usage":{"input_tokens":10,"output_tokens":4}}}`,
	} {
		next, err := reducer.Reduce(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, next...)
	}
	message, errEvent := AssembleMessage(events, "", "claude-haiku-4-5")
	if errEvent != nil {
		t.Fatalf("unexpected error event: %#v", errEvent)
	}
	content := message["content"].([]map[string]any)
	if len(content) != 3 {
		t.Fatalf("content = %#v", content)
	}
	serverUse := content[0]
	input, _ := serverUse["input"].(map[string]any)
	if serverUse["type"] != "server_tool_use" || serverUse["id"] != "srvtoolu_1" ||
		serverUse["name"] != "web_search" || input["query"] != "official Go website" {
		t.Fatalf("server tool use = %#v", serverUse)
	}
	result := content[1]
	if result["type"] != "web_search_tool_result" || result["tool_use_id"] != "srvtoolu_1" {
		t.Fatalf("web search result = %#v", result)
	}
	hits, ok := result["content"].([]map[string]any)
	if !ok || len(hits) != 1 || hits[0]["title"] != "The Go Programming Language" || hits[0]["url"] != "https://go.dev/" {
		t.Fatalf("hits = %#v", result["content"])
	}
	if content[2]["type"] != "text" || content[2]["text"] != "The official site is go.dev." {
		t.Fatalf("text = %#v", content[2])
	}
	if message["stop_reason"] != "end_turn" {
		t.Fatalf("stop_reason = %#v", message["stop_reason"])
	}
}

func TestStreamReducerAssociatesSourcesWithMultiplePendingWebSearches(t *testing.T) {
	reducer := NewStreamReducer("msg_1", "claude-haiku-4-5")
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.output_item.done","item":{"type":"web_search_call","id":"ws_1","action":{"type":"search","query":"first","sources":[{"type":"url","title":"First","url":"https://first.example/"}]}}}`,
		`{"type":"response.output_item.done","item":{"type":"web_search_call","id":"ws_2","action":{"type":"search","query":"second","sources":[{"type":"url","title":"Second","url":"https://second.example/"}]}}}`,
		`{"type":"response.output_text.delta","delta":"combined answer"}`,
		`{"type":"response.completed","response":{"stop_reason":"completed"}}`,
	} {
		next, err := reducer.Reduce(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, next...)
	}
	message, errEvent := AssembleMessage(events, "", "claude-haiku-4-5")
	if errEvent != nil {
		t.Fatalf("unexpected error event: %#v", errEvent)
	}
	content := message["content"].([]map[string]any)
	if len(content) != 5 {
		t.Fatalf("content = %#v", content)
	}
	firstHits := content[2]["content"].([]map[string]any)
	secondHits := content[3]["content"].([]map[string]any)
	if len(firstHits) != 1 || firstHits[0]["url"] != "https://first.example/" {
		t.Fatalf("first hits = %#v", firstHits)
	}
	if len(secondHits) != 1 || secondHits[0]["url"] != "https://second.example/" {
		t.Fatalf("second hits = %#v", secondHits)
	}
	if content[4]["type"] != "text" || content[4]["text"] != "combined answer" {
		t.Fatalf("buffered completion text = %#v", content[4])
	}
}

func TestStreamReducerFlushesWebSearchBeforeClientToolWithoutInterleaving(t *testing.T) {
	reducer := NewStreamReducer("msg_1", "claude-haiku-4-5")
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.output_item.done","output_index":0,"item":{"type":"web_search_call","id":"ws_1","action":{"type":"search","query":"first","sources":[{"type":"url","url":"https://first.example/"}]}}}`,
		`{"type":"response.output_text.delta","delta":"found it"}`,
		`{"type":"response.output_item.added","output_index":1,"item":{"type":"function_call","call_id":"call_1","name":"Read"}}`,
		`{"type":"response.function_call_arguments.done","output_index":1,"arguments":"{\"file_path\":\"a.go\"}"}`,
		`{"type":"response.output_item.done","output_index":1,"item":{"type":"function_call","call_id":"call_1","name":"Read"}}`,
		`{"type":"response.completed","response":{"stop_reason":"tool_calls"}}`,
	} {
		next, err := reducer.Reduce(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, next...)
	}
	active := -1
	for _, event := range events {
		switch event.Event {
		case "content_block_start":
			index := intField(event.Data["index"], -1)
			if active >= 0 {
				t.Fatalf("nested content block start %d while %d active: %#v", index, active, events)
			}
			active = index
		case "content_block_stop":
			index := intField(event.Data["index"], -1)
			if active != index {
				t.Fatalf("content block stop %d while %d active: %#v", index, active, events)
			}
			active = -1
		}
	}
	if active != -1 {
		t.Fatalf("unclosed content block %d", active)
	}
}

func TestStreamReducerEmitsEmptyWebSearchResultsAndMaxUsesError(t *testing.T) {
	reducer := NewStreamReducerWithOptions("msg_1", "claude-haiku-4-5", StreamReducerOptions{
		WebSearchMaxUses: 1,
	})
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.output_item.done","item":{"type":"web_search_call","id":"ws_1","action":{"type":"search","query":"empty","sources":[]}}}`,
		`{"type":"response.output_item.done","item":{"type":"message","content":[]}}`,
		`{"type":"response.output_item.done","item":{"type":"web_search_call","id":"ws_2","action":{"type":"search","query":"too many","sources":[]}}}`,
		`{"type":"response.completed","response":{"stop_reason":"completed"}}`,
	} {
		next, err := reducer.Reduce(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, next...)
	}
	message, errEvent := AssembleMessage(events, "", "claude-haiku-4-5")
	if errEvent != nil {
		t.Fatalf("unexpected error event: %#v", errEvent)
	}
	content := message["content"].([]map[string]any)
	first, ok := content[1]["content"].([]map[string]any)
	if !ok || len(first) != 0 {
		t.Fatalf("empty successful content = %#v, want non-nil empty list", content[1]["content"])
	}
	errorContent, _ := content[3]["content"].(map[string]any)
	if errorContent["error_code"] != "max_uses_exceeded" {
		t.Fatalf("max uses error = %#v", content[3])
	}
}

func TestStreamReducerKeepsWebSearchCyclesIsolatedAndSuppressesOpenPageActions(t *testing.T) {
	reducer := NewStreamReducer("msg_1", "claude-haiku-4-5")
	var events []AnthropicSSE
	for _, raw := range []string{
		`{"type":"response.output_item.done","item":{"type":"web_search_call","id":"ws_search_1","status":"completed","action":{"type":"search","query":"first"}}}`,
		`{"type":"response.output_item.done","item":{"type":"web_search_call","id":"ws_open_1","status":"completed","action":{"type":"open_page","url":"https://first.example/"}}}`,
		`{"type":"response.output_text.annotation.added","annotation":{"type":"url_citation","title":"First","url":"https://first.example/"}}`,
		`{"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"first answer"}]}}`,
		`{"type":"response.output_item.done","item":{"type":"web_search_call","id":"ws_search_2","status":"completed","action":{"type":"search","query":"second"}}}`,
		`{"type":"response.output_text.annotation.added","annotation":{"type":"url_citation","title":"Second","url":"https://second.example/"}}`,
		`{"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"second answer"}]}}`,
		`{"type":"response.completed","response":{"stop_reason":"completed"}}`,
	} {
		next, err := reducer.Reduce(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, next...)
	}
	message, errEvent := AssembleMessage(events, "", "claude-haiku-4-5")
	if errEvent != nil {
		t.Fatalf("unexpected error event: %#v", errEvent)
	}
	content := message["content"].([]map[string]any)
	if len(content) != 6 {
		t.Fatalf("content = %#v", content)
	}
	for index, wantType := range []string{
		"server_tool_use", "web_search_tool_result", "text",
		"server_tool_use", "web_search_tool_result", "text",
	} {
		if content[index]["type"] != wantType {
			t.Fatalf("content[%d] = %#v, want %s", index, content[index], wantType)
		}
	}
	firstHits := content[1]["content"].([]map[string]any)
	secondHits := content[4]["content"].([]map[string]any)
	if len(firstHits) != 1 || firstHits[0]["url"] != "https://first.example/" {
		t.Fatalf("first hits = %#v", firstHits)
	}
	if len(secondHits) != 1 || secondHits[0]["url"] != "https://second.example/" {
		t.Fatalf("second hits = %#v", secondHits)
	}
	for _, block := range content {
		input, _ := block["input"].(map[string]any)
		if input["query"] == "" {
			t.Fatalf("empty-query server tool block emitted: %#v", block)
		}
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

func TestStreamReducerKeepsPreviousAuthoritativeUsageUntilCompletion(t *testing.T) {
	reducer := NewStreamReducerWithOptions("msg_1", "claude-opus-5", StreamReducerOptions{
		FallbackInputTokens: 80_000,
		InitialInputUsage: Usage{
			InputTokens:          32_000,
			CacheReadInputTokens: 18_000,
			OutputTokens:         500,
		},
	})

	start := reducer.usageForStart(map[string]any{})
	if start.InputTokens != 32_000 || start.CacheReadInputTokens != 18_000 || start.OutputTokens != 0 {
		t.Fatalf("start usage = %#v, want previous authoritative 50k input with zero output", start)
	}

	finish := reducer.usageForFinish(map[string]any{
		"response": map[string]any{
			"usage": map[string]any{
				"input_tokens":            33_000,
				"cache_read_input_tokens": 18_000,
				"output_tokens":           600,
			},
		},
	})
	if finish.InputTokens != 33_000 || finish.CacheReadInputTokens != 18_000 || finish.OutputTokens != 600 {
		t.Fatalf("finish usage = %#v, want new authoritative 51k input", finish)
	}
}

func TestStreamReducerTreatsUpstreamInputUsageAsAuthoritative(t *testing.T) {
	tests := []struct {
		name           string
		fallback       int
		input          int
		cacheRead      int
		wantInput      int
		wantTotalInput int
	}{
		{
			name:           "upstream usage below local fallback",
			fallback:       61_962,
			input:          1_000,
			wantInput:      1_000,
			wantTotalInput: 1_000,
		},
		{
			name:           "upstream cache accounting is preserved",
			fallback:       258_400,
			input:          35_614,
			cacheRead:      185_344,
			wantInput:      35_614,
			wantTotalInput: 220_958,
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

func TestStreamReducerMapsAgentModelAliasToConfiguredCodexTier(t *testing.T) {
	reducer := NewStreamReducerWithOptions("msg_1", "gpt-5.5", StreamReducerOptions{
		AgentModels: modelconfig.Config{
			Opus:   "gpt-opus-next",
			Sonnet: "gpt-sonnet-next",
			Haiku:  "gpt-haiku-next",
		},
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
	if input["model"] != "gpt-sonnet-next" {
		t.Fatalf("agent model = %#v, input = %#v", input["model"], input)
	}
}

func TestStreamReducerFallsBackRetiredAgentModelAliasToSol(t *testing.T) {
	reducer := NewStreamReducerWithOptions("msg_1", "gpt-5.6-sol", StreamReducerOptions{
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
		`{"type":"response.function_call_arguments.done","output_index":0,"arguments":"{\"description\":\"Review plan\",\"prompt\":\"review it\",\"model\":\"fable\"}"}`,
		`{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","call_id":"call_agent","name":"Agent"}}`,
		`{"type":"response.completed","response":{"stop_reason":"tool_calls"}}`,
	} {
		next, err := reducer.Reduce(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, next...)
	}
	message, errEvent := AssembleMessage(events, "", "gpt-5.6-sol")
	if errEvent != nil {
		t.Fatalf("unexpected error event: %#v", errEvent)
	}
	content := message["content"].([]map[string]any)
	input := content[0]["input"].(map[string]any)
	if input["model"] != "gpt-5.6-sol" {
		t.Fatalf("retired agent model did not fall back to sol: %#v", input)
	}
}

func TestNormalizeClaudeCodeToolArgsMapsAllAgentAliasesToCodexTiers(t *testing.T) {
	models := modelconfig.Config{
		Opus:   "gpt-sol-custom",
		Sonnet: "gpt-terra-custom",
		Haiku:  "gpt-luna-custom",
	}
	tests := map[string]string{
		"":                  "gpt-sol-custom",
		"inherit":           "gpt-sol-custom",
		"fable":             "gpt-sol-custom",
		"claude-fable-5":    "gpt-sol-custom",
		"opus":              "gpt-sol-custom",
		"claude-opus-5":     "gpt-sol-custom",
		"sonnet":            "gpt-terra-custom",
		"claude-sonnet-4-6": "gpt-terra-custom",
		"haiku":             "gpt-luna-custom",
		"claude-haiku-4-5":  "gpt-luna-custom",
	}
	for input, want := range tests {
		got := normalizeClaudeCodeToolArgs("Agent", map[string]any{"model": input}, models, "")
		if got["model"] != want {
			t.Errorf("model %q mapped to %#v, want %q", input, got["model"], want)
		}
	}
}

func TestNormalizeClaudeCodeToolArgsRewritesPlanMutationToAssignedFile(t *testing.T) {
	const assigned = "/Users/test/.claudodex/claude-config/plans/session-slug.md"
	for _, toolName := range []string{"Write", "Edit"} {
		got := normalizeClaudeCodeToolArgs(toolName, map[string]any{
			"file_path": "/Users/test/.claude/plans/descriptive-name.md",
			"content":   "plan",
		}, modelconfig.Config{}, assigned)
		if got["file_path"] != assigned {
			t.Errorf("%s plan path = %#v, want %q", toolName, got["file_path"], assigned)
		}
	}

	got := normalizeClaudeCodeToolArgs("Write", map[string]any{
		"file_path": "/repo/docs/plan.md",
		"content":   "documentation",
	}, modelconfig.Config{}, assigned)
	if got["file_path"] != "/repo/docs/plan.md" {
		t.Fatalf("ordinary write path was rewritten: %#v", got)
	}
}

func TestStreamReducerDefaultsEmptyAgentModelToSol(t *testing.T) {
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
	if input["model"] != "gpt-5.6-sol" {
		t.Fatalf("empty agent model did not default to sol: %#v", input)
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
	want := strings.TrimRight(strings.ReplaceAll(string(wantBytes), "\r\n", "\n"), "\n") + "\n\n"
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
