package launcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/gorilla/websocket"
)

const instructionResumeGuidanceMarker = "CLX_NATIVE_INSTRUCTION_GUIDANCE_V1"
const instructionResumeFinalMarker = "native-instruction-resume-complete"
const instructionResumeReadRounds = 16

func TestInstalledClaudeInstructionResumeChild(t *testing.T) {
	if os.Getenv("CLAUDODEX_INSTRUCTION_TEST_CHILD") != "1" {
		t.Skip("private terminal child only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	// Keep the argument below Darwin's per-argument limit while reproducing a
	// substantial stable native system block on every Read-tool round.
	guidance := instructionResumeGuidanceMarker + "\n" + strings.Repeat("Stable synthetic guidance. Read each requested fixture file and retain this guidance.\n", 1100)
	err := (ProcessLauncher{}).Launch(ctx, []string{"--model", "opus", "--dangerously-skip-permissions", "--system-prompt", guidance, "Read the sixteen different tiny fixture files requested by the tool calls, then finish."}, Config{
		Version: "smoke", Home: os.Getenv("CLAUDODEX_INSTRUCTION_TEST_HOME"), CodexBaseURL: os.Getenv("CLAUDODEX_INSTRUCTION_TEST_BACKEND"), Interactive: true,
	})
	if err != nil && ctx.Err() == nil {
		t.Fatal(err)
	}
}

type instructionResumeWireObservation struct {
	Turn                  int    `json:"turn"`
	ResponseID            string `json:"response_id"`
	PreviousResponseID    string `json:"previous_response_id,omitempty"`
	InputItems            int    `json:"input_items"`
	InputBytes            int    `json:"input_bytes"`
	DeveloperItems        int    `json:"developer_items"`
	ContextDeveloperItems int    `json:"context_developer_items"`
	GuidanceCopies        int    `json:"guidance_copies"`
	GuidanceBytes         int    `json:"guidance_bytes"`
	ContextBytes          int    `json:"context_bytes"`
	SyntheticInputTokens  int    `json:"synthetic_input_tokens"`
	CompletedReads        int    `json:"completed_reads"`
}

func TestInstalledClaudeInstructionResume(t *testing.T) {
	version := nativeCatalogRequire(t)
	tmux, err := exec.LookPath("tmux")
	if err != nil {
		t.Fatal(err)
	}
	home, userHome, work := t.TempDir(), t.TempDir(), t.TempDir()
	work, err = filepath.EvalSymlinks(work)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := nativeCatalogArtifacts(t)
	nativeCatalogAuth(t, home)
	if err := os.MkdirAll(filepath.Join(userHome, ".claude"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(filepath.Join(userHome, ".claude.json"), map[string]any{"hasCompletedOnboarding": true, "theme": "dark", "projects": map[string]any{work: map[string]any{"hasTrustDialogAccepted": true}}}, 0o600); err != nil {
		t.Fatal(err)
	}
	for index := range instructionResumeReadRounds {
		if err := os.WriteFile(filepath.Join(work, fmt.Sprintf("tiny-%02d.txt", index)), []byte(fmt.Sprintf("native-instruction-result-%02d\n", index)), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var mu sync.Mutex
	contexts := map[string][]codex.InputItem{}
	connections := map[*websocket.Conn]bool{}
	completedReads := map[int]bool{}
	var observations []instructionResumeWireObservation
	var helperRequests, incrementalRequests, fullRequests int
	failure := make(chan error, 1)
	fail := func(format string, args ...any) {
		select {
		case failure <- fmt.Errorf(format, args...):
		default:
		}
	}
	defer func() {
		mu.Lock()
		defer mu.Unlock()
		result := map[string]any{"claude_version": version, "paid_model_requests": 0, "read_rounds": instructionResumeReadRounds, "completed_reads": len(completedReads), "incremental_requests": incrementalRequests, "full_requests": fullRequests, "helper_requests": helperRequests, "wire_observations": observations, "passed": !t.Failed()}
		if err := writeJSONFile(filepath.Join(artifacts, "result.json"), result, 0o600); err != nil {
			t.Error(err)
		}
	}()
	upgrader := websocket.Upgrader{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/codex/models" {
			_ = json.NewEncoder(w).Encode(codex.ModelsResponse{Models: nativeCatalogFixture()})
			return
		}
		if r.URL.Path != "/codex/responses" {
			http.NotFound(w, r)
			return
		}
		if !websocket.IsWebSocketUpgrade(r) {
			fail("native instruction acceptance requires actual WebSocket response.create requests; got %s", r.Method)
			http.Error(w, "WebSocket required", http.StatusBadRequest)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fail("upgrade fake Codex WebSocket: %v", err)
			return
		}
		mu.Lock()
		connections[conn] = true
		mu.Unlock()
		defer func() {
			_ = conn.Close()
			mu.Lock()
			delete(connections, conn)
			mu.Unlock()
		}()
		for {
			_ = conn.SetReadDeadline(time.Now().Add(180 * time.Second))
			var request struct {
				Type string `json:"type"`
				codex.Request
			}
			if err := conn.ReadJSON(&request); err != nil {
				return
			}
			inputJSON, _ := json.Marshal(request.Input)
			mu.Lock()
			prior, knownPrevious := contexts[request.PreviousResponseID]
			isMain := knownPrevious
			developerItems := 0
			for _, item := range request.Input {
				if item.Role == "developer" {
					developerItems++
					for _, part := range item.Content {
						isMain = isMain || strings.Contains(part.Text, instructionResumeGuidanceMarker)
					}
				}
			}
			if !isMain {
				helperRequests++
				responseID := fmt.Sprintf("resp_instruction_helper_%d", helperRequests)
				mu.Unlock()
				if !instructionResumeWriteResponse(conn, responseID, instructionResumeTextItem("fixture helper complete"), 1, true) {
					return
				}
				continue
			}
			if request.Type != "response.create" {
				mu.Unlock()
				fail("wire request type = %q, want response.create", request.Type)
				return
			}
			if request.PreviousResponseID != "" && !knownPrevious {
				mu.Unlock()
				fail("unknown previous response %q", request.PreviousResponseID)
				return
			}
			providerContext := append([]codex.InputItem(nil), request.Input...)
			if knownPrevious {
				incrementalRequests++
				providerContext = append(append([]codex.InputItem(nil), prior...), request.Input...)
			} else {
				fullRequests++
			}
			contextDeveloperItems, guidanceCopies, guidanceBytes := 0, 0, 0
			for _, item := range providerContext {
				if item.Role == "developer" {
					contextDeveloperItems++
					for _, part := range item.Content {
						guidanceCopies += strings.Count(part.Text, instructionResumeGuidanceMarker)
						guidanceBytes += len(part.Text)
					}
				}
				if item.Type == "function_call_output" {
					for index := range instructionResumeReadRounds {
						if item.CallID != fmt.Sprintf("call_instruction_read_%02d", index) {
							continue
						}
						output, _ := json.Marshal(item.Output)
						if !strings.Contains(string(output), fmt.Sprintf("native-instruction-result-%02d", index)) {
							mu.Unlock()
							fail("real Read %02d returned no expected fixture contents: %s", index, output)
							return
						}
						completedReads[index] = true
					}
				}
			}
			contextJSON, _ := json.Marshal(providerContext)
			inputTokens := (len(contextJSON) + 3) / 4
			responseID := fmt.Sprintf("resp_instruction_main_%02d", len(observations))
			observation := instructionResumeWireObservation{Turn: len(observations) + 1, ResponseID: responseID, PreviousResponseID: request.PreviousResponseID, InputItems: len(request.Input), InputBytes: len(inputJSON), DeveloperItems: developerItems, ContextDeveloperItems: contextDeveloperItems, GuidanceCopies: guidanceCopies, GuidanceBytes: guidanceBytes, ContextBytes: len(contextJSON), SyntheticInputTokens: inputTokens, CompletedReads: len(completedReads)}
			observations = append(observations, observation)
			if knownPrevious && developerItems != 0 {
				mu.Unlock()
				fail("incremental turn %d appended %d developer guidance items; provider context now holds %d guidance copies (%d bytes)", observation.Turn, developerItems, guidanceCopies, len(contextJSON))
				return
			}
			if guidanceCopies != 1 || contextDeveloperItems != 1 || guidanceBytes < 80000 {
				mu.Unlock()
				fail("turn %d provider context has developer_items=%d guidance_copies=%d guidance_bytes=%d; want one substantial current guidance copy", observation.Turn, contextDeveloperItems, guidanceCopies, guidanceBytes)
				return
			}
			var item codex.InputItem
			final := len(completedReads) == instructionResumeReadRounds
			if final {
				item = instructionResumeTextItem(instructionResumeFinalMarker)
			} else {
				index := 0
				for completedReads[index] {
					index++
				}
				arguments, _ := json.Marshal(map[string]string{"file_path": filepath.Join(work, fmt.Sprintf("tiny-%02d.txt", index))})
				item = codex.InputItem{Type: "function_call", ID: fmt.Sprintf("fc_instruction_read_%02d", index), CallID: fmt.Sprintf("call_instruction_read_%02d", index), Name: "Read", Arguments: string(arguments)}
			}
			contexts[responseID] = append(providerContext, item)
			mu.Unlock()
			if !instructionResumeWriteResponse(conn, responseID, item, inputTokens, final) {
				return
			}
		}
	}))
	defer upstream.Close()
	defer func() {
		mu.Lock()
		defer mu.Unlock()
		for conn := range connections {
			_ = conn.Close()
		}
	}()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	socketDir, err := os.MkdirTemp("/tmp", "clx-instruction-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(socketDir)
	socket, ready := filepath.Join(socketDir, "tmux.sock"), filepath.Join(socketDir, "ready")
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	command := "while [ ! -f " + shellQuote(ready) + " ]; do sleep 0.05; done; cd " + shellQuote(work) + " && exec env"
	for key, value := range map[string]string{
		"HOME": userHome, "CLAUDE_CONFIG_DIR": "", "CLAUDE_SECURESTORAGE_CONFIG_DIR": "", "CLAUDODEX_DISABLE_REAL_CLAUDE_BRIDGE_AUTH": "1",
		"CLAUDODEX_INSTRUCTION_TEST_CHILD": "1", "CLAUDODEX_INSTRUCTION_TEST_HOME": home, "CLAUDODEX_INSTRUCTION_TEST_BACKEND": upstream.URL, "CLAUDODEX_CODEX_BASE_URL": upstream.URL,
		"CLAUDODEX_FORCE_CODEX_WEBSOCKET": "1", "CLAUDODEX_DISABLE_CODEX_WEBSOCKET": "0", "CLAUDODEX_DISABLE_CLAUDE_PATCH": "", "TERM": "xterm-256color",
	} {
		command += " " + key + "=" + shellQuote(value)
	}
	command += " " + shellQuote(executable) + " -test.run='^TestInstalledClaudeInstructionResumeChild$' -test.v"
	if output, err := exec.CommandContext(ctx, tmux, "-S", socket, "new-session", "-d", "-s", "instructions", "-x", "140", "-y", "48", "/bin/sh -c "+shellQuote(command)).CombinedOutput(); err != nil {
		t.Fatalf("start private terminal: %v: %s", err, output)
	}
	defer func() {
		_ = exec.Command(tmux, "-S", socket, "send-keys", "-t", "instructions:0.0", "C-c").Run()
		_ = exec.Command(tmux, "-S", socket, "send-keys", "-t", "instructions:0.0", "C-d").Run()
		time.Sleep(150 * time.Millisecond)
		_ = exec.Command(tmux, "-S", socket, "send-keys", "-t", "instructions:0.0", "C-d").Run()
		deadline := time.Now().Add(8 * time.Second)
		for time.Now().Before(deadline) {
			if exec.Command(tmux, "-S", socket, "has-session", "-t", "instructions").Run() != nil {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		_ = exec.Command(tmux, "-S", socket, "kill-server").Run()
	}()
	if output, err := exec.CommandContext(ctx, tmux, "-S", socket, "pipe-pane", "-o", "-t", "instructions:0.0", "cat > "+shellQuote(filepath.Join(artifacts, "terminal.raw"))).CombinedOutput(); err != nil {
		t.Fatalf("capture raw terminal: %v: %s", err, output)
	}
	if err := os.WriteFile(ready, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	capture := func() string {
		output, _ := exec.CommandContext(ctx, tmux, "-S", socket, "capture-pane", "-p", "-S", "-2000", "-t", "instructions:0.0").CombinedOutput()
		return string(output)
	}
	var screen string
	for ctx.Err() == nil {
		screen = capture()
		select {
		case err := <-failure:
			_ = os.WriteFile(filepath.Join(artifacts, "terminal.txt"), []byte(screen), 0o600)
			t.Fatal(err)
		default:
		}
		if strings.Contains(screen, instructionResumeFinalMarker) {
			break
		}
		if strings.Contains(screen, "Yes, I accept") || strings.Contains(screen, "Yes, I trust this folder") {
			_ = exec.CommandContext(ctx, tmux, "-S", socket, "send-keys", "-t", "instructions:0.0", "Down", "Enter").Run()
		}
		if strings.Contains(screen, "no server running") || strings.Contains(screen, "can't find session") {
			t.Fatalf("native terminal exited early: %s", screen)
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err := os.WriteFile(filepath.Join(artifacts, "terminal.txt"), []byte(screen), 0o600); err != nil {
		t.Fatal(err)
	}
	if ctx.Err() != nil {
		t.Fatalf("native Read loop did not finish: %s", screen)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(completedReads) != instructionResumeReadRounds || incrementalRequests < 8 {
		t.Fatalf("native tool loop did not exercise enough successful deltas: reads=%d incremental_requests=%d", len(completedReads), incrementalRequests)
	}
	first, last := observations[0], observations[len(observations)-1]
	if last.ContextBytes > first.ContextBytes+instructionResumeReadRounds*4096 {
		t.Fatalf("provider context grew beyond initial guidance plus small tool results: initial=%d final=%d", first.ContextBytes, last.ContextBytes)
	}
	if strings.Contains(screen, "Compacting conversation") || strings.Contains(screen, "Conversation compacted") {
		t.Fatal("tiny native Read loop unexpectedly compacted")
	}
	t.Logf("native instruction acceptance: %d Read results, %d incremental requests, %d full requests; context %d -> %d bytes; artifacts: %s", len(completedReads), incrementalRequests, fullRequests, first.ContextBytes, last.ContextBytes, artifacts)
}

func instructionResumeTextItem(text string) codex.InputItem {
	return codex.InputItem{Type: "message", Role: "assistant", Raw: map[string]json.RawMessage{
		"type": json.RawMessage(`"message"`), "role": json.RawMessage(`"assistant"`), "phase": json.RawMessage(`"final_answer"`), "content": json.RawMessage(fmt.Sprintf(`[{"type":"output_text","text":%q}]`, text)),
	}}
}

func instructionResumeWriteResponse(conn *websocket.Conn, responseID string, item codex.InputItem, inputTokens int, final bool) bool {
	stopReason := "tool_calls"
	if final {
		stopReason = "stop"
	}
	outputJSON, _ := json.Marshal(item)
	outputTokens := (len(outputJSON) + 3) / 4
	for _, event := range []any{
		map[string]any{"type": "response.created", "response": map[string]any{"id": responseID}},
		map[string]any{"type": "response.output_item.added", "output_index": 0, "item": item},
		map[string]any{"type": "response.output_item.done", "output_index": 0, "item": item},
		map[string]any{"type": "response.completed", "response": map[string]any{"id": responseID, "end_turn": true, "stop_reason": stopReason, "usage": map[string]any{"input_tokens": inputTokens, "output_tokens": outputTokens}}},
	} {
		_ = conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
		if err := conn.WriteJSON(event); err != nil {
			return false
		}
	}
	return true
}
