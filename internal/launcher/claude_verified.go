package launcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bassner/claudodex/internal/auth"
	"github.com/bassner/claudodex/internal/filelock"
	"github.com/bassner/claudodex/internal/modelconfig"
	"golang.org/x/mod/semver"
)

// ClaudeBuild is an exact original binary identity supported by this release.
type ClaudeBuild struct {
	Version string `json:"version"`
	GOOS    string `json:"goos"`
	GOARCH  string `json:"goarch"`
	SHA256  string `json:"sha256"`
}

func SupportedClaudeBuilds() []ClaudeBuild {
	builds := make([]ClaudeBuild, 0, len(claudeUIPatches))
	for _, p := range claudeUIPatches {
		builds = append(builds, ClaudeBuild{p.Version, p.GOOS, p.GOARCH, p.SHA256})
	}
	return builds
}

type claudeSource struct {
	Path    string
	Version string
	SHA256  string
	Data    []byte
}

type claudeSelection struct {
	SourcePath string
	Executable string
	Version    string
	Release    func()
}

type claudeSourceRecord struct {
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

type claudePatchRecord struct {
	Schema           string             `json:"schema"`
	ClaudeVersion    string             `json:"claude_version"`
	SourceSHA256     string             `json:"source_sha256"`
	TransformedSHA   string             `json:"transformed_sha256"`
	SignedSHA256     string             `json:"signed_sha256"`
	ClaudodexVersion string             `json:"claudodex_version"`
	Models           modelconfig.Config `json:"models"`
}

type claudeLease struct {
	PID          int    `json:"pid"`
	ProcessStart string `json:"process_start"`
	SourceSHA256 string `json:"source_sha256"`
	PatchPath    string `json:"patch_path"`
}

func claudeCacheRoot(home string) (string, error) {
	dir, err := auth.DataDir(home)
	if err != nil {
		return "", err
	}
	root := filepath.Join(dir, claudodexPatchedClaudeDirName)
	if err := ensurePrivateDirectory(root); err != nil {
		return "", err
	}
	return root, nil
}

// The caller resolves the installed symlink once, before any auth or settings
// changes. All patch checks and transformations consume this one source read.
func readClaudeSource(ctx context.Context, path string) (claudeSource, error) {
	source := claudeSource{Path: path}
	info, err := os.Lstat(path)
	if err != nil {
		return source, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return source, fmt.Errorf("Claude source is not a regular executable")
	}
	source.Data, err = os.ReadFile(path)
	if err != nil {
		return source, err
	}
	source.SHA256 = sha256Hex(source.Data)
	if base := filepath.Base(path); looksLikeVersion(base) {
		source.Version = base
	} else {
		// A registered hash identifies version without executing an unknown file.
		for _, patch := range claudeUIPatches {
			if patch.GOOS == runtime.GOOS && patch.GOARCH == runtime.GOARCH && strings.EqualFold(patch.SHA256, source.SHA256) {
				source.Version = patch.Version
				break
			}
		}
		if source.Version == "" {
			source.Version = detectClaudeVersion(ctx, path)
		}
	}
	return source, nil
}

func selectClaudeExecutable(ctx context.Context, home, installedPath, claudodexVersion string, models modelconfig.Config, stderr io.Writer) (claudeSelection, error) {
	if strings.TrimSpace(os.Getenv("CLAUDODEX_DISABLE_CLAUDE_PATCH")) == "1" {
		return claudeSelection{SourcePath: installedPath, Executable: installedPath, Release: func() {}}, nil
	}
	root, err := claudeCacheRoot(home)
	if err != nil {
		return claudeSelection{}, err
	}
	unlock, err := filelock.Acquire(ctx, filepath.Join(root, ".lock"))
	if err != nil {
		return claudeSelection{}, err
	}
	defer unlock()
	installed, installedErr := readClaudeSource(ctx, installedPath)
	try := func(source claudeSource) (claudeSelection, error) {
		patched, err := preparePatchedClaudeSource(ctx, root, source, claudodexVersion, models.Normalize())
		if err != nil {
			return claudeSelection{}, err
		}
		snapshot, err := cacheClaudeSource(root, source)
		if err != nil {
			return claudeSelection{}, err
		}
		leasePath, err := createClaudeLease(ctx, root, source.SHA256, patched)
		if err != nil {
			return claudeSelection{}, err
		}
		retainClaudeSources(ctx, root, claudodexVersion, models)
		return claudeSelection{SourcePath: snapshot, Executable: patched, Version: source.Version, Release: func() {
			// Removal is safe without the cache lock: retention either sees this
			// lease, or sees that this launch has already stopped using its files.
			_ = os.Remove(leasePath)
		}}, nil
	}
	if installedErr == nil {
		selected, err := try(installed)
		if err == nil {
			return selected, nil
		}
		installedErr = err
	}
	for _, path := range claudeFallbackCandidates(home, installedPath, root) {
		if err := ctx.Err(); err != nil {
			return claudeSelection{}, err
		}
		source, err := readClaudeSource(ctx, path)
		if err != nil {
			continue
		}
		selected, err := try(source)
		if err != nil {
			continue
		}
		if stderr != nil {
			fmt.Fprintf(stderr, "Claudodex: cannot use installed Claude Code %s; using verified local Claude Code %s.\n", displayClaudeVersion(installed.Version), selected.Version)
		}
		return selected, nil
	}
	builds := SupportedClaudeBuilds()
	if len(builds) == 0 {
		return claudeSelection{}, fmt.Errorf("this Claudodex build has no verified Claude Code targets; update Claudodex or explicitly launch without compatibility patches with CLAUDODEX_DISABLE_CLAUDE_PATCH=1")
	}
	target := builds[0]
	supportedPlatform := false
	for _, build := range builds {
		if build.GOOS == runtime.GOOS && build.GOARCH == runtime.GOARCH {
			target = build
			supportedPlatform = true
			break
		}
	}
	if !supportedPlatform {
		return claudeSelection{}, fmt.Errorf("Claude Code %s has no usable verified patch for %s/%s, and this release has no verified builds on this platform. A supported target is Claude Code %s on %s/%s (`claude install %s` on that platform). Update Claudodex, or explicitly launch without compatibility patches with CLAUDODEX_DISABLE_CLAUDE_PATCH=1", displayClaudeVersion(installed.Version), runtime.GOOS, runtime.GOARCH, target.Version, target.GOOS, target.GOARCH, target.Version)
	}
	return claudeSelection{}, fmt.Errorf("Claude Code %s has no usable verified patch for %s/%s (%v), and no verified local fallback is available. Update Claudodex or install supported Claude Code %s with `claude install %s`. To explicitly launch without compatibility patches, set CLAUDODEX_DISABLE_CLAUDE_PATCH=1", displayClaudeVersion(installed.Version), runtime.GOOS, runtime.GOARCH, installedErr, target.Version, target.Version)
}

func displayClaudeVersion(version string) string {
	if version == "" {
		return "unknown"
	}
	return version
}

func preparePatchedClaudeSource(ctx context.Context, root string, source claudeSource, claudodexVersion string, models modelconfig.Config) (string, error) {
	patch := findClaudeUIPatch(source.Version, source.SHA256)
	if patch == nil {
		return "", errClaudePatchUnsupported
	}
	transformed := append([]byte(nil), source.Data...)
	if !patch.Apply(transformed, claudodexVersion, source.Version, models) {
		return "", errClaudePatchNoMatch
	}
	record := claudePatchRecord{
		Schema: claudodexPatchSchemaVersion, ClaudeVersion: source.Version, SourceSHA256: source.SHA256,
		TransformedSHA: sha256Hex(transformed), ClaudodexVersion: claudodexVersion, Models: models,
	}
	dir := filepath.Join(root, patchedClaudeCacheKey(source.Data, claudodexVersion, source.Version, models))
	dest := filepath.Join(dir, "claude")
	if validClaudePatchCache(dir, record) {
		return dest, nil
	}
	if err := ensurePrivateDirectory(dir); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(dir, ".claude-*")
	if err != nil {
		return "", err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(transformed); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Chmod(0o700); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := signPatchedClaude(ctx, name); err != nil {
		return "", err
	}
	signedSHA, err := privateFileSHA256(name, true)
	if err != nil {
		return "", err
	}
	record.SignedSHA256 = signedSHA
	if err := syncFile(name); err != nil {
		return "", err
	}
	if err := os.Rename(name, dest); err != nil {
		return "", err
	}
	if err := atomicPrivateJSON(filepath.Join(dir, "record.json"), record); err != nil {
		return "", err
	}
	return dest, syncDirectory(dir)
}

var signPatchedClaude = func(ctx context.Context, path string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	signCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	output, err := exec.CommandContext(signCtx, "codesign", "--force", "--sign", "-", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("codesign patched Claude: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func validClaudePatchCache(dir string, expected claudePatchRecord) bool {
	if !isPrivateDirectory(dir) {
		return false
	}
	var recorded claudePatchRecord
	if readPrivateJSON(filepath.Join(dir, "record.json"), &recorded) != nil || recorded.SignedSHA256 == "" {
		return false
	}
	expected.SignedSHA256 = recorded.SignedSHA256
	if recorded != expected {
		return false
	}
	digest, err := privateFileSHA256(filepath.Join(dir, "claude"), true)
	return err == nil && digest == recorded.SignedSHA256
}

func cacheClaudeSource(root string, source claudeSource) (string, error) {
	dir := filepath.Join(root, "sources", source.Version+"-"+source.SHA256)
	if err := ensurePrivateDirectory(filepath.Dir(dir)); err != nil {
		return "", err
	}
	if err := ensurePrivateDirectory(dir); err != nil {
		return "", err
	}
	path := filepath.Join(dir, source.Version)
	digest, err := privateFileSHA256(path, true)
	if err != nil || digest != source.SHA256 {
		if err := atomicPrivateFile(path, source.Data, 0o700); err != nil {
			return "", err
		}
	}
	if err := atomicPrivateJSON(filepath.Join(dir, "source.json"), claudeSourceRecord{source.Version, source.SHA256}); err != nil {
		return "", err
	}
	return path, nil
}

func claudeFallbackCandidates(home, installedPath, root string) []string {
	var paths []string
	sourceDirs, _ := os.ReadDir(filepath.Join(root, "sources"))
	for _, entry := range sourceDirs {
		dir := filepath.Join(root, "sources", entry.Name())
		var record claudeSourceRecord
		if !isPrivateDirectory(dir) || readPrivateJSON(filepath.Join(dir, "source.json"), &record) != nil || !looksLikeVersion(record.Version) || findClaudeUIPatch(record.Version, record.SHA256) == nil {
			continue
		}
		path := filepath.Join(dir, record.Version)
		digest, err := privateFileSHA256(path, true)
		if err == nil && digest == record.SHA256 {
			paths = append(paths, path)
		}
	}
	versionDirs := []string{}
	if home != "" {
		versionDirs = append(versionDirs, filepath.Join(home, ".local", "share", "claude", "versions"))
	}
	if filepath.Base(filepath.Dir(installedPath)) == "versions" {
		versionDirs = append(versionDirs, filepath.Dir(installedPath))
	}
	for _, dir := range versionDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())
			if looksLikeVersion(entry.Name()) && path != installedPath && entry.Type().IsRegular() {
				paths = append(paths, path)
			}
		}
	}
	sort.SliceStable(paths, func(i, j int) bool {
		return semver.Compare("v"+filepath.Base(paths[i]), "v"+filepath.Base(paths[j])) > 0
	})
	return paths
}

func createClaudeLease(ctx context.Context, root, sourceSHA, patched string) (string, error) {
	dir := filepath.Join(root, "leases")
	if err := ensurePrivateDirectory(dir); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(dir, strconv.Itoa(os.Getpid())+"-*.json")
	if err != nil {
		return "", err
	}
	path := f.Name()
	_ = f.Close()
	start, _ := claudeProcessStart(ctx, os.Getpid())
	if err := atomicPrivateJSON(path, claudeLease{os.Getpid(), start, sourceSHA, patched}); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func claudeProcessStart(ctx context.Context, pid int) (string, error) {
	psCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(psCtx, "ps", "-p", strconv.Itoa(pid), "-o", "lstart=")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	output, err := cmd.Output()
	start := strings.TrimSpace(string(output))
	if err != nil || start == "" {
		return "", fmt.Errorf("cannot identify process start time")
	}
	return start, nil
}

// Only recorded files are candidates for deletion. Any unknown lease state
// disables retention, so a permission or process-inspection failure is harmless.
func retainClaudeSources(ctx context.Context, root, claudodexVersion string, models modelconfig.Config) {
	models = models.Normalize()
	keepSource := map[string]bool{}
	keepPatch := map[string]bool{}
	leases, err := os.ReadDir(filepath.Join(root, "leases"))
	if err != nil {
		return
	}
	for _, entry := range leases {
		path := filepath.Join(root, "leases", entry.Name())
		var lease claudeLease
		if readPrivateJSON(path, &lease) != nil || lease.PID <= 0 {
			return
		}
		process, err := os.FindProcess(lease.PID)
		if err != nil {
			return
		}
		err = process.Signal(syscall.Signal(0))
		if errors.Is(err, syscall.ESRCH) || errors.Is(err, os.ErrProcessDone) {
			_ = os.Remove(path)
			continue
		}
		if err != nil {
			return
		}
		if lease.ProcessStart == "" || lease.SourceSHA256 == "" || lease.PatchPath == "" {
			// A known dead PID was handled above, even if ps failed when the
			// lease was written. An uncertain live identity still stops deletion.
			return
		}
		start, err := claudeProcessStart(ctx, lease.PID)
		if err != nil {
			return
		}
		if start != lease.ProcessStart {
			_ = os.Remove(path)
			continue
		}
		keepSource[lease.SourceSHA256] = true
		keepPatch[filepath.Dir(lease.PatchPath)] = true
	}
	entries, err := os.ReadDir(filepath.Join(root, "sources"))
	if err != nil {
		return
	}
	type snapshot struct {
		dir    string
		record claudeSourceRecord
	}
	var snapshots []snapshot
	for _, entry := range entries {
		dir := filepath.Join(root, "sources", entry.Name())
		var record claudeSourceRecord
		if isPrivateDirectory(dir) && readPrivateJSON(filepath.Join(dir, "source.json"), &record) == nil && looksLikeVersion(record.Version) && findClaudeUIPatch(record.Version, record.SHA256) != nil {
			digest, err := privateFileSHA256(filepath.Join(dir, record.Version), true)
			if err == nil && digest == record.SHA256 {
				snapshots = append(snapshots, snapshot{dir, record})
			}
		}
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return semver.Compare("v"+snapshots[i].record.Version, "v"+snapshots[j].record.Version) > 0
	})
	for i, snapshot := range snapshots {
		if i < 2 {
			keepSource[snapshot.record.SHA256] = true
		}
	}
	for _, snapshot := range snapshots {
		if !keepSource[snapshot.record.SHA256] {
			_ = os.RemoveAll(snapshot.dir)
		}
	}
	patches, _ := os.ReadDir(root)
	for _, entry := range patches {
		dir := filepath.Join(root, entry.Name())
		var record claudePatchRecord
		if !keepPatch[dir] && isPrivateDirectory(dir) && readPrivateJSON(filepath.Join(dir, "record.json"), &record) == nil && record.Schema != "" && record.SourceSHA256 != "" && record.ClaudodexVersion != "" {
			obsolete := !keepSource[record.SourceSHA256] || record.Schema != claudodexPatchSchemaVersion || record.ClaudodexVersion != claudodexVersion || record.Models != models
			if obsolete {
				_ = os.RemoveAll(dir)
			}
		}
	}
}

func ensurePrivateDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	if !isPrivateDirectory(path) {
		return fmt.Errorf("cache directory must be a private regular directory: %s", path)
	}
	return nil
}

func isPrivateDirectory(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir() && info.Mode().Perm()&0o077 == 0
}

func readPrivateFile(path string, executable bool) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || (executable && info.Mode().Perm()&0o100 == 0) {
		return nil, fmt.Errorf("unsafe private cache file")
	}
	return os.ReadFile(path)
}

// Hash-only checks must not retain another full Claude-sized allocation. The
// original and its fresh transformation remain the two necessary byte buffers.
func privateFileSHA256(path string, executable bool) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || (executable && info.Mode().Perm()&0o100 == 0) {
		return "", fmt.Errorf("unsafe private cache file")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func readPrivateJSON(path string, value any) error {
	data, err := readPrivateFile(path, false)
	if err != nil {
		return err
	}
	if len(data) > 64*1024 {
		return fmt.Errorf("cache record is too large")
	}
	return json.Unmarshal(data, value)
}

func atomicPrivateJSON(path string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return atomicPrivateFile(path, data, 0o600)
}

func atomicPrivateFile(path string, data []byte, mode os.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".write-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Chmod(mode); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func syncFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

func syncDirectory(path string) error { return syncFile(path) }
