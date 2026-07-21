package codex

import (
	"encoding/json"
	"testing"
)

func TestInputItemPreservesOpaqueReasoningFields(t *testing.T) {
	raw := []byte(`{"id":"rs_1","type":"reasoning","summary":[{"type":"summary_text","text":"visible"}],"content":null,"encrypted_content":"opaque-secret","provider_extension":{"keep":true}}`)
	var item InputItem
	if err := json.Unmarshal(raw, &item); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if got["id"] != "rs_1" || got["encrypted_content"] != "opaque-secret" {
		t.Fatalf("reasoning item lost identity or encrypted content: %#v", got)
	}
	extension, _ := got["provider_extension"].(map[string]any)
	if extension["keep"] != true {
		t.Fatalf("provider extension was not preserved: %#v", got)
	}
}

func TestInputItemPreservesMessagePhaseAndRichContent(t *testing.T) {
	raw := []byte(`{"id":"msg_1","type":"message","role":"assistant","phase":"commentary","content":[{"type":"output_text","text":"working","annotations":[{"type":"custom","value":1}]}]}`)
	var item InputItem
	if err := json.Unmarshal(raw, &item); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if got["phase"] != "commentary" || got["id"] != "msg_1" {
		t.Fatalf("message phase or id was not preserved: %#v", got)
	}
	content := got["content"].([]any)[0].(map[string]any)
	if len(content["annotations"].([]any)) != 1 {
		t.Fatalf("rich message content was not preserved: %#v", got)
	}
}
