package codex

import (
	"strings"
	"testing"
)

func TestReadSSEParsesCRLFAndMultilineData(t *testing.T) {
	var events []SSEEvent
	input := "event: response.output_text.delta\r\ndata: {\"type\":\"response.output_text.delta\",\r\ndata: \"delta\":\"hi\"}\r\n\r\n"
	err := ReadSSE(strings.NewReader(input), func(event SSEEvent) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("events len = %d", len(events))
	}
	if events[0].Event != "response.output_text.delta" {
		t.Fatalf("event = %q", events[0].Event)
	}
	if !strings.Contains(string(events[0].Data), `"delta":"hi"`) {
		t.Fatalf("data = %s", events[0].Data)
	}
}
