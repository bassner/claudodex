package proxy

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"

	"github.com/bassner/claudodex/internal/codex"
)

type responseChain struct {
	Request    codex.Request
	ResponseID string
	Output     []codex.InputItem
}

type responseTrace struct {
	ResponseID   string
	Output       []codex.InputItem
	argsByIndex  map[int]string
	itemByIndex  map[int]int
	itemByCallID map[string]int
}

func (s *Server) applyImplicitResume(chainKey string, request *codex.Request) bool {
	used, _, _, _ := s.applyImplicitResumeDetailed(chainKey, request)
	return used
}

func (s *Server) applyStatelessReplayDetailed(chainKey string, request *codex.Request) (bool, string, int, int) {
	chainKey = strings.TrimSpace(chainKey)
	if chainKey == "" || request == nil {
		return false, "missing_chain_key", 0, 0
	}
	s.chainsMu.Lock()
	chain, ok := s.chains[chainKey]
	s.chainsMu.Unlock()
	if !ok {
		return false, "missing_chain", 0, len(request.Input)
	}
	if !resumeCompatible(chain.Request, *request) {
		return false, "request_options_changed", len(chain.Request.Input) + len(chain.Output), len(request.Input)
	}
	trimFrom, ok := trimAfterRecordedOutput(request.Input, chain.Output)
	if !ok {
		return false, "output_prefix_mismatch", len(chain.Request.Input) + len(chain.Output), len(request.Input)
	}
	if trimFrom >= len(request.Input) {
		return false, "no_new_input", trimFrom, len(request.Input)
	}
	replayed := make([]codex.InputItem, 0, len(chain.Request.Input)+len(chain.Output)+len(request.Input)-trimFrom)
	replayed = append(replayed, chain.Request.Input...)
	replayed = append(replayed, chain.Output...)
	replayed = append(replayed, request.Input[trimFrom:]...)
	request.PreviousResponseID = ""
	request.Input = replayed
	return true, "applied", trimFrom, len(replayed)
}

func (s *Server) applyImplicitResumeDetailed(chainKey string, request *codex.Request) (bool, string, int, int) {
	chainKey = strings.TrimSpace(chainKey)
	if chainKey == "" || request == nil {
		return false, "missing_chain_key", 0, 0
	}
	s.chainsMu.Lock()
	chain, ok := s.chains[chainKey]
	s.chainsMu.Unlock()
	if !ok {
		return false, "missing_chain", 0, 0
	}
	if strings.TrimSpace(chain.ResponseID) == "" {
		return false, "missing_response_id", len(chain.Request.Input) + len(chain.Output), len(request.Input)
	}
	if !resumeCompatible(chain.Request, *request) {
		return false, "request_options_changed", len(chain.Request.Input) + len(chain.Output), len(request.Input)
	}
	prefix := make([]codex.InputItem, 0, len(chain.Request.Input)+len(chain.Output))
	prefix = append(prefix, chain.Request.Input...)
	prefix = append(prefix, chain.Output...)
	if inputHasPrefix(request.Input, prefix) {
		if len(prefix) >= len(request.Input) {
			return false, "no_new_input", len(prefix), len(request.Input)
		}
		request.PreviousResponseID = chain.ResponseID
		request.Input = append([]codex.InputItem(nil), request.Input[len(prefix):]...)
		return true, "applied", len(prefix), len(request.Input)
	}
	if trimFrom, ok := trimAfterRecordedOutput(request.Input, chain.Output); ok {
		if trimFrom >= len(request.Input) {
			return false, "no_new_input", trimFrom, len(request.Input)
		}
		request.PreviousResponseID = chain.ResponseID
		request.Input = append([]codex.InputItem(nil), request.Input[trimFrom:]...)
		return true, "applied_by_output_calls", trimFrom, len(request.Input)
	}
	return false, "input_prefix_mismatch", len(prefix), len(request.Input)
}

func (s *Server) clearImplicitResume(chainKey string) {
	chainKey = strings.TrimSpace(chainKey)
	if chainKey == "" {
		return
	}
	s.chainsMu.Lock()
	delete(s.chains, chainKey)
	s.chainsMu.Unlock()
}

func (s *Server) recordResponseChain(chainKey string, request codex.Request, trace responseTrace) {
	chainKey = strings.TrimSpace(chainKey)
	if chainKey == "" || strings.TrimSpace(trace.ResponseID) == "" {
		return
	}
	request.PreviousResponseID = ""
	s.chainsMu.Lock()
	s.chains[chainKey] = responseChain{
		Request:    request,
		ResponseID: strings.TrimSpace(trace.ResponseID),
		Output:     trace.outputInOrder(),
	}
	s.chainsMu.Unlock()
}

func resumeCompatible(previous codex.Request, current codex.Request) bool {
	previous.Input = nil
	current.Input = nil
	previous.PreviousResponseID = ""
	current.PreviousResponseID = ""
	previous.ClientMetadata = nil
	current.ClientMetadata = nil
	previous.Instructions = ""
	current.Instructions = ""
	previous.Tools = nil
	current.Tools = nil
	previous.ToolChoice = nil
	current.ToolChoice = nil
	return reflect.DeepEqual(previous, current)
}

func trimAfterRecordedOutput(input []codex.InputItem, output []codex.InputItem) (int, bool) {
	lastOutputPos := -1
	found := false
	for _, item := range output {
		var pos int
		switch item.Type {
		case "function_call":
			if strings.TrimSpace(item.CallID) == "" {
				continue
			}
			pos = lastMatchingFunctionCall(input, item)
		case "message":
			pos = lastMatchingMessage(input, item)
		default:
			continue
		}
		if pos < 0 {
			continue
		}
		if pos > lastOutputPos {
			lastOutputPos = pos
		}
		found = true
	}
	if !found || lastOutputPos < 0 {
		return 0, false
	}
	return lastOutputPos + 1, true
}

func lastMatchingMessage(input []codex.InputItem, want codex.InputItem) int {
	for i := len(input) - 1; i >= 0; i-- {
		item := input[i]
		if item.Type != "message" || item.Role != want.Role {
			continue
		}
		if reflect.DeepEqual(item.Content, want.Content) {
			return i
		}
	}
	return -1
}

func lastMatchingFunctionCall(input []codex.InputItem, want codex.InputItem) int {
	for i := len(input) - 1; i >= 0; i-- {
		item := input[i]
		if item.Type != "function_call" || item.CallID != want.CallID {
			continue
		}
		if strings.TrimSpace(want.Name) != "" && item.Name != want.Name {
			continue
		}
		return i
	}
	return -1
}

func inputHasPrefix(input []codex.InputItem, prefix []codex.InputItem) bool {
	if len(prefix) > len(input) {
		return false
	}
	for i := range prefix {
		if !reflect.DeepEqual(input[i], prefix[i]) {
			return false
		}
	}
	return true
}

func (t *responseTrace) observe(event codex.SSEEvent) {
	switch event.Event {
	case "response.created":
		var payload struct {
			Response struct {
				ID string `json:"id"`
			} `json:"response"`
		}
		if err := json.Unmarshal(event.Data, &payload); err == nil && strings.TrimSpace(payload.Response.ID) != "" {
			t.ResponseID = strings.TrimSpace(payload.Response.ID)
		}
	case "response.function_call_arguments.done":
		t.observeFunctionCallArguments(event.Data)
	case "response.output_item.done":
		t.observeOutputItemDone(event.Data)
	case "response.completed", "response.done":
		var payload struct {
			Response struct {
				ID     string            `json:"id"`
				Output []codex.InputItem `json:"output"`
			} `json:"response"`
		}
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return
		}
		if strings.TrimSpace(payload.Response.ID) != "" {
			t.ResponseID = strings.TrimSpace(payload.Response.ID)
		}
		for outputIndex, item := range payload.Response.Output {
			t.upsertOutputItem(outputIndex, item)
		}
	}
}

func (t *responseTrace) outputInOrder() []codex.InputItem {
	if len(t.Output) == 0 {
		return nil
	}
	indices := make([]int, 0, len(t.itemByIndex))
	for index := range t.itemByIndex {
		indices = append(indices, index)
	}
	sort.Ints(indices)
	ordered := make([]codex.InputItem, 0, len(t.Output))
	used := make(map[int]struct{}, len(indices))
	for _, index := range indices {
		pos := t.itemByIndex[index]
		if pos < 0 || pos >= len(t.Output) {
			continue
		}
		ordered = append(ordered, t.Output[pos])
		used[pos] = struct{}{}
	}
	for pos, item := range t.Output {
		if _, ok := used[pos]; !ok {
			ordered = append(ordered, item)
		}
	}
	return ordered
}

func (t *responseTrace) observeFunctionCallArguments(data json.RawMessage) {
	var payload struct {
		OutputIndex int    `json:"output_index"`
		Arguments   string `json:"arguments"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || payload.Arguments == "" {
		return
	}
	if t.argsByIndex == nil {
		t.argsByIndex = map[int]string{}
	}
	t.argsByIndex[payload.OutputIndex] = payload.Arguments
	if t.itemByIndex != nil {
		if pos, ok := t.itemByIndex[payload.OutputIndex]; ok && pos >= 0 && pos < len(t.Output) && t.Output[pos].Arguments == "" {
			t.Output[pos].Arguments = payload.Arguments
		}
	}
}

func (t *responseTrace) observeOutputItemDone(data json.RawMessage) {
	var payload struct {
		OutputIndex int             `json:"output_index"`
		Item        codex.InputItem `json:"item"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || strings.TrimSpace(payload.Item.Type) == "" {
		return
	}
	item := payload.Item
	if item.Type == "function_call" && item.Arguments == "" && t.argsByIndex != nil {
		item.Arguments = t.argsByIndex[payload.OutputIndex]
	}
	t.upsertOutputItem(payload.OutputIndex, item)
}

func (t *responseTrace) upsertOutputItem(outputIndex int, item codex.InputItem) {
	if t.itemByIndex == nil {
		t.itemByIndex = map[int]int{}
	}
	if t.itemByCallID == nil {
		t.itemByCallID = map[string]int{}
	}
	if pos, ok := t.itemByIndex[outputIndex]; ok && pos >= 0 && pos < len(t.Output) {
		t.Output[pos] = mergeOutputItem(t.Output[pos], item)
		if item.CallID != "" {
			t.itemByCallID[item.CallID] = pos
		}
		return
	}
	if item.CallID != "" {
		if pos, ok := t.itemByCallID[item.CallID]; ok && pos >= 0 && pos < len(t.Output) {
			t.Output[pos] = mergeOutputItem(t.Output[pos], item)
			t.itemByIndex[outputIndex] = pos
			return
		}
	}
	t.Output = append(t.Output, item)
	pos := len(t.Output) - 1
	t.itemByIndex[outputIndex] = pos
	if item.CallID != "" {
		t.itemByCallID[item.CallID] = pos
	}
}

func mergeOutputItem(previous codex.InputItem, next codex.InputItem) codex.InputItem {
	if len(next.Raw) > 0 {
		previous.Raw = next.Raw
	}
	if strings.TrimSpace(next.Type) != "" {
		previous.Type = next.Type
	}
	if strings.TrimSpace(next.Role) != "" {
		previous.Role = next.Role
	}
	if len(next.Content) > 0 {
		previous.Content = next.Content
	}
	if strings.TrimSpace(next.CallID) != "" {
		previous.CallID = next.CallID
	}
	if strings.TrimSpace(next.Name) != "" {
		previous.Name = next.Name
	}
	if strings.TrimSpace(next.Arguments) != "" {
		previous.Arguments = next.Arguments
	}
	if next.Output != nil {
		previous.Output = next.Output
	}
	return previous
}
