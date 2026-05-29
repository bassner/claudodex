package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
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

	home := strings.TrimSpace(s.cfg.Home)
	if home == "" {
		if userHome, err := os.UserHomeDir(); err == nil && userHome != "" {
			home = filepath.Join(userHome, ".claudodex")
		}
	}
	if home == "" {
		return ""
	}
	return filepath.Join(home, "logs", "proxy-trace-"+time.Now().Format("2006-01-02")+".jsonl")
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
	case "response.created", "response.completed", "response.done", "response.failed", "error":
		return true
	default:
		return strings.HasSuffix(event, ".done")
	}
}
