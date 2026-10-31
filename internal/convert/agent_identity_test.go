package convert

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
)

func agentIdentityFixture(t *testing.T) AnthropicRequest {
	t.Helper()
	var req AnthropicRequest
	if err := json.Unmarshal([]byte(`{"system":"You are an agent for Claude Code. Read /workspace/agent-guidance/README.md and a quoted /repo/.claude/worktrees/agent-deadbeef directory.","messages":[{"role":"user","content":"Review the same change."}]}`), &req); err != nil {
		t.Fatal(err)
	}
	return req
}

func TestNativeAgentIdentityOverridesAllInstructionPaths(t *testing.T) {
	req := agentIdentityFixture(t)
	for _, id := range []string{"worker-a", "worker-b"} {
		got, err := AnthropicToCodex(req, ConvertOptions{SessionID: "root", AgentID: id})
		if err != nil {
			t.Fatal(err)
		}
		if got.RouteSessionID != "root:agent:"+id || got.StatelessSubagent || got.ParentThreadID != "root" {
			t.Fatalf("native identity %q lost: %#v", id, got)
		}
	}
}

func TestMissingNativeWorkerIdentityAlwaysUsesFreshStatelessRoute(t *testing.T) {
	for _, opts := range []ConvertOptions{{SessionID: "root"}, {AgentID: "worker-a"}, {RequestClass: "subagent"}} {
		first, err := AnthropicToCodex(agentIdentityFixture(t), opts)
		if err != nil {
			t.Fatal(err)
		}
		second, err := AnthropicToCodex(agentIdentityFixture(t), opts)
		if err != nil {
			t.Fatal(err)
		}
		if !first.StatelessSubagent || !second.StatelessSubagent || first.RouteSessionID == second.RouteSessionID || first.RouteSessionID == "" {
			t.Fatalf("unidentified workers shared state: first=%q second=%q", first.RouteSessionID, second.RouteSessionID)
		}
	}
}

func TestNativeRequestClassAndAgentIdentityClassifyWorker(t *testing.T) {
	var plain AnthropicRequest
	if err := json.Unmarshal([]byte(`{"messages":[{"role":"user","content":"hello"}]}`), &plain); err != nil {
		t.Fatal(err)
	}
	for _, opts := range []ConvertOptions{{SessionID: "root", AgentID: "worker-a"}, {SessionID: "root", RequestClass: "subagent"}, {SessionID: "root", RequestClass: "workflow"}} {
		got, err := AnthropicToCodex(plain, opts)
		if err != nil || got.Subagent != "collab_spawn" {
			t.Fatalf("native worker metadata ignored: %#v %v", got, err)
		}
	}
	got, err := AnthropicToCodex(agentIdentityFixture(t), ConvertOptions{SessionID: "root", RequestClass: "main"})
	if err != nil || got.Subagent != "" || got.RouteSessionID != "root" {
		t.Fatalf("main request confused by quoted worker instructions: %#v %v", got, err)
	}
}

func TestInvalidNativeWorkerIdentityRejectedWithoutNormalization(t *testing.T) {
	for _, id := range []string{"worker/a", "worker:a", " worker", "worker ", "worker\nnext", "wörker", strings.Repeat("a", 81)} {
		if _, err := AnthropicToCodex(agentIdentityFixture(t), ConvertOptions{SessionID: "root", AgentID: id}); err == nil {
			t.Errorf("invalid identity accepted: %q", id)
		}
	}
}

func TestAgentModelSchemaAdvertisesResolvedTiersAndPreservesNativeRules(t *testing.T) {
	models := modelconfig.Config{Opus: "gpt-8-sol", Sonnet: "gpt-8-terra", Haiku: "gpt-8-luna", Fable: "gpt-8-astra"}
	schema := map[string]any{"type": "object", "properties": map[string]any{"model": map[string]any{"type": "string", "enum": []any{"opus", "sonnet", "haiku"}, "description": "Fork workers always inherit."}}}
	tools := convertToolsWithModels([]AnthropicTool{{Name: "Agent", InputSchema: schema}}, models)
	field := tools[0].Parameters["properties"].(map[string]any)["model"].(map[string]any)
	description, _ := field["description"].(string)
	for _, text := range append(models.RequiredModels(), "Fork workers always inherit.", "A name in the prompt or description does not select the model", "Preserve other explicit supported Codex IDs") {
		if !strings.Contains(description, text) {
			t.Errorf("missing runtime guidance %q: %s", text, description)
		}
	}
	if _, restricted := field["enum"]; restricted {
		t.Fatal("native aliases still restrict exact model pins")
	}
	if _, mutated := schema["properties"].(map[string]any)["model"].(map[string]any)["enum"]; !mutated {
		t.Fatal("conversion mutated native input schema")
	}
	for _, pin := range []string{"gpt-8-astra", "gpt-5.6-sol", "gpt-future-haiku", "gpt-future-opus", "gpt-future-sonnet", "gpt-future-fable", "gpt-future-mythos"} {
		got := normalizeClaudeCodeToolArgs("Agent", map[string]any{"model": pin}, models, "")
		if got["model"] != pin {
			t.Fatalf("explicit runtime %q changed to %#v", pin, got["model"])
		}
		if _, err := AnthropicToCodex(AnthropicRequest{Model: pin}, ConvertOptions{Models: models, CodexModels: []codex.ModelInfo{}}); err == nil {
			t.Fatalf("unavailable explicit pin %q was not rejected", pin)
		}
	}
}
