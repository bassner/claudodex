package proxy

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/codex"
)

func TestImplicitResumeDoesNotAccumulateDeveloperInstructions(t *testing.T) {
	server := New(Config{})
	full := codex.Request{
		Model: "gpt-6.1-sol", Stream: true,
		Instructions: strings.Repeat("Stable system guidance and pre-compaction file context.\n", 2000),
		Input:        []codex.InputItem{{Type: "message", Role: "user", Content: []codex.ContentPart{{Type: "input_text", Text: "Read twenty tiny files"}}}},
	}
	developerMessages := 0
	countGuidance := func(request codex.Request) {
		t.Helper()
		data, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		var wire struct{ Input []codex.InputItem }
		if err := json.Unmarshal(data, &wire); err != nil {
			t.Fatal(err)
		}
		for _, item := range wire.Input {
			if item.Role == "developer" {
				developerMessages++
			}
		}
	}
	countGuidance(full)
	for index := range 20 {
		callID := fmt.Sprintf("call_%d", index)
		trace := responseTrace{ResponseID: fmt.Sprintf("resp_%d", index), Output: []codex.InputItem{{Type: "function_call", CallID: callID, Name: "Read", Arguments: `{"file_path":"tiny.txt"}`}}}
		server.recordResponseChain("guidance-loop", full, trace)
		if server.chains["guidance-loop"].Request.Instructions != full.Instructions {
			t.Fatal("recorded chain lost full developer guidance")
		}
		full.Input = append(append(append([]codex.InputItem(nil), full.Input...), trace.Output...), codex.InputItem{Type: "function_call_output", CallID: callID, Output: "small result"})
		delta := full
		if used, reason, _, _ := server.applyImplicitResumeDetailed("guidance-loop", &delta); !used {
			t.Fatalf("round %d did not resume: %s", index, reason)
		}
		countGuidance(delta)
		if developerMessages != 1 {
			t.Fatalf("round %d appended the stable developer guidance again (%d copies in provider history)", index, developerMessages)
		}
		if full.Instructions == "" {
			t.Fatal("full request lost guidance needed for reconnect and replay")
		}
		if server.chains["guidance-loop"].Request.Instructions != full.Instructions {
			t.Fatal("delta request cleared the recorded chain's guidance")
		}
	}
}

func TestImplicitResumeMismatchRetainsFullDeveloperInstructions(t *testing.T) {
	server := New(Config{})
	previous := codex.Request{Model: "gpt-6.1-sol", Instructions: "stable guidance", Input: []codex.InputItem{{Type: "message", Role: "user", Content: []codex.ContentPart{{Type: "input_text", Text: "original turn"}}}}}
	server.recordResponseChain("mismatch", previous, responseTrace{ResponseID: "resp_old", Output: []codex.InputItem{{Type: "message", Role: "assistant", Content: []codex.ContentPart{{Type: "output_text", Text: "old answer"}}}}})
	current := previous
	current.Input = []codex.InputItem{{Type: "message", Role: "user", Content: []codex.ContentPart{{Type: "input_text", Text: "new compacted summary"}}}}
	if used, _, _, _ := server.applyImplicitResumeDetailed("mismatch", &current); used {
		t.Fatal("mismatching context reused the previous response")
	}
	encoded, err := json.Marshal(current)
	if err != nil {
		t.Fatal(err)
	}
	if current.PreviousResponseID != "" || strings.Count(string(encoded), `"role":"developer"`) != 1 || !strings.Contains(string(encoded), "stable guidance") {
		t.Fatalf("fresh context lost developer guidance: %s", encoded)
	}
}

func TestImplicitResumeChangedInstructionsRequireFullContext(t *testing.T) {
	server := New(Config{})
	previous := codex.Request{Model: "gpt-6.1-sol", Instructions: "old guidance", Input: []codex.InputItem{{Type: "message", Role: "user", Content: []codex.ContentPart{{Type: "input_text", Text: "read"}}}}}
	trace := responseTrace{ResponseID: "resp_previous", Output: []codex.InputItem{{Type: "function_call", CallID: "call_read", Name: "Read", Arguments: `{}`}}}
	server.recordResponseChain("changed-guidance", previous, trace)
	current := previous
	current.Instructions = "new guidance after loading a skill"
	current.Input = append(append(append([]codex.InputItem(nil), previous.Input...), trace.Output...), codex.InputItem{Type: "function_call_output", CallID: "call_read", Output: "done"})
	if used, reason, _, _ := server.applyImplicitResumeDetailed("changed-guidance", &current); used || reason != "instructions_changed" {
		t.Fatalf("changed guidance reused the old provider context: used=%t reason=%s", used, reason)
	}
	if current.PreviousResponseID != "" || current.Instructions != "new guidance after loading a skill" || len(current.Input) != 3 {
		t.Fatalf("full fallback lost updated guidance or history: %#v", current)
	}
	if used, reason, _, _ := server.applyStatelessReplayDetailed("changed-guidance", &current); !used {
		t.Fatalf("updated guidance could not retain stateless response replay: %s", reason)
	}
	if current.Instructions != "new guidance after loading a skill" || current.PreviousResponseID != "" {
		t.Fatal("stateless replay replaced the current guidance")
	}
}
