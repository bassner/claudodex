package convert

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

type AnthropicSSE struct {
	Event string
	Data  map[string]any
}

type Usage struct {
	InputTokens              int `json:"input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	OutputTokens             int `json:"output_tokens"`
}

type StreamReducer struct {
	messageID           string
	model               string
	toolSchemas         map[string]map[string]any
	fallbackInputTokens int
	started             bool
	done                bool
	nextIndex           int
	textActive          bool
	textIndex           int
	textSawDelta        bool
	visibleBlocks       int
	outputChars         int
	toolByOutput        map[int]*toolStreamState
	toolByItemID        map[string]*toolStreamState
	toolByCallID        map[string]*toolStreamState
	toolBlocks          int
	usage               Usage
	failed              bool
	failureType         string
	failureMessage      string
}

type toolStreamState struct {
	outputIndex int
	blockIndex  int
	itemID      string
	callID      string
	name        string
	args        strings.Builder
	active      bool
	sawDelta    bool
	sentArgs    bool
	sentStart   bool
}

func NewStreamReducer(messageID, model string) *StreamReducer {
	return NewStreamReducerWithOptions(messageID, model, StreamReducerOptions{})
}

type StreamReducerOptions struct {
	ToolSchemas         map[string]map[string]any
	FallbackInputTokens int
}

func NewStreamReducerWithOptions(messageID, model string, opts StreamReducerOptions) *StreamReducer {
	if messageID == "" {
		messageID = "msg_claudodex"
	}
	if model == "" {
		model = modelconfig.DefaultClaudeRequestModel
	}
	return &StreamReducer{
		messageID:           messageID,
		model:               model,
		toolSchemas:         cloneToolSchemas(opts.ToolSchemas),
		fallbackInputTokens: opts.FallbackInputTokens,
		textIndex:           -1,
		toolByOutput:        map[int]*toolStreamState{},
		toolByItemID:        map[string]*toolStreamState{},
		toolByCallID:        map[string]*toolStreamState{},
	}
}

func (r *StreamReducer) Done() bool {
	return r.done
}

func (r *StreamReducer) Usage() Usage {
	return r.usage
}

func (r *StreamReducer) Failed() bool {
	return r.failed
}

func (r *StreamReducer) FailureType() string {
	if r.failureType == "" {
		return "api_error"
	}
	return r.failureType
}

func (r *StreamReducer) FailureMessage() string {
	if r.failureMessage == "" {
		return "Codex upstream returned an error"
	}
	return r.failureMessage
}

func (r *StreamReducer) Reduce(raw json.RawMessage) ([]AnthropicSSE, error) {
	return r.ReduceNamed("", raw)
}

func (r *StreamReducer) ReduceNamed(name string, raw json.RawMessage) ([]AnthropicSSE, error) {
	if r.done {
		return nil, nil
	}
	var event map[string]any
	if err := json.Unmarshal(raw, &event); err != nil {
		return r.errorEvents("api_error", "Codex upstream returned malformed SSE"), nil
	}
	eventType, _ := event["type"].(string)
	if eventType == "" {
		eventType, _ = event["event"].(string)
	}
	if eventType == "" {
		eventType = name
	}
	if eventType == "" {
		return nil, nil
	}
	if eventType == "error" {
		return r.errorFromPayload(event), nil
	}

	events := r.ensureStarted(event)
	switch eventType {
	case "response.created":
		return events, nil
	case "response.output_item.added":
		item, _ := event["item"].(map[string]any)
		switch itemType(item) {
		case "message", "output_text":
			events = append(events, r.ensureTextBlock()...)
		case "function_call", "output_tool_call":
			events = append(events, r.ensureToolBlock(event, item)...)
		}
	case "response.output_text.delta":
		text := stringField(event["delta"])
		if text != "" {
			events = append(events, r.ensureTextBlock()...)
			events = append(events, contentBlockDelta(r.textIndex, map[string]any{
				"type": "text_delta",
				"text": text,
			}))
			r.textSawDelta = true
			r.outputChars += len(text)
		}
	case "response.output_text.done":
		events = append(events, r.stopTextBlock()...)
	case "response.function_call_arguments.delta":
		state := r.toolStateForEvent(event)
		events = append(events, r.startToolState(state)...)
		delta := stringField(event["delta"])
		if delta != "" {
			state.args.WriteString(delta)
			state.sawDelta = true
			r.outputChars += len(delta)
			if !r.shouldBufferToolArgs(state) {
				state.sentArgs = true
				events = append(events, contentBlockDelta(state.blockIndex, map[string]any{
					"type":         "input_json_delta",
					"partial_json": delta,
				}))
			}
		}
	case "response.function_call_arguments.done":
		state := r.toolStateForEvent(event)
		events = append(events, r.startToolState(state)...)
		if args := stringField(event["arguments"]); args != "" && !state.sawDelta {
			state.args.WriteString(args)
			state.sawDelta = true
			r.outputChars += len(args)
			if !r.shouldBufferToolArgs(state) {
				state.sentArgs = true
				events = append(events, contentBlockDelta(state.blockIndex, map[string]any{
					"type":         "input_json_delta",
					"partial_json": args,
				}))
			}
		}
	case "response.output_item.done":
		item, _ := event["item"].(map[string]any)
		switch itemType(item) {
		case "message", "output_text":
			events = append(events, r.finishMessageItem(item)...)
		case "function_call", "output_tool_call":
			state := r.toolStateForItem(event, item)
			events = append(events, r.startToolState(state)...)
			if args := stringField(item["arguments"]); args != "" && !state.sawDelta {
				state.args.WriteString(args)
				state.sawDelta = true
				r.outputChars += len(args)
				if !r.shouldBufferToolArgs(state) {
					state.sentArgs = true
					events = append(events, contentBlockDelta(state.blockIndex, map[string]any{
						"type":         "input_json_delta",
						"partial_json": args,
					}))
				}
			}
			events = append(events, r.stopToolState(state)...)
		}
	case "response.completed", "response.done":
		events = append(events, r.finish(event, "")...)
	case "response.incomplete":
		if r.visibleBlocks > 0 {
			events = append(events, r.finish(event, "max_tokens")...)
		} else {
			events = append(events, r.errorEvents("api_error", "Codex response ended incomplete before visible output")...)
		}
	case "response.failed":
		events = append(events, r.errorEvents("api_error", failureMessage(event))...)
	default:
		// Ignore reasoning, metadata, model-verification and rate-limit events for
		// Anthropic visible block indexing.
	}
	return events, nil
}

func (r *StreamReducer) ensureStarted(event map[string]any) []AnthropicSSE {
	if r.started {
		return nil
	}
	if response, _ := event["response"].(map[string]any); response != nil {
		if id, _ := response["id"].(string); id != "" && r.messageID == "msg_claudodex" {
			r.messageID = id
		}
	}
	usage := r.usageForStart(event)
	r.started = true
	return []AnthropicSSE{{
		Event: "message_start",
		Data: map[string]any{
			"type": "message_start",
			"message": map[string]any{
				"id":            r.messageID,
				"type":          "message",
				"role":          "assistant",
				"model":         r.model,
				"content":       []any{},
				"stop_reason":   nil,
				"stop_sequence": nil,
				"usage":         usage,
			},
		},
	}}
}

func (r *StreamReducer) ensureTextBlock() []AnthropicSSE {
	if r.textActive {
		return nil
	}
	r.textIndex = r.nextIndex
	r.nextIndex++
	r.textActive = true
	r.textSawDelta = false
	return []AnthropicSSE{{
		Event: "content_block_start",
		Data: map[string]any{
			"type":          "content_block_start",
			"index":         r.textIndex,
			"content_block": map[string]any{"type": "text", "text": ""},
		},
	}}
}

func (r *StreamReducer) stopTextBlock() []AnthropicSSE {
	if !r.textActive {
		return nil
	}
	index := r.textIndex
	r.textActive = false
	r.textIndex = -1
	r.visibleBlocks++
	return []AnthropicSSE{{
		Event: "content_block_stop",
		Data:  map[string]any{"type": "content_block_stop", "index": index},
	}}
}

func (r *StreamReducer) finishMessageItem(item map[string]any) []AnthropicSSE {
	var events []AnthropicSSE
	text := outputTextFromItem(item)
	if text != "" && !r.textSawDelta {
		events = append(events, r.ensureTextBlock()...)
		events = append(events, contentBlockDelta(r.textIndex, map[string]any{
			"type": "text_delta",
			"text": text,
		}))
		r.outputChars += len(text)
	}
	events = append(events, r.stopTextBlock()...)
	return events
}

func (r *StreamReducer) toolStateForEvent(event map[string]any) *toolStreamState {
	index := intField(event["output_index"], 0)
	if itemID, _ := event["item_id"].(string); itemID != "" {
		if state := r.toolByItemID[itemID]; state != nil {
			return state
		}
	}
	if callID, _ := event["call_id"].(string); callID != "" {
		if state := r.toolByCallID[callID]; state != nil {
			return state
		}
	}
	state := r.ensureToolState(index)
	r.applyToolFields(state, event)
	return state
}

func (r *StreamReducer) toolStateForItem(event, item map[string]any) *toolStreamState {
	index := intField(event["output_index"], intField(item["output_index"], 0))
	if itemID, _ := item["id"].(string); itemID != "" {
		if state := r.toolByItemID[itemID]; state != nil {
			r.applyToolFields(state, item)
			return state
		}
	}
	if callID, _ := item["call_id"].(string); callID != "" {
		if state := r.toolByCallID[callID]; state != nil {
			r.applyToolFields(state, item)
			return state
		}
	}
	state := r.ensureToolState(index)
	r.applyToolFields(state, item)
	return state
}

func (r *StreamReducer) ensureToolBlock(event, item map[string]any) []AnthropicSSE {
	state := r.toolStateForItem(event, item)
	return r.startToolState(state)
}

func (r *StreamReducer) ensureToolState(outputIndex int) *toolStreamState {
	if state := r.toolByOutput[outputIndex]; state != nil {
		return state
	}
	state := &toolStreamState{outputIndex: outputIndex, blockIndex: -1}
	r.toolByOutput[outputIndex] = state
	return state
}

func (r *StreamReducer) applyToolFields(state *toolStreamState, fields map[string]any) {
	if itemID, _ := fields["id"].(string); itemID != "" {
		state.itemID = itemID
		r.toolByItemID[itemID] = state
	}
	if itemID, _ := fields["item_id"].(string); itemID != "" {
		state.itemID = itemID
		r.toolByItemID[itemID] = state
	}
	if callID, _ := fields["call_id"].(string); callID != "" {
		state.callID = callID
		r.toolByCallID[callID] = state
	}
	if name, _ := fields["name"].(string); name != "" {
		state.name = name
	}
}

func (r *StreamReducer) startToolState(state *toolStreamState) []AnthropicSSE {
	if state.active {
		return nil
	}
	if state.blockIndex < 0 {
		state.blockIndex = r.nextIndex
		r.nextIndex++
	}
	state.active = true
	if r.shouldBufferToolArgs(state) {
		return nil
	}
	return r.startToolBlock(state)
}

func (r *StreamReducer) startToolBlock(state *toolStreamState) []AnthropicSSE {
	if state.sentStart {
		return nil
	}
	state.sentStart = true
	id := state.callID
	if id == "" {
		id = state.itemID
	}
	if id == "" {
		id = fmt.Sprintf("call_%d", state.outputIndex)
	}
	name := state.name
	if name == "" {
		name = "tool"
	}
	return []AnthropicSSE{{
		Event: "content_block_start",
		Data: map[string]any{
			"type":  "content_block_start",
			"index": state.blockIndex,
			"content_block": map[string]any{
				"type":  "tool_use",
				"id":    id,
				"name":  name,
				"input": map[string]any{},
			},
		},
	}}
}

func (r *StreamReducer) stopToolState(state *toolStreamState) []AnthropicSSE {
	if !state.active {
		return nil
	}
	state.active = false
	r.visibleBlocks++
	r.toolBlocks++
	events := make([]AnthropicSSE, 0, 3)
	events = append(events, r.startToolBlock(state)...)
	if r.shouldBufferToolArgs(state) && !state.sentArgs {
		if args := r.finalToolArgs(state); args != "" {
			state.sentArgs = true
			events = append(events, contentBlockDelta(state.blockIndex, map[string]any{
				"type":         "input_json_delta",
				"partial_json": args,
			}))
		}
	}
	events = append(events, AnthropicSSE{
		Event: "content_block_stop",
		Data:  map[string]any{"type": "content_block_stop", "index": state.blockIndex},
	})
	return events
}

func (r *StreamReducer) shouldBufferToolArgs(state *toolStreamState) bool {
	return len(r.toolSchemas) > 0
}

func (r *StreamReducer) finalToolArgs(state *toolStreamState) string {
	raw := strings.TrimSpace(state.args.String())
	if raw == "" {
		return ""
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(raw), &args); err != nil || args == nil {
		return raw
	}
	if schema := r.toolSchemas[state.name]; schema != nil {
		args = pruneEmptyOptionalToolArgs(args, schema)
	}
	data, err := json.Marshal(args)
	if err != nil {
		return raw
	}
	return string(data)
}

func (r *StreamReducer) finish(event map[string]any, forcedStop string) []AnthropicSSE {
	events := r.stopTextBlock()
	states := make([]*toolStreamState, 0, len(r.toolByOutput))
	for _, state := range r.toolByOutput {
		states = append(states, state)
	}
	sort.Slice(states, func(i, j int) bool {
		return states[i].outputIndex < states[j].outputIndex
	})
	for _, state := range states {
		events = append(events, r.stopToolState(state)...)
	}
	r.usage = r.usageForFinish(event)
	stopReason := forcedStop
	if stopReason == "" {
		stopReason = stopReasonFromEvent(event, r.toolBlocks > 0)
	}
	events = append(events, AnthropicSSE{
		Event: "message_delta",
		Data: map[string]any{
			"type": "message_delta",
			"delta": map[string]any{
				"stop_reason":   stopReason,
				"stop_sequence": nil,
			},
			"usage": r.usage,
		},
	}, AnthropicSSE{
		Event: "message_stop",
		Data:  map[string]any{"type": "message_stop"},
	})
	r.done = true
	return events
}

func (r *StreamReducer) errorFromPayload(event map[string]any) []AnthropicSSE {
	payload, _ := event["error"].(map[string]any)
	typ, _ := payload["type"].(string)
	if typ == "" {
		typ = "api_error"
	}
	message, _ := payload["message"].(string)
	if message == "" {
		message = "Codex upstream returned an error"
	}
	return r.errorEvents(typ, message)
}

func (r *StreamReducer) errorEvents(typ, message string) []AnthropicSSE {
	if typ == "" {
		typ = "api_error"
	}
	if message == "" {
		message = "Codex upstream returned an error"
	}
	r.failed = true
	r.failureType = typ
	r.failureMessage = message
	r.done = true
	return []AnthropicSSE{{
		Event: "error",
		Data: map[string]any{
			"type": "error",
			"error": map[string]any{
				"type":    typ,
				"message": message,
			},
		},
	}}
}

func contentBlockDelta(index int, delta map[string]any) AnthropicSSE {
	return AnthropicSSE{
		Event: "content_block_delta",
		Data: map[string]any{
			"type":  "content_block_delta",
			"index": index,
			"delta": delta,
		},
	}
}

func zeroUsage() Usage {
	return Usage{}
}

func usageFromEvent(event map[string]any) Usage {
	response, _ := event["response"].(map[string]any)
	usage, _ := response["usage"].(map[string]any)
	if usage == nil {
		usage, _ = event["usage"].(map[string]any)
	}
	return usageFromMap(usage)
}

func usageFromMap(usage map[string]any) Usage {
	if usage == nil {
		return Usage{}
	}
	input := intField(usage["input_tokens"], 0)
	if input == 0 {
		input = firstIntField(usage, "prompt_tokens")
	}
	output := firstIntField(usage, "output_tokens", "completion_tokens")
	if input == 0 {
		if total := firstIntField(usage, "total_tokens"); total > 0 {
			input = total
			if output > 0 && total > output {
				input = total - output
			}
		}
	}
	details, _ := usage["input_tokens_details"].(map[string]any)
	if details == nil {
		details, _ = usage["prompt_tokens_details"].(map[string]any)
	}
	cached := firstIntField(details, "cached_tokens")
	cachedIncludedInInput := cached > 0
	if cached == 0 {
		cached = firstIntField(usage, "cached_tokens", "cached_input_tokens", "input_cached_tokens")
		cachedIncludedInInput = cached > 0
	}
	if cached > input {
		cached = input
	}
	cacheCreation := firstIntField(usage, "cache_creation_input_tokens", "cache_creation_tokens")
	cacheRead := firstIntField(usage, "cache_read_input_tokens", "cache_read_tokens")
	if cachedIncludedInInput {
		cacheRead = cached
		input -= cached
	}
	return Usage{
		InputTokens:              input,
		CacheCreationInputTokens: cacheCreation,
		CacheReadInputTokens:     cacheRead,
		OutputTokens:             output,
	}
}

func (r *StreamReducer) usageForFinish(event map[string]any) Usage {
	usage := usageWithInputFloor(usageFromEvent(event), r.fallbackInputTokens)
	if r.visibleBlocks > 0 && usage.OutputTokens == 0 {
		usage.OutputTokens = estimateVisibleOutputTokens(r.outputChars, r.visibleBlocks)
	}
	return usage
}

func (r *StreamReducer) usageForStart(event map[string]any) Usage {
	usage := usageWithInputFloor(usageFromEvent(event), r.fallbackInputTokens)
	// Claude Code observes message_start usage before message_delta arrives.
	// Keep output at zero here so per-content-block progress tracking does not
	// count the same completion tokens once per streamed block.
	usage.OutputTokens = 0
	return usage
}

func usageInputTokens(usage Usage) int {
	return usage.InputTokens + usage.CacheCreationInputTokens + usage.CacheReadInputTokens
}

func usageWithInputFloor(usage Usage, floor int) Usage {
	if missing := floor - usageInputTokens(usage); missing > 0 {
		usage.InputTokens += missing
	}
	return usage
}

func estimateVisibleOutputTokens(chars, visibleBlocks int) int {
	if chars > 0 {
		return (chars + 2) / 3
	}
	if visibleBlocks > 0 {
		return 1
	}
	return 0
}

func firstIntField(values map[string]any, keys ...string) int {
	if values == nil {
		return 0
	}
	for _, key := range keys {
		if value := intField(values[key], 0); value > 0 {
			return value
		}
	}
	return 0
}

func stopReasonFromEvent(event map[string]any, hasTools bool) string {
	response, _ := event["response"].(map[string]any)
	reason, _ := response["stop_reason"].(string)
	if reason == "" {
		reason, _ = response["finish_reason"].(string)
	}
	if reason == "" {
		reason, _ = event["stop_reason"].(string)
	}
	switch strings.ToLower(reason) {
	case "tool_use", "tool_calls", "function_call":
		return "tool_use"
	case "length", "max_tokens", "max_output_tokens", "incomplete":
		return "max_tokens"
	case "stop_sequence":
		return "stop_sequence"
	default:
		if hasTools {
			return "tool_use"
		}
		return "end_turn"
	}
}

func failureMessage(event map[string]any) string {
	response, _ := event["response"].(map[string]any)
	errorObj, _ := response["error"].(map[string]any)
	if msg, _ := errorObj["message"].(string); msg != "" {
		return msg
	}
	if msg, _ := event["message"].(string); msg != "" {
		return msg
	}
	return "Codex response failed"
}

func itemType(item map[string]any) string {
	typ, _ := item["type"].(string)
	return typ
}

func outputTextFromItem(item map[string]any) string {
	content, _ := item["content"].([]any)
	var parts []string
	for _, value := range content {
		block, _ := value.(map[string]any)
		typ, _ := block["type"].(string)
		if typ != "output_text" && typ != "text" {
			continue
		}
		text, _ := block["text"].(string)
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "")
}

func stringField(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case map[string]any:
		for _, key := range []string{"text", "arguments", "partial_json", "delta"} {
			if s, _ := v[key].(string); s != "" {
				return s
			}
		}
	}
	return ""
}

func cloneToolSchemas(in map[string]map[string]any) map[string]map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]map[string]any, len(in))
	for name, schema := range in {
		out[name] = cloneMap(schema)
	}
	return out
}

func pruneEmptyOptionalToolArgs(args map[string]any, schema map[string]any) map[string]any {
	if args == nil {
		return nil
	}
	out := cloneMap(args)
	pruneEmptyOptionalObject(out, schema)
	return out
}

func pruneEmptyOptionalObject(args map[string]any, schema map[string]any) {
	required := map[string]bool{}
	for _, key := range requiredStrings(schema["required"]) {
		required[key] = true
	}
	properties := objectProperties(schema)
	for key, value := range args {
		propSchema, _ := properties[key].(map[string]any)
		if !required[key] && isEmptyOptionalValue(value) {
			delete(args, key)
			continue
		}
		if propSchema == nil {
			continue
		}
		switch typed := value.(type) {
		case map[string]any:
			pruneEmptyOptionalObject(typed, propSchema)
		case []any:
			itemSchema, _ := propSchema["items"].(map[string]any)
			if itemSchema == nil {
				continue
			}
			for _, item := range typed {
				if obj, ok := item.(map[string]any); ok {
					pruneEmptyOptionalObject(obj, itemSchema)
				}
			}
		}
	}
}

func isEmptyOptionalValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return typed == ""
	default:
		return false
	}
}

func intField(value any, fallback int) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fallback
		}
		return int(v)
	case json.Number:
		i, err := v.Int64()
		if err == nil {
			return int(i)
		}
	}
	return fallback
}

type assembledBlock struct {
	Type string
	Text strings.Builder
	ID   string
	Name string
	Args strings.Builder
}

func AssembleMessage(events []AnthropicSSE, messageID, model string) (map[string]any, *AnthropicSSE) {
	if messageID == "" {
		messageID = "msg_claudodex"
	}
	var blocks []*assembledBlock
	var stopReason any = "end_turn"
	var usage any = zeroUsage()
	for _, event := range events {
		if event.Event == "error" {
			errEvent := event
			return nil, &errEvent
		}
		switch event.Event {
		case "message_start":
			if msg, _ := event.Data["message"].(map[string]any); msg != nil {
				if id, _ := msg["id"].(string); id != "" {
					messageID = id
				}
			}
		case "content_block_start":
			contentBlock, _ := event.Data["content_block"].(map[string]any)
			typ, _ := contentBlock["type"].(string)
			block := &assembledBlock{Type: typ}
			if typ == "tool_use" {
				block.ID, _ = contentBlock["id"].(string)
				block.Name, _ = contentBlock["name"].(string)
			}
			blocks = append(blocks, block)
		case "content_block_delta":
			index := intField(event.Data["index"], -1)
			if index < 0 || index >= len(blocks) {
				continue
			}
			delta, _ := event.Data["delta"].(map[string]any)
			switch blocks[index].Type {
			case "text":
				blocks[index].Text.WriteString(stringField(delta["text"]))
			case "tool_use":
				blocks[index].Args.WriteString(stringField(delta["partial_json"]))
			}
		case "message_delta":
			if delta, _ := event.Data["delta"].(map[string]any); delta != nil {
				stopReason = delta["stop_reason"]
			}
			if got := event.Data["usage"]; got != nil {
				usage = got
			}
		}
	}
	content := make([]map[string]any, 0, len(blocks))
	for _, block := range blocks {
		switch block.Type {
		case "text":
			content = append(content, map[string]any{"type": "text", "text": block.Text.String()})
		case "tool_use":
			var input map[string]any
			if err := json.Unmarshal([]byte(block.Args.String()), &input); err != nil || input == nil {
				input = map[string]any{}
			}
			content = append(content, map[string]any{
				"type":  "tool_use",
				"id":    block.ID,
				"name":  block.Name,
				"input": input,
			})
		}
	}
	return map[string]any{
		"id":            messageID,
		"type":          "message",
		"role":          "assistant",
		"model":         model,
		"content":       content,
		"stop_reason":   stopReason,
		"stop_sequence": nil,
		"usage":         usage,
	}, nil
}
