package launcher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/bassner/claudodex/internal/auth"
	"github.com/bassner/claudodex/internal/modelconfig"
)

const (
	claudodexPatchedClaudeDirName = "patched-claude"
	claudodexPatchSchemaVersion   = "claude-ui-patch-v23"
)

var (
	errClaudePatchUnsupported = errors.New("unsupported Claude Code version for UI patch")
	errClaudePatchNoMatch     = errors.New("Claude Code UI patch did not match binary")
)

type claudeUIPatchFunc func(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool

type claudeUIPatchSpec struct {
	Version string
	GOOS    string
	GOARCH  string
	SHA256  string
	Apply   claudeUIPatchFunc
}

var claudeUIPatches = []claudeUIPatchSpec{
	claudeUIPatch_2_1_156,
	claudeUIPatch_2_1_154,
	claudeUIPatch_2_1_153,
}

func prepareClaudeExecutable(ctx context.Context, home, claudePath, claudodexVersion string, modelCfg modelconfig.Config, stderr io.Writer) string {
	if strings.TrimSpace(os.Getenv("CLAUDODEX_DISABLE_CLAUDE_PATCH")) == "1" {
		return claudePath
	}
	patched, claudeVersion, sourceSHA, err := preparePatchedClaude(ctx, home, claudePath, claudodexVersion, modelCfg)
	if err != nil {
		warnClaudePatchSkipped(stderr, claudeVersion, sourceSHA, err)
		return claudePath
	}
	return patched
}

func preparePatchedClaude(ctx context.Context, home, claudePath, claudodexVersion string, modelCfg modelconfig.Config) (string, string, string, error) {
	modelCfg = modelCfg.Normalize()
	claudeVersion := detectClaudeVersion(ctx, claudePath)
	sourceData, err := os.ReadFile(claudePath)
	if err != nil {
		return "", claudeVersion, "", err
	}
	sourceSHA := sha256Hex(sourceData)
	patcher := findClaudeUIPatch(claudeVersion, sourceSHA)
	if patcher == nil {
		return "", claudeVersion, sourceSHA, errClaudePatchUnsupported
	}
	patched := append([]byte(nil), sourceData...)
	changed := patcher.Apply(patched, claudodexVersion, claudeVersion, modelCfg)
	if !changed {
		return "", claudeVersion, sourceSHA, errClaudePatchNoMatch
	}

	dataDir, err := auth.DataDir(home)
	if err != nil {
		return "", claudeVersion, sourceSHA, err
	}
	key := patchedClaudeCacheKey(sourceData, claudodexVersion, claudeVersion, modelCfg)
	dir := filepath.Join(dataDir, claudodexPatchedClaudeDirName, key)
	dest := filepath.Join(dir, "claude")
	if isExecutableFile(dest) {
		return dest, claudeVersion, sourceSHA, nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", claudeVersion, sourceSHA, err
	}

	mode := os.FileMode(0o755)
	if info, err := os.Stat(claudePath); err == nil {
		mode = info.Mode() | 0o700
	}
	tmp, err := os.CreateTemp(dir, ".claude-*.tmp")
	if err != nil {
		return "", claudeVersion, sourceSHA, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(patched); err != nil {
		_ = tmp.Close()
		return "", claudeVersion, sourceSHA, err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return "", claudeVersion, sourceSHA, err
	}
	if err := tmp.Close(); err != nil {
		return "", claudeVersion, sourceSHA, err
	}
	if runtime.GOOS == "darwin" {
		signCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		cmd := exec.CommandContext(signCtx, "codesign", "--force", "--sign", "-", tmpName)
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", claudeVersion, sourceSHA, fmt.Errorf("codesign patched Claude: %w: %s", err, strings.TrimSpace(string(output)))
		}
	}
	if err := os.Rename(tmpName, dest); err != nil {
		return "", claudeVersion, sourceSHA, err
	}
	return dest, claudeVersion, sourceSHA, nil
}

func findClaudeUIPatch(version, sourceSHA string) *claudeUIPatchSpec {
	for i := range claudeUIPatches {
		patch := &claudeUIPatches[i]
		if patch.Version == version &&
			patch.GOOS == runtime.GOOS &&
			patch.GOARCH == runtime.GOARCH &&
			strings.EqualFold(patch.SHA256, sourceSHA) {
			return patch
		}
	}
	return nil
}

func warnClaudePatchSkipped(stderr io.Writer, claudeVersion, sourceSHA string, err error) {
	if stderr == nil {
		return
	}
	version := strings.TrimSpace(claudeVersion)
	if version == "" {
		version = "unknown"
	}
	fingerprint := "unknown"
	if sourceSHA != "" {
		fingerprint = sourceSHA
		if len(fingerprint) > 12 {
			fingerprint = fingerprint[:12]
		}
	}
	target := fmt.Sprintf("Claude Code %s for %s/%s sha256:%s", version, runtime.GOOS, runtime.GOARCH, fingerprint)
	switch {
	case errors.Is(err, errClaudePatchUnsupported):
		fmt.Fprintf(stderr, "warning: Claudodex has no verified UI patch for %s; launching with the unpatched Claude Code UI. Update Claudodex, or open an issue if none exists for this Claude Code patch target: https://github.com/bassner/claudodex/issues\n", target)
	case errors.Is(err, errClaudePatchNoMatch):
		fmt.Fprintf(stderr, "warning: Claudodex UI patch for %s did not match this binary; launching with the unpatched Claude Code UI. Update Claudodex, or open an issue if none exists for this Claude Code patch target: https://github.com/bassner/claudodex/issues\n", target)
	default:
		fmt.Fprintf(stderr, "warning: Claudodex could not prepare the UI patch for %s (%v); launching with the unpatched Claude Code UI. Update Claudodex, or open an issue if this persists: https://github.com/bassner/claudodex/issues\n", target, err)
	}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func patchedClaudeCacheKey(sourceData []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) string {
	h := sha256.New()
	h.Write([]byte(claudodexPatchSchemaVersion))
	h.Write([]byte{0})
	h.Write(sourceData)
	h.Write([]byte{0})
	h.Write([]byte(claudodexVersion))
	h.Write([]byte{0})
	h.Write([]byte(claudeVersion))
	h.Write([]byte{0})
	h.Write([]byte(modelCfg.Opus))
	h.Write([]byte{0})
	h.Write([]byte(modelCfg.Sonnet))
	h.Write([]byte{0})
	h.Write([]byte(modelCfg.Haiku))
	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:12])
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

func detectClaudeVersion(ctx context.Context, claudePath string) string {
	if resolved, err := filepath.EvalSymlinks(claudePath); err == nil {
		base := filepath.Base(resolved)
		if looksLikeVersion(base) {
			return base
		}
	}
	versionCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	output, err := exec.CommandContext(versionCtx, claudePath, "--version").Output()
	if err != nil {
		return "unknown"
	}
	fields := strings.Fields(string(output))
	if len(fields) == 0 || !looksLikeVersion(fields[0]) {
		return "unknown"
	}
	return fields[0]
}

func looksLikeVersion(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

func claudodexInfoLine() string {
	return "Bugs:\\ngithub.com/bassner/claudodex/issues"
}

func quotedVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		version = "dev"
	}
	return `"v` + version + `"`
}

func claudodexLogoVersion(claudodexVersion, claudeVersion string) string {
	claudodexVersion = strings.TrimSpace(claudodexVersion)
	if claudodexVersion == "" {
		claudodexVersion = "dev"
	}
	claudeVersion = strings.TrimSpace(claudeVersion)
	if claudeVersion == "" {
		claudeVersion = "unknown"
	}
	if looksLikeVersion(claudeVersion) {
		claudeVersion = "v" + claudeVersion
	}
	return claudodexVersion + " using Claude Code " + claudeVersion
}

func quoteJSString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func modelDescriptionPatch(model, suffix string) string {
	model = modelconfig.StripLongContext(strings.TrimSpace(model))
	if model == "" {
		return suffix
	}
	return model + " " + suffix
}

func replaceAllFixed(data []byte, old, replacement string) bool {
	oldBytes := []byte(old)
	newBytes, ok := fitReplacement(oldBytes, replacement)
	if !ok {
		return false
	}
	changed := false
	for {
		index := bytes.Index(data, oldBytes)
		if index < 0 {
			return changed
		}
		copy(data[index:index+len(oldBytes)], newBytes)
		changed = true
	}
}

func replaceAllPatternString(data []byte, pattern, old, replacement string) bool {
	patternBytes := []byte(pattern)
	oldBytes := []byte(old)
	newBytes, ok := fitReplacement(oldBytes, replacement)
	if !ok {
		return false
	}
	changed := false
	searchFrom := 0
	for {
		index := bytes.Index(data[searchFrom:], patternBytes)
		if index < 0 {
			return changed
		}
		absolute := searchFrom + index
		inner := bytes.Index(data[absolute:absolute+len(patternBytes)], oldBytes)
		if inner >= 0 {
			copy(data[absolute+inner:absolute+inner+len(oldBytes)], newBytes)
			changed = true
		}
		searchFrom = absolute + len(patternBytes)
	}
}

func replaceFirstFixed(data []byte, old, replacement string) bool {
	oldBytes := []byte(old)
	newBytes, ok := fitReplacement(oldBytes, replacement)
	if !ok {
		return false
	}
	index := bytes.Index(data, oldBytes)
	if index < 0 {
		return false
	}
	copy(data[index:index+len(oldBytes)], newBytes)
	return true
}

func fitReplacement(old []byte, replacement string) ([]byte, bool) {
	newBytes := []byte(replacement)
	if len(newBytes) > len(old) {
		newBytes = newBytes[:len(old)]
	}
	if len(newBytes) < len(old) {
		padded := make([]byte, len(old))
		copy(padded, newBytes)
		for i := len(newBytes); i < len(padded); i++ {
			padded[i] = ' '
		}
		newBytes = padded
	}
	return newBytes, true
}
