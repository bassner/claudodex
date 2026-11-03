package proxy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/convert"
)

const (
	proxyTraceEnv             = "CLAUDODEX_PROXY_TRACE"
	proxyTraceIdleIntervalEnv = "CLAUDODEX_PROXY_TRACE_IDLE_INTERVAL"
)

var proxyTraceSeq atomic.Uint64

func (s *Server) nextTraceID() string {
	return fmt.Sprintf("%d-%d", os.Getpid(), proxyTraceSeq.Add(1))
}

func (s *Server) trace(event string, fields map[string]any) {
	tracePath := s.proxyTracePath()
	if tracePath == "" || strings.TrimSpace(event) == "" {
		return
	}
	entry := make(map[string]any, len(fields)+2)
	entry["time"] = time.Now().UTC().Format(time.RFC3339Nano)
	entry["event"] = event
	for key, value := range fields {
		if strings.TrimSpace(key) == "" || value == nil {
			continue
		}
		entry[key] = value
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	s.traceMu.Lock()
	defer s.traceMu.Unlock()
	if err := os.MkdirAll(filepath.Dir(tracePath), 0o700); err != nil {
		return
	}
	f, err := os.OpenFile(tracePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(data, '\n'))
}

func (s *Server) proxyTracePath() string {
	value := strings.TrimSpace(os.Getenv(proxyTraceEnv))
	switch strings.ToLower(value) {
	case "0", "false", "no", "off":
		return ""
	}
	if value != "" && !isTruthyTraceValue(value) {
		return expandHomePath(value)
	}
	if value == "" && !s.cfg.Interactive {
		return ""
	}

	dir := proxyTraceDirectory(s.cfg.Home)
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "proxy-trace-"+time.Now().Format("2006-01-02")+".jsonl")
}

func proxyTraceDirectory(home string) string {
	home = strings.TrimSpace(home)
	if home == "" {
		if userHome, err := os.UserHomeDir(); err == nil && userHome != "" {
			home = filepath.Join(userHome, ".claudodex")
		}
	}
	if home == "" {
		return ""
	}
	return filepath.Join(home, "logs")
}

// PruneTraceLogs only deletes managed daily trace files. Explicit custom trace
// paths, symlinks, directories, and other diagnostics remain user-owned.
func PruneTraceLogs(home string, now time.Time) (int, int64, error) {
	dir := proxyTraceDirectory(home)
	if dir == "" {
		return 0, 0, nil
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	cutoff := now.Add(-7 * 24 * time.Hour)
	var removed int
	var bytes int64
	var failures []error
	for _, entry := range entries {
		name := entry.Name()
		date := strings.TrimSuffix(strings.TrimPrefix(name, "proxy-trace-"), ".jsonl")
		if _, err := time.Parse("2006-01-02", date); err != nil || name != "proxy-trace-"+date+".jsonl" {
			continue
		}
		info, err := entry.Info()
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if !info.Mode().IsRegular() || !info.ModTime().Before(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			if !os.IsNotExist(err) {
				failures = append(failures, err)
			}
			continue
		}
		removed++
		bytes += info.Size()
	}
	return removed, bytes, errors.Join(failures...)
}

func isTruthyTraceValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func expandHomePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			return home
		}
		return path
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}

func mergeTraceFields(base map[string]any, extra map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(extra))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func traceDurationMS(start time.Time) int64 {
	if start.IsZero() {
		return 0
	}
	return time.Since(start).Milliseconds()
}

func proxyTraceIdleInterval() time.Duration {
	value := strings.TrimSpace(os.Getenv(proxyTraceIdleIntervalEnv))
	if value == "" {
		return 30 * time.Second
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}
	interval, err := time.ParseDuration(value)
	if err != nil || interval < 0 {
		return 30 * time.Second
	}
	return interval
}

func (s *Server) startStreamIdleTrace(base map[string]any, started time.Time) (func(string), func()) {
	if s.proxyTracePath() == "" {
		return func(string) {}, func() {}
	}
	interval := proxyTraceIdleInterval()
	if interval <= 0 {
		return func(string) {}, func() {}
	}
	activity := make(chan string, 1)
	done := make(chan struct{})
	go func() {
		lastActivity := started
		if lastActivity.IsZero() {
			lastActivity = time.Now()
		}
		lastEvent := ""
		timer := time.NewTimer(interval)
		defer timer.Stop()
		for {
			select {
			case event := <-activity:
				lastActivity = time.Now()
				lastEvent = event
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(interval)
			case <-timer.C:
				now := time.Now()
				s.trace("stream.idle", mergeTraceFields(base, map[string]any{
					"elapsed_ms": now.Sub(started).Milliseconds(),
					"idle_ms":    now.Sub(lastActivity).Milliseconds(),
					"last_event": lastEvent,
				}))
				timer.Reset(interval)
			case <-done:
				return
			}
		}
	}()
	return func(event string) {
			select {
			case activity <- event:
			default:
			}
		}, func() {
			close(done)
		}
}

func shouldTraceCodexEvent(event string) bool {
	switch event {
	case "response.created", "response.completed", "response.done", "response.incomplete", "response.failed", "error":
		return true
	default:
		return strings.HasSuffix(event, ".done")
	}
}

func (s *Server) traceStreamEvent(base map[string]any, started time.Time, event codex.SSEEvent, reducer *convert.StreamReducer, converted []convert.AnthropicSSE) {
	if !shouldTraceCodexEvent(event.Event) || s.proxyTracePath() == "" {
		return
	}
	fields := map[string]any{
		"elapsed_ms":              traceDurationMS(started),
		"codex_event":             event.Event,
		"reducer_done":            reducer.Done(),
		"reducer_needs_follow_up": reducer.NeedsFollowUp(),
		"reducer_failed":          reducer.Failed(),
	}
	if msg := eventErrorMessage(event); msg != "" {
		fields["upstream_error"] = msg
	}
	var payload map[string]any
	if json.Unmarshal(event.Data, &payload) == nil {
		if response, ok := payload["response"].(map[string]any); ok {
			for _, key := range []string{"id", "status", "stop_reason", "finish_reason"} {
				if value, ok := response[key].(string); ok {
					fields["upstream_"+key] = value
				}
			}
			value, present := response["end_turn"]
			endTurn, valid := value.(bool)
			fields["upstream_end_turn_present"] = present
			fields["upstream_end_turn_valid"] = valid
			if valid {
				fields["upstream_end_turn"] = endTurn
			}
			if output, ok := response["output"].([]any); ok {
				types := make([]string, 0, len(output))
				phases := make([]string, 0)
				for _, raw := range output {
					item, _ := raw.(map[string]any)
					typ, _ := item["type"].(string)
					types = append(types, typ)
					if phase, ok := item["phase"].(string); ok {
						phases = append(phases, phase)
					}
				}
				fields["upstream_output_item_types"] = types
				fields["upstream_message_phases"] = phases
			}
			if details, ok := response["incomplete_details"].(map[string]any); ok {
				if reason, ok := details["reason"].(string); ok {
					fields["upstream_incomplete_reason"] = reason
				}
			}
		}
		if endTurn, ok := payload["end_turn"].(bool); ok {
			fields["event_end_turn"] = endTurn
		}
		if item, ok := payload["item"].(map[string]any); ok {
			for _, key := range []string{"type", "phase", "status"} {
				if value, ok := item[key].(string); ok {
					fields["upstream_item_"+key] = value
				}
			}
		}
	}
	for _, event := range converted {
		if event.Event == "message_delta" {
			if delta, ok := event.Data["delta"].(map[string]any); ok {
				if reason, ok := delta["stop_reason"].(string); ok {
					fields["anthropic_stop_reason"] = reason
				}
			}
		}
	}
	s.trace("stream.event", mergeTraceFields(base, fields))
}

// instructionsTraceHash identifies guidance changes without logging its text.
func instructionsTraceHash(instructions string) string {
	hash := sha256.Sum256([]byte(strings.TrimSpace(instructions)))
	return hex.EncodeToString(hash[:8])
}
