package codex

import (
	"io"
	"strings"
	"testing"
)

type boundedChunkReader struct {
	r    io.Reader
	size int
}

func (r boundedChunkReader) Read(p []byte) (int, error) {
	if len(p) > r.size {
		p = p[:r.size]
	}
	return r.r.Read(p)
}

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

func TestReadSSEParsesEveryByteBoundary(t *testing.T) {
	input := "event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"first\"}\n\n" +
		"event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"second\"}\n\n"
	var events []SSEEvent
	err := ReadSSE(boundedChunkReader{r: strings.NewReader(input), size: 1}, func(event SSEEvent) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("events len = %d, want 2", len(events))
	}
	if got := string(events[0].Data); !strings.Contains(got, `"delta":"first"`) {
		t.Fatalf("first data = %s", got)
	}
	if got := string(events[1].Data); !strings.Contains(got, `"delta":"second"`) {
		t.Fatalf("second data = %s", got)
	}
}
