package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/convert"
)

func TestStreamTraceRecordsContinuationAndFinalStop(t *testing.T) {
	for _, stream := range []bool{true, false} {
		t.Run("stream="+strconv.FormatBool(stream), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "trace.jsonl")
			t.Setenv(proxyTraceEnv, path)
			t.Setenv(proxyTraceIdleIntervalEnv, "0")
			server := New(Config{})
			segment := func(id, phase string, endTurn bool) string {
				item := map[string]any{
					"type": "message", "role": "assistant", "phase": phase,
					"content": []any{map[string]any{"type": "output_text", "text": "private-response-text"}},
				}
				itemEvent, _ := json.Marshal(map[string]any{"type": "response.output_item.done", "output_index": 0, "item": item})
				completion, _ := json.Marshal(map[string]any{
					"type": "response.completed",
					"response": map[string]any{
						"id": id, "status": "completed", "end_turn": endTurn,
						"output":   []any{item, map[string]any{"type": "reasoning", "encrypted_content": "private-reasoning"}},
						"metadata": map[string]any{"secret": "private-metadata"},
					},
				})
				return "event: response.output_item.done\ndata: " + string(itemEvent) + "\n\nevent: response.completed\ndata: " + string(completion) + "\n\n"
			}
			continuations := 0
			continueResponse := func(request codex.Request) (*http.Response, error) {
				continuations++
				return &http.Response{Body: io.NopCloser(strings.NewReader(segment("resp-final", "final_answer", true)))}, nil
			}
			body := strings.NewReader(segment("resp-partial", "commentary", false))
			writer := httptest.NewRecorder()
			var err error
			if stream {
				err = server.streamAnthropicWithSchemas(writer, body, nil, "test-model", nil, "", 0, 0, convert.Usage{}, "", codex.Request{}, nil, false, continueResponse)
			} else {
				err = server.writeNonStreamingMessageWithSchemas(writer, body, nil, "test-model", nil, "", 0, 0, convert.Usage{}, "", codex.Request{}, nil, continueResponse)
			}
			if err != nil || continuations != 1 || !strings.Contains(writer.Body.String(), "private-response-text") {
				t.Fatalf("continuation failed: err=%v continuations=%d body=%s", err, continuations, writer.Body.String())
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, secret := range []string{"private-response-text", "private-reasoning", "private-metadata"} {
				if strings.Contains(string(data), secret) {
					t.Fatalf("trace leaked %s: %s", secret, data)
				}
			}
			var completed, continued int
			for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				var entry map[string]any
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					t.Fatal(err)
				}
				if entry["event"] == "stream.continue" {
					continued++
					if entry["response_id"] != "resp-partial" || entry["same_websocket"] != false {
						t.Fatalf("incorrect continuation trace: %#v", entry)
					}
				}
				if entry["codex_event"] != "response.completed" {
					continue
				}
				completed++
				final := entry["upstream_id"] == "resp-final"
				if entry["upstream_end_turn_present"] != true || entry["upstream_end_turn_valid"] != true || entry["upstream_end_turn"] != final || entry["reducer_done"] != final || entry["reducer_needs_follow_up"] != !final || entry["upstream_status"] != "completed" {
					t.Fatalf("incorrect completion decision trace: %#v", entry)
				}
				phase := "commentary"
				if final {
					phase = "final_answer"
					if entry["anthropic_stop_reason"] != "end_turn" {
						t.Fatalf("missing downstream stop reason: %#v", entry)
					}
				} else if _, ok := entry["anthropic_stop_reason"]; ok {
					t.Fatalf("continuation claimed to stop downstream: %#v", entry)
				}
				phases := entry["upstream_message_phases"].([]any)
				types := entry["upstream_output_item_types"].([]any)
				if len(phases) != 1 || phases[0] != phase || len(types) != 2 || types[0] != "message" || types[1] != "reasoning" {
					t.Fatalf("incorrect output metadata trace: %#v", entry)
				}
			}
			if completed != 2 || continued != 1 {
				t.Fatalf("missing completion or continuation trace: %s", data)
			}
		})
	}
}

func TestStreamTraceDistinguishesMissingAndInvalidEndTurn(t *testing.T) {
	for _, tc := range []struct {
		name, field string
		present     bool
	}{
		{name: "absent"},
		{name: "null", field: `,"end_turn":null`, present: true},
		{name: "wrong-type", field: `,"end_turn":"private-invalid-value"`, present: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "trace.jsonl")
			t.Setenv(proxyTraceEnv, path)
			server := New(Config{})
			event := codex.SSEEvent{Event: "response.completed", Data: json.RawMessage(`{"type":"response.completed","response":{"status":"completed"` + tc.field + `}}`)}
			reducer := convert.NewStreamReducer("", "test-model")
			converted, err := reducer.ReduceNamed(event.Event, event.Data)
			if err != nil {
				t.Fatal(err)
			}
			server.traceStreamEvent(nil, time.Now(), event, reducer, converted)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var entry map[string]any
			if err := json.Unmarshal(data, &entry); err != nil {
				t.Fatal(err)
			}
			if entry["upstream_end_turn_present"] != tc.present || entry["upstream_end_turn_valid"] != false || entry["reducer_done"] != true || entry["anthropic_stop_reason"] != "end_turn" || strings.Contains(string(data), "private-invalid-value") {
				t.Fatalf("incorrect missing or invalid end_turn trace: %s", data)
			}
		})
	}
}

func TestPruneTraceLogsUsesLastWriteAndOnlyManagedRegularFiles(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "logs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	cutoff := now.Add(-7 * 24 * time.Hour)
	for _, file := range []struct {
		name  string
		mtime time.Time
	}{
		{"proxy-trace-2026-01-01.jsonl", cutoff.Add(-time.Second)},
		{"proxy-trace-2026-01-02.jsonl", cutoff},
		{"proxy-trace-2026-01-03.jsonl", now},
		{"custom-trace.jsonl", cutoff.Add(-time.Hour)},
		{"proxy-trace-invalid.jsonl", cutoff.Add(-time.Hour)},
	} {
		path := filepath.Join(dir, file.name)
		if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, file.mtime, file.mtime); err != nil {
			t.Fatal(err)
		}
	}
	outside := filepath.Join(t.TempDir(), "keep.jsonl")
	if err := os.WriteFile(outside, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(outside, cutoff.Add(-time.Hour), cutoff.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "proxy-trace-2026-01-04.jsonl")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "proxy-trace-2026-01-05.jsonl"), 0o700); err != nil {
		t.Fatal(err)
	}
	removed, bytes, err := PruneTraceLogs(home, now)
	if err != nil || removed != 1 || bytes != 4 {
		t.Fatalf("cleanup = %d files, %d bytes, %v", removed, bytes, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 6 {
		t.Fatalf("cleanup changed protected entries: %v, %v", entries, err)
	}
	if data, err := os.ReadFile(outside); err != nil || string(data) != "keep" {
		t.Fatalf("cleanup changed symlink target: %q, %v", data, err)
	}
	if removed, _, err := PruneTraceLogs(t.TempDir(), now); err != nil || removed != 0 {
		t.Fatalf("cleanup of missing directory failed: %d, %v", removed, err)
	}
}
