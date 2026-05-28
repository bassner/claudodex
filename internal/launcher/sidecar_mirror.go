package launcher

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const (
	claudeConfigLockStaleAfter = time.Minute
	claudeConfigLockWait       = claudeConfigLockStaleAfter + 30*time.Second
)

var claudeConfigMirrorInterval = time.Second

var errClaudeConfigLockBusy = errors.New("Claude config lock busy")

var protectedGlobalConfigKeys = map[string]struct{}{
	"additionalModelOptionsCache":  {},
	"bridgeOauthDeadExpiresAt":     {},
	"bridgeOauthDeadFailCount":     {},
	"cachedDynamicConfigs":         {},
	"cachedGrowthBookFeatures":     {},
	"cachedStatsigGates":           {},
	"claudeCodeFirstTokenDate":     {},
	"clientDataCache":              {},
	"customApiKeyResponses":        {},
	"hasAvailableSubscription":     {},
	"metricsStatusCache":           {},
	"oauthAccount":                 {},
	"primaryApiKey":                {},
	"recommendedSubscription":      {},
	"subscriptionNoticeCount":      {},
	"subscriptionUpsellShownCount": {},
	"userID":                       {},
}

type ClaudeConfigMirror struct {
	cancel  context.CancelFunc
	done    chan struct{}
	sidecar string
	user    string

	mu       sync.Mutex
	baseline map[string]any
	lastErr  error
	models   modelconfig.Config
	started  time.Time
	cursor   time.Time
}

func StartClaudeConfigMirror(ctx context.Context, sidecarDir string, modelCfg modelconfig.Config) (*ClaudeConfigMirror, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	baseline := map[string]any{}
	if err := reconcileClaudeGlobalConfig(sidecarDir, userHome, &baseline, claudeConfigLockWait); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	mirror := &ClaudeConfigMirror{
		cancel:   cancel,
		done:     make(chan struct{}),
		sidecar:  sidecarDir,
		user:     userHome,
		baseline: baseline,
		models:   modelCfg.Normalize(),
		started:  time.Now().UTC(),
	}
	mirror.cursor = mirror.started
	go mirror.loop(ctx)
	return mirror, nil
}

func (m *ClaudeConfigMirror) Close() error {
	m.cancel()
	<-m.done

	syncErr := m.sync(claudeConfigLockWait)
	m.mu.Lock()
	defer m.mu.Unlock()
	return errors.Join(m.lastErr, syncErr)
}

func (m *ClaudeConfigMirror) loop(ctx context.Context) {
	defer close(m.done)
	ticker := time.NewTicker(claudeConfigMirrorInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.sync(0); err != nil && !errors.Is(err, errClaudeConfigLockBusy) {
				m.mu.Lock()
				m.lastErr = err
				m.mu.Unlock()
			}
		}
	}
}

func (m *ClaudeConfigMirror) sync(lockWait time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	nextCursor, transcriptErr := syncTranscriptModelDefault(m.sidecar, m.user, lockWait, m.models, m.cursor)
	if transcriptErr == nil && nextCursor.After(m.cursor) {
		m.cursor = nextCursor
	}
	return errors.Join(
		transcriptErr,
		normalizeSharedClaudeSettings(m.user, lockWait, m.models),
		reconcileClaudeGlobalConfig(m.sidecar, m.user, &m.baseline, lockWait),
	)
}

func syncTranscriptModelDefault(sidecarDir, userHome string, wait time.Duration, modelCfg modelconfig.Config, since time.Time) (time.Time, error) {
	model, timestamp, ok := latestTranscriptSavedDefaultModel(sidecarDir, since)
	if !ok {
		return since, nil
	}
	alias, ok := codexRuntimeSettingsAlias(model, modelCfg)
	if !ok {
		if family, familyOK := modelconfig.FamilyForModel(model); familyOK {
			alias = string(family)
			ok = true
		}
	}
	if !ok {
		return timestamp, nil
	}
	settingsPath := filepath.Join(userHome, ".claude", "settings.json")
	err := withClaudeConfigLocks([]string{settingsPath}, wait, func() error {
		settings, err := readJSONMap(settingsPath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			settings = map[string]any{}
		}
		if settings["model"] == alias {
			return nil
		}
		next := cloneJSONMap(settings)
		next["model"] = alias
		return writeJSONFile(settingsPath, next, 0o600)
	})
	if err != nil {
		return since, err
	}
	return timestamp, nil
}

type transcriptEntry struct {
	Timestamp string `json:"timestamp"`
	Message   struct {
		Content any `json:"content"`
	} `json:"message"`
}

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

func latestTranscriptSavedDefaultModel(sidecarDir string, since time.Time) (string, time.Time, bool) {
	projectsDir := filepath.Join(sidecarDir, "projects")
	var latestModel string
	var latestTime time.Time
	_ = filepath.WalkDir(projectsDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry == nil || entry.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		model, timestamp, ok := latestSavedDefaultModelInTranscript(path, since)
		if ok && (latestModel == "" || timestamp.After(latestTime)) {
			latestModel = model
			latestTime = timestamp
		}
		return nil
	})
	if latestModel == "" {
		return "", time.Time{}, false
	}
	return latestModel, latestTime, true
}

func latestSavedDefaultModelInTranscript(path string, since time.Time) (string, time.Time, bool) {
	file, err := os.Open(path)
	if err != nil {
		return "", time.Time{}, false
	}
	defer file.Close()

	var latestModel string
	var latestTime time.Time
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		var entry transcriptEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		timestamp, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
		if err != nil || !timestamp.After(since) {
			continue
		}
		content, ok := entry.Message.Content.(string)
		if !ok {
			continue
		}
		model, ok := savedDefaultModelFromCommandOutput(content)
		if ok && (latestModel == "" || timestamp.After(latestTime)) {
			latestModel = model
			latestTime = timestamp
		}
	}
	if latestModel == "" {
		return "", time.Time{}, false
	}
	return latestModel, latestTime, true
}

func savedDefaultModelFromCommandOutput(content string) (string, bool) {
	content = ansiEscapePattern.ReplaceAllString(content, "")
	const prefix = "Set model to "
	start := strings.Index(content, prefix)
	if start < 0 {
		return "", false
	}
	rest := content[start+len(prefix):]
	end := strings.Index(rest, " and saved as your default")
	if end < 0 {
		return "", false
	}
	model := strings.TrimSpace(rest[:end])
	model = strings.TrimSuffix(model, " (default)")
	model = strings.TrimSpace(model)
	if model == "" {
		return "", false
	}
	return model, true
}

func reconcileClaudeGlobalConfig(sidecarDir, userHome string, baseline *map[string]any, lockWait time.Duration) error {
	realPath := realClaudeGlobalConfigPath(userHome)
	sidecarPaths := sidecarGlobalConfigPaths(sidecarDir)
	paths := append([]string{realPath}, sidecarPaths...)

	return withClaudeConfigLocks(paths, lockWait, func() error {
		return reconcileClaudeGlobalConfigLocked(realPath, sidecarPaths, baseline, false)
	})
}

func initializeClaudeConfigSidecar(sidecarDir, userHome string) error {
	realPath := realClaudeGlobalConfigPath(userHome)
	sidecarPaths := sidecarGlobalConfigPaths(sidecarDir)
	paths := append([]string{realPath}, sidecarPaths...)
	baseline := map[string]any{}

	return withClaudeConfigLocks(paths, claudeConfigLockWait, func() error {
		return reconcileClaudeGlobalConfigLocked(realPath, sidecarPaths, &baseline, true)
	})
}

func reconcileClaudeGlobalConfigLocked(realPath string, sidecarPaths []string, baseline *map[string]any, ensureOnboarding bool) error {
	real, err := readConfigSnapshot(realPath)
	if err != nil {
		return err
	}
	sidecars := make([]configSnapshot, 0, len(sidecarPaths))
	for _, path := range sidecarPaths {
		snapshot, err := readConfigSnapshot(path)
		if err != nil {
			return err
		}
		sidecars = append(sidecars, snapshot)
	}
	active := mergeSidecarSnapshots(*baseline, sidecars)

	merged := mergeClaudeConfig(*baseline, active.sanitized, real.sanitized, active.mtime, real.mtime)
	if ensureOnboarding {
		if _, ok := merged["hasCompletedOnboarding"]; !ok {
			merged["hasCompletedOnboarding"] = true
		}
	}
	realNext := composeGlobalConfig(merged, real.raw)
	if !reflect.DeepEqual(real.raw, realNext) {
		if err := writeJSONFile(real.path, realNext, 0o600); err != nil {
			return err
		}
	}

	for _, sidecar := range sidecars {
		sidecarNext := composeGlobalConfig(merged, sidecar.raw)
		if !reflect.DeepEqual(sidecar.raw, sidecarNext) {
			if err := writeJSONFile(sidecar.path, sidecarNext, 0o600); err != nil {
				return err
			}
		}
	}
	*baseline = cloneJSONMap(merged)
	return nil
}

type configSnapshot struct {
	path      string
	raw       map[string]any
	sanitized map[string]any
	mtime     time.Time
	exists    bool
}

func readConfigSnapshot(path string) (configSnapshot, error) {
	config, err := readJSONMap(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return configSnapshot{
				path:      path,
				raw:       map[string]any{},
				sanitized: map[string]any{},
			}, nil
		}
		return configSnapshot{}, err
	}
	info, err := os.Stat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return configSnapshot{}, err
	}
	var mtime time.Time
	if info != nil {
		mtime = info.ModTime()
	}
	return configSnapshot{
		path:      path,
		raw:       config,
		sanitized: sanitizeGlobalConfig(config),
		mtime:     mtime,
		exists:    true,
	}, nil
}

func mergeSidecarSnapshots(base map[string]any, snapshots []configSnapshot) configSnapshot {
	if len(snapshots) == 0 {
		return configSnapshot{raw: map[string]any{}, sanitized: cloneJSONMap(base)}
	}
	ordered := append([]configSnapshot(nil), snapshots...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].mtime.Before(ordered[j].mtime)
	})

	merged := cloneJSONMap(base)
	var mergedTime time.Time
	for _, snapshot := range ordered {
		if !snapshot.exists {
			continue
		}
		merged = mergeJSONMap(base, snapshot.sanitized, merged, snapshot.mtime, mergedTime)
		if snapshot.mtime.After(mergedTime) {
			mergedTime = snapshot.mtime
		}
	}
	return configSnapshot{
		raw:       map[string]any{},
		sanitized: merged,
		mtime:     mergedTime,
	}
}

func realClaudeGlobalConfigPath(userHome string) string {
	return filepath.Join(userHome, claudeGlobalConfigName)
}

func sidecarGlobalConfigPaths(sidecarDir string) []string {
	return []string{
		filepath.Join(sidecarDir, claudeGlobalConfigName),
		filepath.Join(sidecarDir, claudeLocalOAuthConfigName),
	}
}

func sanitizeGlobalConfig(config map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range config {
		if isProtectedGlobalConfigKey(key) {
			continue
		}
		out[key] = cloneJSONValue(value)
	}
	return out
}

func composeGlobalConfig(sanitized map[string]any, current map[string]any) map[string]any {
	out := cloneJSONMap(sanitized)
	for key, value := range current {
		if isProtectedGlobalConfigKey(key) {
			out[key] = cloneJSONValue(value)
		}
	}
	return out
}

func isProtectedGlobalConfigKey(key string) bool {
	_, ok := protectedGlobalConfigKeys[key]
	return ok
}

func mergeClaudeConfig(base, sidecar, real map[string]any, sidecarTime, realTime time.Time) map[string]any {
	return mergeJSONMap(base, sidecar, real, sidecarTime, realTime)
}

func mergeJSONMap(base, sidecar, real map[string]any, sidecarTime, realTime time.Time) map[string]any {
	out := map[string]any{}
	for _, key := range unionKeys(base, sidecar, real) {
		baseValue, baseOK := base[key]
		sidecarValue, sidecarOK := sidecar[key]
		realValue, realOK := real[key]
		if value, keep := mergeJSONField(baseValue, baseOK, sidecarValue, sidecarOK, realValue, realOK, sidecarTime, realTime); keep {
			out[key] = value
		}
	}
	return out
}

func mergeJSONField(baseValue any, baseOK bool, sidecarValue any, sidecarOK bool, realValue any, realOK bool, sidecarTime, realTime time.Time) (any, bool) {
	sidecarChanged := !sameJSONPresence(baseValue, baseOK, sidecarValue, sidecarOK)
	realChanged := !sameJSONPresence(baseValue, baseOK, realValue, realOK)

	switch {
	case !sidecarChanged && !realChanged:
		if realOK {
			return cloneJSONValue(realValue), true
		}
		if sidecarOK {
			return cloneJSONValue(sidecarValue), true
		}
		return nil, false
	case sidecarChanged && !realChanged:
		if !sidecarOK {
			return nil, false
		}
		return cloneJSONValue(sidecarValue), true
	case !sidecarChanged && realChanged:
		if !realOK {
			return nil, false
		}
		return cloneJSONValue(realValue), true
	}

	baseMap, _ := asJSONMap(baseValue)
	sidecarMap, sidecarMapOK := asJSONMap(sidecarValue)
	realMap, realMapOK := asJSONMap(realValue)
	if sidecarMapOK && realMapOK {
		return mergeJSONMap(baseMap, sidecarMap, realMap, sidecarTime, realTime), true
	}

	if sidecarTime.After(realTime) || sidecarTime.Equal(realTime) {
		if !sidecarOK {
			return nil, false
		}
		return cloneJSONValue(sidecarValue), true
	}
	if !realOK {
		return nil, false
	}
	return cloneJSONValue(realValue), true
}

func sameJSONPresence(left any, leftOK bool, right any, rightOK bool) bool {
	if leftOK != rightOK {
		return false
	}
	if !leftOK {
		return true
	}
	return reflect.DeepEqual(left, right)
}

func asJSONMap(value any) (map[string]any, bool) {
	if value == nil {
		return map[string]any{}, false
	}
	typed, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}, false
	}
	return typed, true
}

func unionKeys(maps ...map[string]any) []string {
	seen := map[string]struct{}{}
	for _, item := range maps {
		for key := range item {
			seen[key] = struct{}{}
		}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func cloneJSONMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = cloneJSONValue(value)
	}
	return out
}

func cloneJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneJSONMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = cloneJSONValue(item)
		}
		return out
	default:
		return typed
	}
}

func withClaudeConfigLocks(paths []string, wait time.Duration, fn func() error) error {
	unique := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, path := range paths {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		unique = append(unique, path)
	}
	sort.Strings(unique)

	deadline := time.Now().Add(wait)
	for {
		releases, err := tryAcquireClaudeConfigLocks(unique)
		if err == nil {
			defer releaseClaudeConfigLocks(releases)
			return fn()
		}
		if !errors.Is(err, errClaudeConfigLockBusy) {
			return err
		}
		if wait == 0 || time.Now().After(deadline) {
			return err
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func tryAcquireClaudeConfigLocks(paths []string) ([]func() error, error) {
	releases := make([]func() error, 0, len(paths))
	for _, path := range paths {
		release, err := tryAcquireClaudeConfigLock(path)
		if err != nil {
			releaseClaudeConfigLocks(releases)
			return nil, err
		}
		releases = append(releases, release)
	}
	return releases, nil
}

func tryAcquireClaudeConfigLock(path string) (func() error, error) {
	lockPath := path + ".lock"
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}

	err := os.Mkdir(lockPath, 0o700)
	if err == nil {
		return func() error { return os.Remove(lockPath) }, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("acquire Claude config lock %s: %w", lockPath, err)
	}

	if removeStaleClaudeConfigLock(lockPath) {
		err = os.Mkdir(lockPath, 0o700)
		if err == nil {
			return func() error { return os.Remove(lockPath) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("acquire Claude config lock %s: %w", lockPath, err)
		}
	}
	return nil, errClaudeConfigLockBusy
}

func removeStaleClaudeConfigLock(lockPath string) bool {
	info, err := os.Stat(lockPath)
	if err != nil {
		return false
	}
	if time.Since(info.ModTime()) <= claudeConfigLockStaleAfter {
		return false
	}
	_ = os.RemoveAll(lockPath)
	return true
}

func releaseClaudeConfigLocks(releases []func() error) {
	for i := len(releases) - 1; i >= 0; i-- {
		_ = releases[i]()
	}
}

func withClaudeSidecarSetupLock(sidecarDir string, wait time.Duration, fn func() error) error {
	lockPath := filepath.Join(sidecarDir, ".setup")
	return withClaudeConfigLocks([]string{lockPath}, wait, fn)
}
