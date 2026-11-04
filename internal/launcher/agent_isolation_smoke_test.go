package launcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/bassner/claudodex/internal/modelconfig"
)

// TestInstalledClaudeAgentIsolationLifecycle uses the real terminal client and
// native Agent, SendMessage, and TaskStop tools. All generation is synthetic.
func TestInstalledClaudeAgentIsolationLifecycle(t *testing.T) {
	if os.Getenv("CLAUDODEX_AGENT_ISOLATION_CHILD") == "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 210*time.Second)
		defer cancel()
		defer func() { _ = os.WriteFile(os.Getenv("CLAUDODEX_AGENT_ISOLATION_CHILD_DONE"), []byte("done"), 0o600) }()
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if _, err := os.Stat(os.Getenv("CLAUDODEX_AGENT_ISOLATION_SHUTDOWN")); err == nil {
						cancel()
						return
					}
				}
			}
		}()
		args := []string{"--model", "gpt-6-astra", "--effort", "high", "--dangerously-skip-permissions", "--system-prompt", "Local synthetic agent isolation acceptance. Follow the requested native tool calls.", "Run the native agent isolation lifecycle fixture."}
		if err := (ProcessLauncher{}).Launch(ctx, args, Config{Version: "smoke", Home: os.Getenv("CLAUDODEX_AGENT_ISOLATION_HOME"), CodexBaseURL: os.Getenv("CLAUDODEX_AGENT_ISOLATION_BACKEND"), Interactive: true}); err != nil && ctx.Err() == nil {
			t.Fatal(err)
		}
		return
	}
	version := nativeCatalogRequire(t)
	tmux, err := exec.LookPath("tmux")
	if err != nil {
		t.Fatalf("installed-Claude agent lifecycle acceptance requires tmux: %v", err)
	}
	home, userHome, work := t.TempDir(), t.TempDir(), t.TempDir()
	work, err = filepath.EvalSymlinks(work)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := os.Getenv("CLAUDODEX_AGENT_ISOLATION_ARTIFACT_DIR")
	if artifacts == "" {
		artifacts = t.TempDir()
	}
	if err := os.MkdirAll(artifacts, 0o700); err != nil {
		t.Fatal(err)
	}
	nativeCatalogAuth(t, home)
	if err := writeJSONFile(filepath.Join(userHome, ".claude.json"), map[string]any{"hasCompletedOnboarding": true, "theme": "dark", "projects": map[string]any{work: map[string]any{"hasTrustDialogAccepted": true}}}, 0o600); err != nil {
		t.Fatal(err)
	}
	redact := strings.NewReplacer(work, "<work>", home, "<auth-home>", userHome, "<user-home>")
	writeArtifact := func(name string, value any) {
		t.Helper()
		data, marshalErr := json.MarshalIndent(value, "", "  ")
		if marshalErr != nil {
			t.Error(marshalErr)
			return
		}
		var redacted any
		if err := json.Unmarshal(data, &redacted); err == nil {
			agentIsolationRedactIDs(redacted)
			data, _ = json.MarshalIndent(redacted, "", "  ")
		}
		if err := os.WriteFile(filepath.Join(artifacts, name), []byte(redact.Replace(string(data))), 0o600); err != nil {
			t.Error(err)
		}
	}
	type observation struct {
		ThreadID string         `json:"thread_id"`
		ParentID string         `json:"parent_thread_id,omitempty"`
		Request  map[string]any `json:"request"`
	}
	var mu sync.Mutex
	var observations []observation
	var parentPhase int
	var astraThread, solThread string
	var astraRequests, inheritedAbsent, inheritedExplicit int
	astraStarted, solStarted := make(chan struct{}), make(chan struct{})
	astraDone, solCanceled, followed := make(chan struct{}), make(chan struct{}), make(chan struct{})
	finished := make(chan struct{}, 1)
	var astraStartOnce, solStartOnce, astraDoneOnce, solCancelOnce, followOnce sync.Once
	wait := func(r *http.Request, ch <-chan struct{}) bool {
		select {
		case <-ch:
			return true
		case <-r.Context().Done():
			return false
		}
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/codex/models" {
			models := nativeCatalogFixture()
			var current []codex.ModelInfo
			for _, model := range models {
				if !strings.HasPrefix(model.Slug, "gpt-8") {
					current = append(current, model)
				}
			}
			_ = json.NewEncoder(w).Encode(codex.ModelsResponse{Models: current})
			return
		}
		if r.URL.Path != "/codex/responses" || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusUpgradeRequired)
			return
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			if err != io.EOF {
				t.Errorf("decode native request: %v", err)
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		thread, parent := r.Header.Get("thread-id"), r.Header.Get("x-codex-parent-thread-id")
		mu.Lock()
		observations = append(observations, observation{ThreadID: thread, ParentID: parent, Request: request})
		mu.Unlock()
		if parent != "" {
			input, _ := json.Marshal(request["input"])
			switch {
			case strings.Contains(string(input), "isolation-inherit-absent"):
				mu.Lock()
				inheritedAbsent++
				mu.Unlock()
				nativeCatalogText(w, "inherited-absent-ok")
			case strings.Contains(string(input), "isolation-inherit-explicit"):
				mu.Lock()
				inheritedExplicit++
				mu.Unlock()
				nativeCatalogText(w, "inherited-explicit-ok")
			case request["model"] == "gpt-6.1-sol":
				mu.Lock()
				solThread = thread
				mu.Unlock()
				solStartOnce.Do(func() { close(solStarted) })
				w.Header().Set("content-type", "text/event-stream")
				_, _ = io.WriteString(w, "event: response.created\ndata: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_isolation_sol\"}}\n\n")
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
				<-r.Context().Done()
				solCancelOnce.Do(func() { close(solCanceled) })
			case request["model"] == "gpt-6-astra":
				mu.Lock()
				astraRequests++
				count := astraRequests
				if astraThread == "" {
					astraThread = thread
				} else if astraThread != thread {
					t.Errorf("SendMessage changed native worker thread: %q -> %q", astraThread, thread)
				}
				mu.Unlock()
				astraStartOnce.Do(func() { close(astraStarted) })
				if count == 1 {
					if !wait(r, solStarted) {
						return
					}
					nativeCatalogText(w, "astra-first-ok")
					astraDoneOnce.Do(func() { close(astraDone) })
				} else {
					if !wait(r, solCanceled) {
						return
					}
					nativeCatalogText(w, "astra-followup-ok")
					followOnce.Do(func() { close(followed) })
				}
			default:
				t.Errorf("unexpected native child model %v", request["model"])
				nativeCatalogText(w, "unexpected-child-model")
			}
			return
		}
		mu.Lock()
		phase := parentPhase
		parentPhase++
		mu.Unlock()
		switch phase {
		case 0:
			prompt := "Identical delegated prompt. Return the fixture response and accept a follow-up."
			agentIsolationCalls(w,
				agentIsolationCall("spawn_astra", "Agent", map[string]any{"name": "isolation-astra", "description": "Native Astra worker", "prompt": prompt, "subagent_type": "general-purpose", "model": "gpt-6-astra", "run_in_background": true}),
				agentIsolationCall("spawn_sol", "Agent", map[string]any{"name": "isolation-sol", "description": "Native Sol worker", "prompt": prompt, "subagent_type": "general-purpose", "model": "gpt-6.1-sol", "run_in_background": true}))
		case 1:
			if !wait(r, astraStarted) || !wait(r, solStarted) || !wait(r, astraDone) {
				return
			}
			agentIsolationCalls(w,
				agentIsolationCall("follow_astra", "SendMessage", map[string]any{"to": "isolation-astra", "message": "Native follow-up: reply astra-followup-ok", "summary": "Check stable worker identity"}),
				agentIsolationCall("stop_sol", "TaskStop", map[string]any{"task_id": "isolation-sol"}))
		case 2:
			if !wait(r, followed) || !wait(r, solCanceled) {
				return
			}
			agentIsolationCalls(w,
				agentIsolationCall("inherit_absent", "Agent", map[string]any{"name": "inherit-absent", "description": "Missing model inheritance", "prompt": "isolation-inherit-absent", "subagent_type": "general-purpose"}),
				agentIsolationCall("inherit_explicit", "Agent", map[string]any{"name": "inherit-explicit", "description": "Explicit native inheritance", "prompt": "isolation-inherit-explicit", "subagent_type": "general-purpose", "model": "inherit"}))
		default:
			nativeCatalogText(w, "agent-isolation-lifecycle-finished")
			select {
			case finished <- struct{}{}:
			default:
			}
		}
	}))
	defer upstream.Close()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	socketDir, err := os.MkdirTemp("/tmp", "clx-agent-tmux-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(socketDir)
	socket := filepath.Join(socketDir, "tmux.sock")
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	tracePath := filepath.Join(artifacts, "proxy.jsonl")
	if err := os.WriteFile(tracePath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	childDone := filepath.Join(socketDir, "child.done")
	shutdown := filepath.Join(socketDir, "shutdown")
	assignments := map[string]string{"HOME": userHome, "CLAUDE_CONFIG_DIR": "", "CLAUDE_SECURESTORAGE_CONFIG_DIR": "", "CLAUDODEX_DISABLE_REAL_CLAUDE_BRIDGE_AUTH": "1", "CLAUDODEX_AGENT_ISOLATION_CHILD": "1", "CLAUDODEX_AGENT_ISOLATION_HOME": home, "CLAUDODEX_AGENT_ISOLATION_BACKEND": upstream.URL, "CLAUDODEX_AGENT_ISOLATION_CHILD_DONE": childDone, "CLAUDODEX_AGENT_ISOLATION_SHUTDOWN": shutdown, "CLAUDODEX_CODEX_BASE_URL": upstream.URL, "CLAUDODEX_PROXY_TRACE": tracePath, "CLAUDODEX_PROXY_LOG": filepath.Join(artifacts, "http.log"), "CLAUDODEX_DISABLE_CLAUDE_PATCH": "", "TERM": "xterm-256color"}
	command := "cd " + shellQuote(work) + " && env"
	for key, value := range assignments {
		command += " " + key + "=" + shellQuote(value)
	}
	command += " " + shellQuote(executable) + " -test.run='^TestInstalledClaudeAgentIsolationLifecycle$' -test.v"
	if output, err := exec.CommandContext(ctx, tmux, "-S", socket, "new-session", "-d", "-s", "isolation", "-x", "150", "-y", "45", command).CombinedOutput(); err != nil {
		t.Fatalf("start private tmux: %v: %s", err, output)
	}
	shutdownChild := sync.OnceFunc(func() {
		// Wait for Launch's settings reconciliation and proxy shutdown before
		// Go removes the isolated homes. Killing tmux alone races those writes.
		_ = os.WriteFile(shutdown, []byte("stop"), 0o600)
		deadline := time.Now().Add(12 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(childDone); err == nil {
				_ = exec.Command(tmux, "-S", socket, "kill-server").Run()
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		_ = exec.Command(tmux, "-S", socket, "kill-server").Run()
		t.Error("native child did not finish launcher shutdown before cleanup")
	})
	defer shutdownChild()
	capture := func() string {
		captureCtx, captureCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer captureCancel()
		output, _ := exec.CommandContext(captureCtx, tmux, "-S", socket, "capture-pane", "-p", "-S", "-2000", "-t", "isolation:0.0").CombinedOutput()
		return redact.Replace(string(output))
	}
	var screen string
	defer func() {
		terminal := screen
		if terminal == "" {
			terminal = capture()
		}
		_ = os.WriteFile(filepath.Join(artifacts, "terminal-final.txt"), []byte(terminal), 0o600)
		mu.Lock()
		captured := append([]observation(nil), observations...)
		mu.Unlock()
		writeArtifact("requests.json", captured)
	}()
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	complete := false
	for !complete {
		select {
		case <-finished:
			complete = true
		case <-ctx.Done():
			t.Fatalf("native agent lifecycle did not finish; artifacts=%s\n%s", artifacts, capture())
		case <-ticker.C:
			screen = capture()
			if strings.Contains(screen, "Yes, I accept") || strings.Contains(screen, "Yes, I trust this folder") {
				_ = exec.CommandContext(ctx, tmux, "-S", socket, "send-keys", "-t", "isolation:0.0", "Down", "Enter").Run()
			}
		}
	}
	for !strings.Contains(screen, "agent-isolation-lifecycle-finished") && ctx.Err() == nil {
		<-ticker.C
		screen = capture()
	}
	if ctx.Err() != nil || !strings.Contains(screen, "using Claude Code v"+version) {
		t.Fatalf("native lifecycle terminal acceptance failed: %s", screen)
	}
	// Both counts must describe the same completed lifetime. The native UI
	// can still submit background requests after displaying the final text.
	shutdownChild()
	mu.Lock()
	captured := append([]observation(nil), observations...)
	gotAstra, gotSol, gotAstraRequests := astraThread, solThread, astraRequests
	gotAbsent, gotExplicit := inheritedAbsent, inheritedExplicit
	mu.Unlock()
	if gotAstra == "" || gotSol == "" || gotAstra == gotSol || gotAstraRequests < 2 {
		t.Fatalf("native child routes astra=%q sol=%q astra_requests=%d", gotAstra, gotSol, gotAstraRequests)
	}
	if gotAbsent != 1 || gotExplicit != 1 {
		t.Fatalf("native inheritance requests absent=%d explicit=%d", gotAbsent, gotExplicit)
	}
	var agentSchema, rootThread string
	nativeIDs := make(map[string]string)
	var resumedID, stoppedID string
	for _, observed := range captured {
		if observed.ParentID == "" && rootThread == "" {
			rootThread = observed.ThreadID
		}
		if observed.ParentID == "" && agentSchema == "" {
			tools, _ := observed.Request["tools"].([]any)
			for _, tool := range tools {
				entry, _ := tool.(map[string]any)
				if entry["name"] == "Agent" {
					data, _ := json.Marshal(entry)
					agentSchema = string(data)
				}
			}
		}
		input, _ := json.Marshal(observed.Request["input"])
		if observed.ParentID != "" && (strings.Contains(string(input), "isolation-inherit-absent") || strings.Contains(string(input), "isolation-inherit-explicit")) && observed.Request["model"] != "gpt-6-astra" {
			t.Errorf("parent Astra native inheritance routed to %v", observed.Request["model"])
		}
		if observed.Request["previous_response_id"] != nil {
			t.Errorf("HTTP native fixture unexpectedly used previous_response_id: %v", observed.Request["previous_response_id"])
		}
		if observed.Request["store"] != false {
			t.Error("native HTTP fixture enabled upstream response storage")
		}
		items, _ := observed.Request["input"].([]any)
		for _, item := range items {
			entry, _ := item.(map[string]any)
			if entry["type"] != "function_call_output" {
				continue
			}
			output, _ := entry["output"].(string)
			callID, _ := entry["call_id"].(string)
			if suffix, found := strings.CutPrefix(callID, "call_spawn_"); found {
				if _, after, ok := strings.Cut(output, "\nagentId: "); ok {
					if fields := strings.Fields(after); len(fields) > 0 {
						nativeIDs[suffix] = fields[0]
					}
				}
			}
			var result map[string]any
			if json.Unmarshal([]byte(output), &result) == nil {
				if callID == "call_follow_astra" && result["success"] == true {
					resumedID, _ = result["resumedAgentId"].(string)
				}
				if callID == "call_stop_sol" && strings.Contains(output, "Successfully stopped task") {
					stoppedID, _ = result["task_id"].(string)
				}
			}
		}
	}
	if nativeIDs["astra"] == "" || nativeIDs["sol"] == "" || gotAstra != rootThread+":agent:"+nativeIDs["astra"] || gotSol != rootThread+":agent:"+nativeIDs["sol"] {
		t.Errorf("worker routes do not use native IDs: astra=%q sol=%q native=%v", gotAstra, gotSol, nativeIDs)
	}
	if resumedID != nativeIDs["astra"] || stoppedID != nativeIDs["sol"] {
		t.Errorf("native lifecycle tools did not confirm worker IDs: resumed=%q stopped=%q", resumedID, stoppedID)
	}
	for _, model := range modelconfig.Default().RequiredModels() {
		if !strings.Contains(agentSchema, model) {
			t.Errorf("upstream Agent schema omits resolved model %s", model)
		}
	}
	trace, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	var requestIDs []string
	for _, line := range strings.Split(string(trace), "\n") {
		var event map[string]any
		if json.Unmarshal([]byte(line), &event) == nil && event["event"] == "messages.request" {
			id, _ := event["request_id"].(string)
			requestIDs = append(requestIDs, id)
			if event["stateless_subagent"] != false {
				t.Errorf("native header identity unexpectedly used stateless fallback: %v", event)
			}
		}
	}
	if len(requestIDs) != len(captured) {
		t.Errorf("proxy request trace count %d differs from fake generation count %d", len(requestIDs), len(captured))
	}
	writeArtifact("result.json", map[string]any{"passed": !t.Failed(), "claude_version": version, "paid_model_requests": 0, "native_parallel_workers": true, "identical_delegated_prompts": true, "explicit_models": []string{"gpt-6-astra", "gpt-6.1-sol"}, "separate_worker_threads": true, "send_message_stable_thread": gotAstraRequests >= 2, "task_stop_canceled_upstream": true, "sibling_completed_after_cancel": true, "absent_model_inherits_astra": gotAbsent == 1, "inherit_model_inherits_astra": gotExplicit == 1, "generation_requests": len(captured), "proxy_request_ids": requestIDs})
	t.Logf("native Claude %s agent isolation artifacts: %s", version, artifacts)
}

func agentIsolationRedactIDs(value any) {
	switch entry := value.(type) {
	case map[string]any:
		for key, child := range entry {
			if key == "x-codex-installation-id" || key == "installation_id" {
				entry[key] = "<synthetic-installation>"
				continue
			}
			if key == "x-codex-turn-metadata" {
				if encoded, ok := child.(string); ok {
					var metadata any
					if json.Unmarshal([]byte(encoded), &metadata) == nil {
						agentIsolationRedactIDs(metadata)
						redacted, _ := json.Marshal(metadata)
						entry[key] = string(redacted)
					}
				}
				continue
			}
			agentIsolationRedactIDs(child)
		}
	case []any:
		for _, child := range entry {
			agentIsolationRedactIDs(child)
		}
	}
}

func agentIsolationCall(id, name string, args map[string]any) map[string]any {
	arguments, _ := json.Marshal(args)
	return map[string]any{"type": "function_call", "id": "fc_" + id, "call_id": "call_" + id, "name": name, "arguments": string(arguments)}
}

func agentIsolationCalls(w http.ResponseWriter, calls ...map[string]any) {
	w.Header().Set("content-type", "text/event-stream")
	events := []map[string]any{{"type": "response.created", "response": map[string]any{"id": "resp_isolation"}}}
	for index, item := range calls {
		events = append(events, map[string]any{"type": "response.output_item.added", "output_index": index, "item": item}, map[string]any{"type": "response.output_item.done", "output_index": index, "item": item})
	}
	events = append(events, map[string]any{"type": "response.completed", "response": map[string]any{"stop_reason": "tool_calls", "usage": map[string]any{"input_tokens": 2, "output_tokens": 1}}})
	for _, event := range events {
		data, _ := json.Marshal(event)
		_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event["type"], data)
	}
}
