package convert

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
)

func TestAnthropicToCodexStripsBillingAndMapsMaxEffortToXHigh(t *testing.T) {
	body := []byte(`{
		"model":"claude-sonnet-4-6",
		"system":"keep this\nx-anthropic-billing-header: cc_version=1; cch=secret\nand this",
		"output_config":{"effort":"max"},
		"messages":[{"role":"user","content":"hello"}]
	}`)
	var req AnthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	got, err := AnthropicToCodex(req, ConvertOptions{SessionID: "session-1"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Request.Model != "gpt-5.4" {
		t.Fatalf("model = %q", got.Request.Model)
	}
	if got.Request.Reasoning == nil || got.Request.Reasoning.Effort != "xhigh" {
		t.Fatalf("effort = %#v, want xhigh", got.Request.Reasoning)
	}
	if !strings.HasPrefix(got.Request.Instructions, "keep this\nand this\n\nClaude Code compatibility:\n") {
		t.Fatalf("instructions = %q", got.Request.Instructions)
	}
	if strings.Contains(got.Request.Instructions, "x-anthropic-billing-header") || strings.Contains(got.Request.Instructions, "cch=secret") {
		t.Fatalf("billing header leaked into instructions: %q", got.Request.Instructions)
	}
	if !strings.Contains(got.Request.Instructions, "Treat the follow-up after tool results as a continuation of the same request") {
		t.Fatalf("compatibility instructions missing: %q", got.Request.Instructions)
	}
	if !strings.Contains(got.Request.Instructions, "the follow-up after tool results must not greet again or restart the conversation") {
		t.Fatalf("same-turn greeting guard missing: %q", got.Request.Instructions)
	}
	if !strings.Contains(got.Request.Instructions, "This applies even when session, skill, project, or global instructions normally require an initial greeting or setup message") {
		t.Fatalf("setup continuation guard missing: %q", got.Request.Instructions)
	}
	if !strings.Contains(got.Request.Instructions, "Treat that directory as an implementation sidecar") {
		t.Fatalf("sidecar path guidance missing: %q", got.Request.Instructions)
	}
	if got.Request.PromptCacheKey != "session-1" {
		t.Fatalf("prompt_cache_key = %q", got.Request.PromptCacheKey)
	}
	if got.Stream {
		t.Fatalf("omitted Anthropic stream flag mapped to streaming response")
	}
	if !got.Request.Stream {
		t.Fatalf("Codex upstream request must stay streaming for conversion")
	}
}

func TestAnthropicToCodexAddsCompatibilityInstructionsWithoutSystemPrompt(t *testing.T) {
	var req AnthropicRequest
	if err := json.Unmarshal([]byte(`{"messages":[{"role":"user","content":"hello"}]}`), &req); err != nil {
		t.Fatal(err)
	}
	got, err := AnthropicToCodex(req, ConvertOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got.Request.Instructions, "Claude Code compatibility:\n") {
		t.Fatalf("instructions = %q", got.Request.Instructions)
	}
	if !strings.Contains(got.Request.Instructions, "For tool calls, omit optional fields unless they have meaningful values.") {
		t.Fatalf("tool argument guidance missing: %q", got.Request.Instructions)
	}
}

func TestAnthropicToCodexFoldsSystemRoleMessagesIntoInstructions(t *testing.T) {
	var req AnthropicRequest
	if err := json.Unmarshal([]byte(`{
		"system":"base system",
		"messages":[
			{"role":"system","content":[
				{"type":"text","text":"message system"},
				{"type":"text","text":"x-anthropic-billing-header: cch=secret"}
			]},
			{"role":"user","content":"hello"}
		]
	}`), &req); err != nil {
		t.Fatal(err)
	}
	got, err := AnthropicToCodex(req, ConvertOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got.Request.Instructions, "base system\n\nmessage system\n\nClaude Code compatibility:\n") {
		t.Fatalf("instructions = %q", got.Request.Instructions)
	}
	if strings.Contains(got.Request.Instructions, "cch=secret") {
		t.Fatalf("billing header leaked into system-role instructions: %q", got.Request.Instructions)
	}
	if len(got.Request.Input) != 1 || got.Request.Input[0].Role != "user" {
		t.Fatalf("system role leaked into input: %#v", got.Request.Input)
	}
}

func TestAnthropicToCodexUsesConfiguredModelTargets(t *testing.T) {
	var req AnthropicRequest
	if err := json.Unmarshal([]byte(`{
		"model":"claude-sonnet-4-6",
		"output_config":{"effort":"max"},
		"messages":[{"role":"user","content":"hello"}]
	}`), &req); err != nil {
		t.Fatal(err)
	}
	got, err := AnthropicToCodex(req, ConvertOptions{
		Models: modelconfig.Config{Sonnet: "gpt-sonnet-next"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Request.Model != "gpt-sonnet-next" {
		t.Fatalf("model = %q", got.Request.Model)
	}
	if got.Request.Reasoning == nil || got.Request.Reasoning.Effort != "xhigh" {
		t.Fatalf("effort = %#v, want xhigh", got.Request.Reasoning)
	}
}

func TestAnthropicToCodexMapsFastSpeedToPriorityServiceTier(t *testing.T) {
	var req AnthropicRequest
	if err := json.Unmarshal([]byte(`{
		"model":"claude-opus-4-6",
		"speed":"fast",
		"messages":[{"role":"user","content":"hello"}]
	}`), &req); err != nil {
		t.Fatal(err)
	}
	got, err := AnthropicToCodex(req, ConvertOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Request.Model != "gpt-5.5" {
		t.Fatalf("model = %q, want gpt-5.5", got.Request.Model)
	}
	if got.Request.ServiceTier != "priority" {
		t.Fatalf("service_tier = %q, want priority", got.Request.ServiceTier)
	}
}

func TestAnthropicToCodexMapsStructuredOutputFormat(t *testing.T) {
	body := []byte(`{
		"model":"claude-opus-4-6",
		"output_config":{"format":{"type":"json_schema","schema":{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"]}}},
		"messages":[{"role":"user","content":"hello"}]
	}`)
	var req AnthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	got, err := AnthropicToCodex(req, ConvertOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Request.Text == nil || got.Request.Text.Format == nil {
		t.Fatalf("text format missing: %#v", got.Request.Text)
	}
	format := got.Request.Text.Format
	if format.Type != "json_schema" || format.Name != "claudodex_response" || format.Strict == nil || *format.Strict != true {
		t.Fatalf("format = %#v", format)
	}
	if format.Schema["type"] != "object" {
		t.Fatalf("schema = %#v", format.Schema)
	}
}

func TestAnthropicToCodexMapsJSONOutputFormat(t *testing.T) {
	body := []byte(`{
		"output_config":{"format":{"type":"json_object"}},
		"messages":[{"role":"user","content":"hello"}]
	}`)
	var req AnthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	got, err := AnthropicToCodex(req, ConvertOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Request.Text == nil || got.Request.Text.Format == nil || got.Request.Text.Format.Type != "json_object" {
		t.Fatalf("format = %#v", got.Request.Text)
	}
}

func TestAnthropicToCodexPreservesMixedTextAndScreenshotUserInput(t *testing.T) {
	body := []byte(`{
		"messages":[{"role":"user","content":[
			{"type":"text","text":"Here is the failing screen before the click:"},
			{"type":"image","detail":"high","source":{"type":"base64","media_type":"image/png","data":"SCREENSHOT"}},
			{"type":"text","text":"The submit button should be enabled."}
		]}]
	}`)
	var req AnthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	got, err := AnthropicToCodex(req, ConvertOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Request.Input) != 1 {
		t.Fatalf("input len = %d, want one mixed user message: %#v", len(got.Request.Input), got.Request.Input)
	}
	parts := got.Request.Input[0].Content
	if len(parts) != 3 {
		t.Fatalf("parts len = %d, want 3: %#v", len(parts), parts)
	}
	if parts[0].Type != "input_text" || parts[0].Text != "Here is the failing screen before the click:" {
		t.Fatalf("first text part = %#v", parts[0])
	}
	if parts[1].Type != "input_image" || parts[1].ImageURL != "data:image/png;base64,SCREENSHOT" || parts[1].Detail != "high" {
		t.Fatalf("screenshot part = %#v", parts[1])
	}
	if parts[2].Type != "input_text" || parts[2].Text != "The submit button should be enabled." {
		t.Fatalf("second text part = %#v", parts[2])
	}
}

func TestAnthropicToCodexConvertsToolsImagesAndResults(t *testing.T) {
	body := []byte(`{
		"model":"claude-haiku-4-5",
		"messages":[
			{"role":"user","content":[
				{"type":"text","text":"look"},
				{"type":"image","source":{"type":"base64","media_type":"image/png","data":"AAA"}}
			]},
			{"role":"assistant","content":[
				{"type":"text","text":"checking"},
				{"type":"tool_use","id":"toolu_123","name":"read_file","input":{"path":"a.go"}}
			]},
			{"role":"user","content":[
				{"type":"tool_result","tool_use_id":"toolu_123","content":[{"type":"text","text":"ok"}]}
			]}
		],
		"tools":[{"name":"read_file","description":"read","input_schema":{"type":"object","properties":{"path":{"type":"string"}}}}],
		"tool_choice":{"type":"tool","name":"read_file"}
	}`)
	var req AnthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	got, err := AnthropicToCodex(req, ConvertOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Request.Model != "gpt-5.4-mini" {
		t.Fatalf("model = %q", got.Request.Model)
	}
	if got.Request.Reasoning == nil || got.Request.Reasoning.Effort != "low" {
		t.Fatalf("effort = %#v, want low", got.Request.Reasoning)
	}
	if len(got.Request.Input) != 4 {
		t.Fatalf("input len = %d, want 4: %#v", len(got.Request.Input), got.Request.Input)
	}
	if got.Request.Input[0].Content[1].ImageURL != "data:image/png;base64,AAA" {
		t.Fatalf("image url = %q", got.Request.Input[0].Content[1].ImageURL)
	}
	if got.Request.Input[2].Type != "function_call" || got.Request.Input[2].CallID != "toolu_123" {
		t.Fatalf("function call = %#v", got.Request.Input[2])
	}
	if got.Request.Input[3].Type != "function_call_output" || got.Request.Input[3].Output != "ok" {
		t.Fatalf("function output = %#v", got.Request.Input[3])
	}
	choice, ok := got.Request.ToolChoice.(map[string]string)
	if !ok || choice["type"] != "function" || choice["name"] != "read_file" {
		t.Fatalf("tool_choice = %#v", got.Request.ToolChoice)
	}
}

func TestAnthropicToCodexKeepsImageToolResultsOnFunctionOutput(t *testing.T) {
	body := []byte(`{
		"messages":[
			{"role":"assistant","content":[
				{"type":"tool_use","id":"toolu_img","name":"view","input":{}}
			]},
			{"role":"user","content":[
				{"type":"tool_result","tool_use_id":"toolu_img","content":[
					{"type":"text","text":"screenshot"},
					{"type":"image","detail":"high","source":{"type":"base64","media_type":"image/png","data":"AAA"}}
				]}
			]}
		]
	}`)
	var req AnthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	got, err := AnthropicToCodex(req, ConvertOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Request.Input) != 2 {
		t.Fatalf("input len = %d, want 2: %#v", len(got.Request.Input), got.Request.Input)
	}
	output, ok := got.Request.Input[1].Output.([]codex.ContentPart)
	if !ok {
		t.Fatalf("tool output = %#v, want []codex.ContentPart", got.Request.Input[1].Output)
	}
	if len(output) != 2 {
		t.Fatalf("output len = %d, want 2: %#v", len(output), output)
	}
	if output[0].Type != "input_text" || output[0].Text != "screenshot" {
		t.Fatalf("text output = %#v", output[0])
	}
	if output[1].Type != "input_image" || output[1].ImageURL != "data:image/png;base64,AAA" || output[1].Detail != "high" {
		t.Fatalf("image output = %#v", output[1])
	}
}

func TestAnthropicToCodexConvertsServerToolResults(t *testing.T) {
	longID := "srv_" + "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"
	body := []byte(`{
		"model":"claude-opus-4-6",
		"messages":[
			{"role":"assistant","content":[
				{"type":"server_tool_use","id":` + jsonString(longID) + `,"name":"advisor","input":{"question":"inspect"}},
				{"type":"advisor_tool_result","tool_use_id":` + jsonString(longID) + `,"content":{"type":"advisor_result","text":"reviewed"}}
			]}
		]
	}`)
	var req AnthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	got, err := AnthropicToCodex(req, ConvertOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Request.Input) != 2 {
		t.Fatalf("input len = %d, want 2: %#v", len(got.Request.Input), got.Request.Input)
	}
	wantID := ClampCallID(longID)
	if got.Request.Input[0].Type != "function_call" || got.Request.Input[0].CallID != wantID {
		t.Fatalf("server function call = %#v, want call_id %q", got.Request.Input[0], wantID)
	}
	if got.Request.Input[1].Type != "function_call_output" || got.Request.Input[1].CallID != wantID {
		t.Fatalf("server function output = %#v, want call_id %q", got.Request.Input[1], wantID)
	}
	if got.Request.Input[1].Output != "reviewed" {
		t.Fatalf("server function output value = %#v, want reviewed", got.Request.Input[1].Output)
	}
}

func TestAnthropicToCodexConvertsServerToolResultWithoutIDToExplicitError(t *testing.T) {
	body := []byte(`{
		"messages":[{"role":"assistant","content":[
			{"type":"web_search_tool_result","content":[{"type":"text","text":"found"}]}
		]}]
	}`)
	var req AnthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	got, err := AnthropicToCodex(req, ConvertOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Request.Input) != 1 {
		t.Fatalf("input len = %d, want 1: %#v", len(got.Request.Input), got.Request.Input)
	}
	item := got.Request.Input[0]
	if item.Type != "function_call_output" || item.CallID != "call_missing_web_search" {
		t.Fatalf("server function output = %#v", item)
	}
	output, _ := item.Output.(string)
	if output == "" || !strings.Contains(output, "tool_use_id") || !strings.Contains(output, "web_search_tool_result") {
		t.Fatalf("output = %#v, want explicit missing tool_use_id error", item.Output)
	}
}

func TestSanitizeSchemaMergesCombiners(t *testing.T) {
	got := sanitizeSchema(map[string]any{
		"type":       "object",
		"not":        map[string]any{"required": []any{"blocked"}},
		"required":   []any{"root"},
		"properties": map[string]any{"root": map[string]any{"type": "string"}},
		"allOf": []any{
			map[string]any{
				"type":       "object",
				"required":   []any{"a"},
				"properties": map[string]any{"a": map[string]any{"type": "string"}},
			},
			map[string]any{
				"required":   []any{"b"},
				"properties": map[string]any{"b": map[string]any{"type": "number"}},
			},
		},
		"oneOf": []any{
			map[string]any{
				"required":   []any{"common", "c"},
				"properties": map[string]any{"common": map[string]any{"type": "string"}, "c": map[string]any{"type": "string"}},
			},
			map[string]any{
				"required":   []any{"common", "d"},
				"properties": map[string]any{"common": map[string]any{"description": "shared"}, "d": map[string]any{"type": "string"}},
			},
		},
		"anyOf": []any{
			map[string]any{
				"required":   []any{"either"},
				"properties": map[string]any{"either": map[string]any{"type": "boolean"}},
			},
			map[string]any{
				"properties": map[string]any{"optional": map[string]any{"type": "string"}},
			},
		},
	})
	if got["type"] != "object" {
		t.Fatalf("type = %#v", got["type"])
	}
	for _, key := range []string{"not", "allOf", "oneOf", "anyOf"} {
		if _, ok := got[key]; ok {
			t.Fatalf("%s was not removed: %#v", key, got)
		}
	}
	props := got["properties"].(map[string]any)
	for _, key := range []string{"root", "a", "b", "common", "c", "d", "either", "optional"} {
		if _, ok := props[key]; !ok {
			t.Fatalf("properties missing %s: %#v", key, props)
		}
	}
	for _, key := range []string{"root", "a", "b", "common"} {
		if !requiredHas(got["required"], key) {
			t.Fatalf("required missing %s: %#v", key, got["required"])
		}
	}
	for _, key := range []string{"c", "d", "either", "optional"} {
		if requiredHas(got["required"], key) {
			t.Fatalf("required unexpectedly contains %s: %#v", key, got["required"])
		}
	}
}

func TestSanitizeSchemaDropsScalarRootEnum(t *testing.T) {
	got := sanitizeSchema(map[string]any{
		"type": "string",
		"enum": []any{"read", "write"},
	})
	if got["type"] != "object" {
		t.Fatalf("type = %#v", got["type"])
	}
	if _, ok := got["enum"]; ok {
		t.Fatalf("enum should be dropped: %#v", got)
	}
	if _, ok := got["properties"].(map[string]any); !ok {
		t.Fatalf("properties missing: %#v", got)
	}
}

func requiredHas(value any, want string) bool {
	values, _ := value.([]any)
	for _, item := range values {
		if item == want {
			return true
		}
	}
	return false
}
