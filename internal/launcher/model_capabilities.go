package launcher

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
)

const claudeModelCapabilitiesFileName = "model-capabilities.json"

type claudeModelCapabilitiesCache struct {
	Models    []claudeModelCapability `json:"models"`
	Timestamp int64                   `json:"timestamp"`
}

type claudeModelCapability struct {
	ID             string `json:"id"`
	MaxInputTokens int64  `json:"max_input_tokens,omitempty"`
	MaxTokens      int64  `json:"max_tokens,omitempty"`
}

func WriteClaudeModelCapabilitiesCache(sidecarDir string, codexModels []codex.ModelInfo, modelCfg modelconfig.Config) error {
	cacheDir := filepath.Join(sidecarDir, "cache")
	if err := ensurePrivateClaudeCacheDir(cacheDir); err != nil {
		return err
	}
	if err := writeJSONFile(filepath.Join(cacheDir, claudeModelCapabilitiesFileName), claudeModelCapabilitiesCache{
		Models:    claudeModelCapabilities(codexModels, modelCfg),
		Timestamp: time.Now().UnixMilli(),
	}, 0o600); err != nil {
		return err
	}
	return writeClaudeContextCompatibilityCache(sidecarDir, codexModels, modelCfg)
}

func ensurePrivateClaudeCacheDir(cacheDir string) error {
	info, err := os.Lstat(cacheDir)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			if err := os.RemoveAll(cacheDir); err != nil {
				return err
			}
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(cacheDir, 0o700)
}

func claudeModelCapabilities(models []codex.ModelInfo, modelCfg modelconfig.Config) []claudeModelCapability {
	modelCfg = modelCfg.Normalize()
	specs := append(modelconfig.DirectModelSpecs(modelCfg), modelconfig.ClaudeAliasSpecs(modelCfg)...)
	specs = append(specs, modelconfig.FamilyAliasSpecs()...)
	out := make([]claudeModelCapability, 0, len(specs))
	for _, spec := range specs {
		target := modelCfg.Target(spec.Family)
		out = append(out, claudeModelCapability{
			ID:             spec.ID,
			MaxInputTokens: modelContextWindow(models, target),
			MaxTokens:      128_000,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if len(out[i].ID) != len(out[j].ID) {
			return len(out[i].ID) > len(out[j].ID)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func writeClaudeContextCompatibilityCache(sidecarDir string, models []codex.ModelInfo, modelCfg modelconfig.Config) error {
	modelCfg = modelCfg.Normalize()
	path := filepath.Join(sidecarDir, claudeGlobalConfigName)
	return withClaudeConfigLocks([]string{path}, claudeConfigLockWait, func() error {
		config, err := readJSONMap(path)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			config = map[string]any{}
		}
		next := cloneJSONMap(config)
		clientData := mapValue(next["clientDataCache"])
		if clientData == nil {
			clientData = map[string]any{}
		} else {
			clientData = cloneJSONMap(clientData)
		}
		clientData["kelp_forest_sonnet"] = strconv.FormatInt(modelContextWindow(models, modelCfg.Sonnet), 10)
		next["clientDataCache"] = clientData
		if reflect.DeepEqual(config, next) {
			return nil
		}
		return writeJSONFile(path, next, 0o600)
	})
}
