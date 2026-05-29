package convert

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
)

type AnthropicRequest struct {
	Model        string             `json:"model"`
	System       json.RawMessage    `json:"system"`
	Messages     []AnthropicMessage `json:"messages"`
	Tools        []AnthropicTool    `json:"tools"`
	ToolChoice   json.RawMessage    `json:"tool_choice"`
	Thinking     AnthropicThinking  `json:"thinking"`
	OutputConfig OutputConfig       `json:"output_config"`
	Speed        string             `json:"speed"`
	Stream       *bool              `json:"stream"`
}

type AnthropicMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type AnthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type AnthropicThinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}

type OutputConfig struct {
	Effort string          `json:"effort"`
	Format json.RawMessage `json:"format"`
}

type ConvertOptions struct {
	SessionID string
	Models    modelconfig.Config
}

type Result struct {
	Request       codex.Request
	OriginalModel string
	Stream        bool
	ToolSchemas   map[string]map[string]any
}

type BadRequestError struct {
	Message string
}

const claudeCodeCompatibilityInstructions = `Claude Code compatibility:
You are serving as the model backend for Claude Code through an API compatibility layer. A single Claude Code user request may be fulfilled as one assistant trajectory containing visible assistant text, tool calls, tool results, and a follow-up assistant message. Treat the follow-up after tool results as a continuation of the same request, not as a fresh conversational opening.

When the latest user message contains tool results, you are continuing the same Claude Code turn. Do not start that continuation with a greeting, salutation, welcome, repeated acknowledgment, repeated setup announcement, or other conversational opening. Start directly with the result of the tools, the next required action, or the answer.

Do not repeat content already emitted earlier in the same turn. In particular, if visible assistant text before tool calls already greeted the user, acknowledged the request, described what you are about to do, or performed an opening/setup/status ritual required by instructions, the follow-up after tool results must not greet again or restart the conversation. This applies even when session, skill, project, or global instructions normally require an initial greeting or setup message: perform that opening at most once per user-visible turn, then continue without another opening.

Preserve and obey Claude Code system, project, user, skill, slash-command, and tool instructions as given.

Claudodex may run Claude Code with a compatibility config directory under .claudodex/claude-config. Treat that directory as an implementation sidecar, not as the user's canonical Claude config location. If you need to edit, inspect, or report Claude config or instruction files and the path is inside .claudodex/claude-config, resolve symlinks first and operate on the real target path, usually under .claude. Prefer showing the real target path to the user.

For tool calls, omit optional fields unless they have meaningful values.`

func (e BadRequestError) Error() string {
	return e.Message
}

func AnthropicToCodex(req AnthropicRequest, opts ConvertOptions) (Result, error) {
	models := opts.Models.Normalize()
	model := req.Model
	if strings.TrimSpace(model) == "" {
		model = modelconfig.DefaultClaudeRequestModel
	}
	codexModel := models.MapModel(model)
	input, messageInstructions, err := convertMessages(req.Messages)
	if err != nil {
		return Result{}, err
	}
	if len(input) == 0 {
		input = append(input, messageItem("user", []codex.ContentPart{{Type: "input_text", Text: ""}}))
	}
	stream := false
	if req.Stream != nil {
		stream = *req.Stream
	}
	instructions := strings.TrimSpace(systemInstructions(req.System))
	if strings.TrimSpace(messageInstructions) != "" {
		if instructions != "" {
			instructions += "\n\n"
		}
		instructions += strings.TrimSpace(messageInstructions)
	}
	effort := MapReasoningEffortWithConfig(codexModel, req.OutputConfig.Effort, req.Thinking.BudgetTokens, models)
	out := codex.Request{
		Model:             codexModel,
		Instructions:      withClaudeCodeCompatibilityInstructions(instructions),
		Input:             input,
		Tools:             convertTools(req.Tools),
		ToolChoice:        convertToolChoice(req.ToolChoice, len(req.Tools) > 0),
		ParallelToolCalls: false,
		Store:             false,
		Stream:            true,
		ServiceTier:       mapServiceTier(req.Speed),
		Reasoning:         &codex.Reasoning{Effort: string(effort)},
		Text:              convertOutputFormat(req.OutputConfig.Format),
		PromptCacheKey:    opts.SessionID,
	}
	return Result{Request: out, OriginalModel: model, Stream: stream, ToolSchemas: toolSchemas(req.Tools)}, nil
}

func mapServiceTier(speed string) string {
	if strings.EqualFold(strings.TrimSpace(speed), "fast") {
		return "priority"
	}
	return ""
}

func withClaudeCodeCompatibilityInstructions(instructions string) string {
	instructions = strings.TrimSpace(instructions)
	if instructions == "" {
		return claudeCodeCompatibilityInstructions
	}
	return instructions + "\n\n" + claudeCodeCompatibilityInstructions
}

func systemInstructions(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.TrimSpace(StripAnthropicBillingHeader(text))
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	var parts []string
	for _, block := range blocks {
		if block.Type != "" && block.Type != "text" {
			continue
		}
		stripped := strings.TrimSpace(StripAnthropicBillingHeader(block.Text))
		if stripped != "" {
			parts = append(parts, stripped)
		}
	}
	return strings.Join(parts, "\n\n")
}

func convertMessages(messages []AnthropicMessage) ([]codex.InputItem, string, error) {
	var input []codex.InputItem
	var systemTexts []string
	converter := messageConverter{callIDs: make(map[string]string)}
	for _, msg := range messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		switch role {
		case "system":
			if text := strings.TrimSpace(systemInstructions(msg.Content)); text != "" {
				systemTexts = append(systemTexts, text)
			}
			continue
		case "user", "assistant":
		default:
			return nil, "", BadRequestError{Message: fmt.Sprintf("unsupported message role %q", msg.Role)}
		}
		items, err := converter.convertContent(role, msg.Content)
		if err != nil {
			return nil, "", err
		}
		input = append(input, items...)
	}
	return input, strings.Join(systemTexts, "\n\n"), nil
}

type messageConverter struct {
	callIDs map[string]string
}

func (c *messageConverter) convertContent(role string, raw json.RawMessage) ([]codex.InputItem, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		partType := "input_text"
		if role == "assistant" {
			partType = "output_text"
		}
		return []codex.InputItem{messageItem(role, []codex.ContentPart{{Type: partType, Text: text}})}, nil
	}

	var blocks []map[string]any
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, BadRequestError{Message: "message content must be a string or array of content blocks"}
	}
	var items []codex.InputItem
	var parts []codex.ContentPart
	flush := func() {
		if len(parts) == 0 {
			return
		}
		items = append(items, messageItem(role, parts))
		parts = nil
	}

	for _, block := range blocks {
		typ, _ := block["type"].(string)
		switch typ {
		case "text":
			text, _ := block["text"].(string)
			if role == "assistant" {
				parts = append(parts, codex.ContentPart{Type: "output_text", Text: text})
			} else {
				parts = append(parts, codex.ContentPart{Type: "input_text", Text: text})
			}
		case "image":
			if role != "user" {
				continue
			}
			imagePart, err := imagePartFromBlock(block)
			if err != nil {
				return nil, err
			}
			parts = append(parts, imagePart)
		case "tool_use", "server_tool_use":
			flush()
			if role != "assistant" {
				return nil, BadRequestError{Message: "tool_use blocks are only valid in assistant messages"}
			}
			id, _ := block["id"].(string)
			name, _ := block["name"].(string)
			if name == "" {
				name, _ = block["tool_name"].(string)
			}
			if name == "" {
				name = "tool"
			}
			if id == "" {
				id = "call_" + name
			}
			callID := c.registerCallID(id)
			args, err := json.Marshal(block["input"])
			if err != nil || string(args) == "null" {
				args = []byte("{}")
			}
			items = append(items, codex.InputItem{
				Type:      "function_call",
				CallID:    callID,
				Name:      name,
				Arguments: string(args),
			})
		case "tool_result":
			flush()
			if role != "user" {
				return nil, BadRequestError{Message: "tool_result blocks are only valid in user messages"}
			}
			callID, _ := block["tool_use_id"].(string)
			if callID == "" {
				callID = "call_unknown"
			}
			resolvedCallID := c.resolveCallID(callID)
			output := toolResultOutput(block)
			items = append(items, codex.InputItem{
				Type:   "function_call_output",
				CallID: resolvedCallID,
				Output: output,
			})
		case "thinking", "redacted_thinking":
			continue
		default:
			if isServerToolResultBlock(typ, block) {
				flush()
				items = append(items, c.serverToolResultItem(typ, block))
				continue
			}
			continue
		}
	}
	flush()
	return items, nil
}

func (c *messageConverter) registerCallID(id string) string {
	clamped := ClampCallID(id)
	c.callIDs[id] = clamped
	return clamped
}

func (c *messageConverter) resolveCallID(id string) string {
	if clamped, ok := c.callIDs[id]; ok {
		return clamped
	}
	return ClampCallID(id)
}

func isServerToolResultBlock(typ string, block map[string]any) bool {
	return strings.HasSuffix(typ, "_tool_result")
}

func (c *messageConverter) serverToolResultItem(typ string, block map[string]any) codex.InputItem {
	toolUseID, _ := block["tool_use_id"].(string)
	if toolUseID != "" {
		return codex.InputItem{
			Type:   "function_call_output",
			CallID: c.resolveCallID(toolUseID),
			Output: serverToolResultOutput(block),
		}
	}
	return codex.InputItem{
		Type:   "function_call_output",
		CallID: ClampCallID("call_missing_" + strings.TrimSuffix(typ, "_tool_result")),
		Output: `{"error":"server tool result block did not include a tool_use_id","block_type":` + jsonString(typ) + `}`,
	}
}

func messageItem(role string, parts []codex.ContentPart) codex.InputItem {
	return codex.InputItem{Type: "message", Role: role, Content: parts}
}

func imagePartFromBlock(block map[string]any) (codex.ContentPart, error) {
	source, _ := block["source"].(map[string]any)
	detail, _ := block["detail"].(string)
	switch sourceType, _ := source["type"].(string); sourceType {
	case "base64":
		mediaType, _ := source["media_type"].(string)
		data, _ := source["data"].(string)
		if mediaType == "" || data == "" {
			return codex.ContentPart{}, BadRequestError{Message: "base64 image blocks require media_type and data"}
		}
		return codex.ContentPart{Type: "input_image", ImageURL: "data:" + mediaType + ";base64," + data, Detail: detail}, nil
	case "url":
		url, _ := source["url"].(string)
		if url == "" {
			return codex.ContentPart{}, BadRequestError{Message: "url image blocks require url"}
		}
		return codex.ContentPart{Type: "input_image", ImageURL: url, Detail: detail}, nil
	default:
		return codex.ContentPart{}, BadRequestError{Message: "unsupported image source type"}
	}
}

func toolResultOutput(block map[string]any) any {
	output := toolResultOutputValue(block["content"])
	if isErr, _ := block["is_error"].(bool); isErr {
		return errorToolResultOutput(output)
	}
	return output
}

func serverToolResultOutput(block map[string]any) string {
	if content, ok := block["content"]; ok {
		if obj, ok := content.(map[string]any); ok {
			if text, _ := obj["text"].(string); text != "" {
				return text
			}
			if code, _ := obj["error_code"].(string); code != "" {
				return "Error: " + code
			}
		}
		output := toolResultOutputValue(content)
		if text, ok := output.(string); ok && text != "" {
			return text
		}
	}
	data, err := json.Marshal(block)
	if err != nil {
		return fmt.Sprint(block)
	}
	return string(data)
}

func toolResultOutputValue(value any) any {
	switch v := value.(type) {
	case string:
		return v
	case []any:
		var text []string
		var parts []codex.ContentPart
		for _, item := range v {
			block, ok := item.(map[string]any)
			if !ok {
				continue
			}
			switch typ, _ := block["type"].(string); typ {
			case "text":
				if s, _ := block["text"].(string); s != "" {
					text = append(text, s)
					parts = append(parts, codex.ContentPart{Type: "input_text", Text: s})
				}
			case "image":
				if imagePart, err := imagePartFromBlock(block); err == nil {
					parts = append(parts, imagePart)
				}
			}
		}
		if hasImagePart(parts) {
			return parts
		}
		return strings.Join(text, "\n")
	default:
		if value == nil {
			return ""
		}
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprint(value)
		}
		return string(data)
	}
}

func hasImagePart(parts []codex.ContentPart) bool {
	for _, part := range parts {
		if part.Type == "input_image" {
			return true
		}
	}
	return false
}

func errorToolResultOutput(output any) any {
	switch v := output.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return "Error"
		}
		return "Error: " + v
	case []codex.ContentPart:
		if len(v) == 0 {
			return []codex.ContentPart{{Type: "input_text", Text: "Error"}}
		}
		out := append([]codex.ContentPart(nil), v...)
		for i := range out {
			if out[i].Type == "input_text" {
				if strings.TrimSpace(out[i].Text) == "" {
					out[i].Text = "Error"
				} else {
					out[i].Text = "Error: " + out[i].Text
				}
				return out
			}
		}
		return append([]codex.ContentPart{{Type: "input_text", Text: "Error"}}, out...)
	default:
		return "Error"
	}
}

func jsonString(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(data)
}

func convertOutputFormat(raw json.RawMessage) *codex.TextConfig {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil
	}
	typ, _ := obj["type"].(string)
	switch typ {
	case "json_schema":
		schema, _ := obj["schema"].(map[string]any)
		if schema == nil {
			return nil
		}
		name, _ := obj["name"].(string)
		if strings.TrimSpace(name) == "" {
			name = "claudodex_response"
		}
		strict := true
		if provided, ok := obj["strict"].(bool); ok {
			strict = provided
		}
		return &codex.TextConfig{Format: &codex.TextFormat{
			Type:   "json_schema",
			Name:   name,
			Schema: cloneMap(schema),
			Strict: &strict,
		}}
	case "json_object":
		return &codex.TextConfig{Format: &codex.TextFormat{Type: "json_object"}}
	default:
		return nil
	}
}

func convertTools(tools []AnthropicTool) []codex.Tool {
	out := make([]codex.Tool, 0, len(tools))
	for _, tool := range tools {
		if strings.TrimSpace(tool.Name) == "" {
			continue
		}
		out = append(out, codex.Tool{
			Type:        "function",
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  sanitizeSchema(tool.InputSchema),
		})
	}
	return out
}

func toolSchemas(tools []AnthropicTool) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, tool := range tools {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		out[name] = sanitizeSchema(tool.InputSchema)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func sanitizeSchema(schema map[string]any) map[string]any {
	if schema == nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	out := cloneMap(schema)
	originalType, _ := out["type"].(string)
	if originalType != "object" {
		out["type"] = "object"
		delete(out, "enum")
	}
	if _, ok := out["properties"]; !ok {
		out["properties"] = map[string]any{}
	}
	delete(out, "not")
	mergeAllOf(out)
	mergeVariantOf(out, "oneOf")
	mergeVariantOf(out, "anyOf")
	delete(out, "allOf")
	delete(out, "oneOf")
	delete(out, "anyOf")
	return out
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneSchemaValue(value)
	}
	return out
}

func cloneSchemaValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return cloneMap(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = cloneSchemaValue(item)
		}
		return out
	default:
		return value
	}
}

func mergeAllOf(dst map[string]any) {
	for _, branch := range schemaBranches(dst["allOf"]) {
		mergeObjectSchema(dst, branch)
		setRequired(dst, unionStrings(requiredStrings(dst["required"]), requiredStrings(branch["required"])))
	}
}

func mergeVariantOf(dst map[string]any, key string) {
	branches := schemaBranches(dst[key])
	if len(branches) == 0 {
		return
	}
	var intersection []string
	for i, branch := range branches {
		mergeObjectSchema(dst, branch)
		required := requiredStrings(branch["required"])
		if i == 0 {
			intersection = required
		} else {
			intersection = intersectStrings(intersection, required)
		}
	}
	setRequired(dst, unionStrings(requiredStrings(dst["required"]), intersection))
}

func schemaBranches(value any) []map[string]any {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	branches := make([]map[string]any, 0, len(values))
	for _, item := range values {
		branch, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if typ, _ := branch["type"].(string); typ != "" && typ != "object" {
			continue
		}
		branches = append(branches, branch)
	}
	return branches
}

func mergeObjectSchema(dst, branch map[string]any) {
	if description, _ := branch["description"].(string); description != "" {
		if _, exists := dst["description"]; !exists {
			dst["description"] = description
		}
	}
	dstProps := ensureProperties(dst)
	for name, value := range objectProperties(branch) {
		if existing, ok := dstProps[name].(map[string]any); ok {
			if next, ok := value.(map[string]any); ok {
				dstProps[name] = mergeMaps(existing, next)
				continue
			}
		}
		dstProps[name] = cloneSchemaValue(value)
	}
}

func ensureProperties(schema map[string]any) map[string]any {
	if props, ok := schema["properties"].(map[string]any); ok {
		return props
	}
	props := map[string]any{}
	schema["properties"] = props
	return props
}

func objectProperties(schema map[string]any) map[string]any {
	if props, ok := schema["properties"].(map[string]any); ok {
		return props
	}
	return nil
}

func mergeMaps(left, right map[string]any) map[string]any {
	out := cloneMap(left)
	for key, value := range right {
		if existing, ok := out[key].(map[string]any); ok {
			if next, ok := value.(map[string]any); ok {
				out[key] = mergeMaps(existing, next)
				continue
			}
		}
		out[key] = cloneSchemaValue(value)
	}
	return out
}

func requiredStrings(value any) []string {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, item := range values {
		if text, ok := item.(string); ok && text != "" {
			out = append(out, text)
		}
	}
	return out
}

func setRequired(schema map[string]any, required []string) {
	if len(required) == 0 {
		delete(schema, "required")
		return
	}
	out := make([]any, len(required))
	for i, value := range required {
		out[i] = value
	}
	schema["required"] = out
}

func unionStrings(left, right []string) []string {
	seen := make(map[string]bool, len(left)+len(right))
	out := make([]string, 0, len(left)+len(right))
	for _, value := range append(left, right...) {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func intersectStrings(left, right []string) []string {
	allowed := make(map[string]bool, len(right))
	for _, value := range right {
		allowed[value] = true
	}
	var out []string
	seen := map[string]bool{}
	for _, value := range left {
		if allowed[value] && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func convertToolChoice(raw json.RawMessage, hasTools bool) any {
	if !hasTools {
		return "none"
	}
	if len(raw) == 0 || string(raw) == "null" {
		return "auto"
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		switch value {
		case "auto":
			return "auto"
		case "none":
			return "none"
		case "any":
			return "required"
		default:
			return "auto"
		}
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "auto"
	}
	if typ, _ := obj["type"].(string); typ == "tool" {
		if name, _ := obj["name"].(string); name != "" {
			return map[string]string{"type": "function", "name": name}
		}
	}
	return "auto"
}
