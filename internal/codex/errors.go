package codex

import (
	"encoding/json"
	"strings"
)

const BioPolicyFallbackMessage = "This content was flagged for possible biological risk."

type ResponseFailure struct {
	Code    string
	Message string
}

func ParseResponseFailure(data []byte) ResponseFailure {
	var payload map[string]any
	if json.Unmarshal(data, &payload) != nil {
		return ResponseFailure{}
	}
	for _, candidate := range responseFailureCandidates(payload) {
		code, _ := candidate["code"].(string)
		message, _ := candidate["message"].(string)
		if strings.TrimSpace(code) == "" && strings.TrimSpace(message) == "" {
			continue
		}
		failure := ResponseFailure{Code: strings.TrimSpace(code), Message: message}
		if strings.EqualFold(failure.Code, "bio_policy") && strings.TrimSpace(failure.Message) == "" {
			failure.Message = BioPolicyFallbackMessage
		}
		return failure
	}
	return ResponseFailure{}
}

func responseFailureCandidates(payload map[string]any) []map[string]any {
	candidates := make([]map[string]any, 0, 3)
	if wrapped, _ := payload["error"].(map[string]any); wrapped != nil {
		candidates = append(candidates, wrapped)
	}
	if response, _ := payload["response"].(map[string]any); response != nil {
		if wrapped, _ := response["error"].(map[string]any); wrapped != nil {
			candidates = append(candidates, wrapped)
		}
	}
	return append(candidates, payload)
}
