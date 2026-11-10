package codex

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const BudgetReachedFallbackMessage = "The Codex session budget has been reached. Start a new session to continue."

const BioPolicyFallbackMessage = "This content was flagged for possible biological risk."

type ResponseFailure struct {
	Code          string
	Message       string
	Status        int
	RetryAfter    time.Duration
	RetryAfterSet bool
	Retryable     bool
}

func ParseResponseFailure(data []byte) ResponseFailure {
	var payload map[string]any
	if json.Unmarshal(data, &payload) != nil {
		return ResponseFailure{}
	}
	candidates := responseFailureCandidates(payload)
	for _, candidate := range candidates {
		code, _ := candidate["code"].(string)
		message, _ := candidate["message"].(string)
		status := responseFailureStatus(candidate["status"])
		if status == 0 {
			status = responseFailureStatus(candidate["status_code"])
		}
		if strings.TrimSpace(code) == "" && strings.TrimSpace(message) == "" && status == 0 {
			continue
		}
		// A WebSocket envelope can carry HTTP status outside its selected error.
		if status == 0 {
			for _, enclosing := range candidates {
				status = responseFailureStatus(enclosing["status"])
				if status == 0 {
					status = responseFailureStatus(enclosing["status_code"])
				}
				if status != 0 {
					break
				}
			}
		}
		var retryAfter time.Duration
		var retryAfterSet bool
		for _, headerCandidate := range candidates {
			if retryAfter, retryAfterSet = responseFailureRetryAfter(headerCandidate["headers"]); retryAfterSet {
				break
			}
		}
		retryable, _ := candidate["retryable"].(bool)
		failure := ResponseFailure{
			Code:          strings.TrimSpace(code),
			Message:       message,
			Status:        status,
			RetryAfter:    retryAfter,
			RetryAfterSet: retryAfterSet,
			Retryable:     retryable,
		}
		if strings.EqualFold(failure.Code, "bio_policy") && strings.TrimSpace(failure.Message) == "" {
			failure.Message = BioPolicyFallbackMessage
		}
		if strings.EqualFold(failure.Code, "budget_reached") && strings.TrimSpace(failure.Message) == "" {
			failure.Message = BudgetReachedFallbackMessage
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
	names := make([]string, 0, len(headers))
	for name := range headers {
		if !strings.EqualFold(strings.TrimSpace(name), "retry-after") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if delay, set := responseFailureRetryAfterValue(headers[name]); set {
			return delay, true
		}
	}
	return 0, false
}

func responseFailureRetryAfterValue(value any) (time.Duration, bool) {
	switch typed := value.(type) {
	case string:
		return parseResponseRetryAfter(typed)
	case float64:
		return parseResponseRetryAfter(strconv.FormatFloat(typed, 'f', -1, 64))
	case json.Number:
		return parseResponseRetryAfter(typed.String())
	case []any:
		for _, entry := range typed {
			if delay, set := responseFailureRetryAfterValue(entry); set {
				return delay, true
			}
		}
	}
	return 0, false
}

func responseFailureStatus(value any) int {
	switch typed := value.(type) {
	case float64:
		if typed >= 100 && typed <= 999 && typed == float64(int(typed)) {
			return int(typed)
		}
	case json.Number:
		status, _ := strconv.Atoi(typed.String())
		if status >= 100 && status <= 999 {
			return status
		}
	case string:
		status, _ := strconv.Atoi(strings.TrimSpace(typed))
		if status >= 100 && status <= 999 {
			return status
		}
	}
	return 0
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
		candidates = append(candidates, response)
	}
	return append(candidates, payload)
}
