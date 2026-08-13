package codex

import (
	"encoding/json"
	"strings"
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

func TestInputItemCreationTimePreservesExistingAndOpaqueMetadata(t *testing.T) {
	item := InputItem{Type: "message", Role: "user"}
	item.SetCreateTimeIfMissing(1234.5)
	item.SetCreateTimeIfMissing(6789.0)
	if got := item.InternalChatMessageMetadataPassthrough; got == nil || got.CreateTime != 1234.5 {
		t.Fatalf("locally stamped metadata = %#v", got)
	}
	encoded, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"internal_chat_message_metadata_passthrough":{"create_time":1234.5}`) {
		t.Fatalf("creation timestamp missing from encoded item: %s", encoded)
	}

	var upstream InputItem
	if err := json.Unmarshal([]byte(`{"type":"message","role":"user","content":[],"internal_chat_message_metadata_passthrough":{"create_time":42.25,"future":"keep"}}`), &upstream); err != nil {
		t.Fatal(err)
	}
	upstream.SetCreateTimeIfMissing(99)
	encoded, err = json.Marshal(upstream)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"create_time":42.25`) || !strings.Contains(string(encoded), `"future":"keep"`) {
		t.Fatalf("upstream metadata was not preserved: %s", encoded)
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
