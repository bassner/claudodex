package codex

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const BioPolicyFallbackMessage = "This content was flagged for possible biological risk."

type ResponseFailure struct {
	Code          string
	Message       string
	RetryAfter    time.Duration
	RetryAfterSet bool
	Retryable     bool
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
		retryAfter, retryAfterSet := responseFailureRetryAfter(candidate["headers"])
		retryable, _ := candidate["retryable"].(bool)
		failure := ResponseFailure{
			Code:          strings.TrimSpace(code),
			Message:       message,
			RetryAfter:    retryAfter,
			RetryAfterSet: retryAfterSet,
			Retryable:     retryable,
		}
		if strings.EqualFold(failure.Code, "bio_policy") && strings.TrimSpace(failure.Message) == "" {
			failure.Message = BioPolicyFallbackMessage
		}
		return failure
	}
	return ResponseFailure{}
}

func responseFailureRetryAfter(raw any) (time.Duration, bool) {
	headers, ok := raw.(map[string]any)
	if !ok {
		return 0, false
	}
	for name, value := range headers {
		if !strings.EqualFold(strings.TrimSpace(name), "retry-after") {
			continue
		}
		var text string
		switch typed := value.(type) {
		case string:
			text = typed
		case []any:
			if len(typed) > 0 {
				text, _ = typed[0].(string)
			}
		}
		return parseResponseRetryAfter(text)
	}
	return 0, false
}

func parseResponseRetryAfter(value string) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.ParseFloat(value, 64); err == nil && seconds >= 0 {
		return time.Duration(seconds * float64(time.Second)), true
	}
	if retryAt, err := http.ParseTime(value); err == nil {
		delay := time.Until(retryAt)
		if delay < 0 {
			delay = 0
		}
		return delay, true
	}
	return 0, false
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
