package launcher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
	claudodexPatchSchemaVersion   = "claude-ui-patch-v6"
)

func prepareClaudeExecutable(ctx context.Context, home, claudePath, claudodexVersion string, modelCfg modelconfig.Config) string {
	if strings.TrimSpace(os.Getenv("CLAUDODEX_DISABLE_CLAUDE_PATCH")) == "1" {
		return claudePath
	}
	patched, err := preparePatchedClaude(ctx, home, claudePath, claudodexVersion, modelCfg)
	if err != nil {
		return claudePath
	}
	return patched
}

func preparePatchedClaude(ctx context.Context, home, claudePath, claudodexVersion string, modelCfg modelconfig.Config) (string, error) {
	modelCfg = modelCfg.Normalize()
	sourceData, err := os.ReadFile(claudePath)
	if err != nil {
		return "", err
	}
	claudeVersion := detectClaudeVersion(ctx, claudePath)
	patched := append([]byte(nil), sourceData...)
	changed := applyClaudeUIPatches(patched, claudodexVersion, claudeVersion, modelCfg)
	if !changed {
		return claudePath, nil
	}

	dataDir, err := auth.DataDir(home)
	if err != nil {
		return "", err
	}
	key := patchedClaudeCacheKey(sourceData, claudodexVersion, claudeVersion, modelCfg)
	dir := filepath.Join(dataDir, claudodexPatchedClaudeDirName, key)
	dest := filepath.Join(dir, "claude")
	if isExecutableFile(dest) {
		return dest, nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}

	mode := os.FileMode(0o755)
	if info, err := os.Stat(claudePath); err == nil {
		mode = info.Mode() | 0o700
	}
	tmp, err := os.CreateTemp(dir, ".claude-*.tmp")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(patched); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if runtime.GOOS == "darwin" {
		signCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		cmd := exec.CommandContext(signCtx, "codesign", "--force", "--sign", "-", tmpName)
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("codesign patched Claude: %w: %s", err, strings.TrimSpace(string(output)))
		}
	}
	if err := os.Rename(tmpName, dest); err != nil {
		return "", err
	}
	return dest, nil
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

func applyClaudeUIPatches(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	changed := false
	changed = replaceAllFixed(data, "Check the Claude Code changelog for updates", claudodexInfoLine()) || changed
	changed = replaceAllFixed(data, "What's new", "Info") || changed
	changed = replaceAllFixed(data, "Welcome back!", "Welcome back") || changed
	changed = replaceAllFixed(data, "Claude Max", "Codex Plan") || changed
	changed = replaceAllFixed(data, "Switch between Claude models. Your pick becomes the default for new sessions. For other/previous model names, specify with --model.", "Switch between Codex-backed models. Your pick becomes the default for new sessions. For direct model names, use --model.") || changed
	changed = replaceAllFixed(data, "Select model", "Codex model") || changed
	changed = replaceAllFixed(data, "Default (recommended)", "Default (Claudodex)") || changed
	changed = replaceAllFixed(data, "Most capable for complex work", "default Codex work") || changed
	changed = replaceAllFixed(data, "Best for everyday tasks", modelDescriptionPatch(modelCfg.Sonnet, "everyday coding")) || changed
	changed = replaceAllFixed(data, "Fastest for quick answers", modelDescriptionPatch(modelCfg.Haiku, "quick code")) || changed
	changed = replaceAllFixed(data, ` with 1M context \xB7 `, ` via Codex model \xB7 `) || changed

	changed = replaceAllPatternString(data, `j4.createElement(V,{bold:!0},"Claude Code")`, "Claude Code", "Claudodex") || changed
	changed = replaceAllPatternString(data, `Lq("claude",d)("Claude Code")`, "Claude Code", "Claudodex") || changed
	changed = replaceAllPatternString(data, `Lq("claude",d)(" Claude Code ")`, "Claude Code", "Claudodex") || changed
	changed = replaceFirstFixed(data, "Lq(\"inactive\",d)(`v${h}`)", quotedVersion(claudodexVersion)) || changed
	changed = replaceFirstFixed(data, `j4.createElement(V,{dimColor:!0},"v",E)`, quotedVersion(claudodexVersion)) || changed
	changed = replaceFirstFixed(data, "w_=h4()?", "w_=0?") || changed
	changed = patchMaxModelPickerBase(data) || changed
	return changed
}

func claudodexInfoLine() string {
	return "Issues: github.com/bassner/claudodex/issues"
}

func quotedVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		version = "dev"
	}
	return `"v` + version + `"`
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

func patchMaxModelPickerBase(data []byte) bool {
	start := bytes.Index(data, []byte("function jl3(H=!1){"))
	if start < 0 {
		return false
	}
	end := bytes.Index(data[start:], []byte("function Jl3("))
	if end < 0 {
		return false
	}
	window := data[start : start+end]
	changed := false
	for _, patch := range []struct {
		old string
		new string
	}{
		{"let z=[ML6(H)]", "let z=[]"},
		{"z.push(lkK())", "void 0"},
		{"z.push(Al3)", "void 0"},
		{"z.push(ckK())", "void 0"},
		{"return z.push(nkK),z", "return z"},
	} {
		changed = replaceFirstFixed(window, patch.old, patch.new) || changed
	}
	return changed
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
