package launcher

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/bassner/claudodex/internal/modelconfig"
)

func normalizeSharedClaudeSettings(userHome string, wait time.Duration, modelCfg modelconfig.Config) error {
	projectSettingsPath, localSettingsPath := projectClaudeSettingsPaths()
	paths := []string{
		filepath.Join(userHome, ".claude", "settings.json"),
		projectSettingsPath,
		localSettingsPath,
	}
	var errs []error
	for _, path := range paths {
		if path == "" {
			continue
		}
		if err := normalizeClaudeSettingsModel(path, wait, modelCfg); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func normalizeClaudeSettingsModel(path string, wait time.Duration, modelCfg modelconfig.Config) error {
	if _, err := os.Lstat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return withClaudeConfigLocks([]string{path}, wait, func() error {
		settings, err := readJSONMap(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		model, _ := settings["model"].(string)
		alias, ok := codexRuntimeSettingsAlias(model, modelCfg)
		if !ok {
			return nil
		}
		next := cloneJSONMap(settings)
		next["model"] = alias
		if reflect.DeepEqual(settings, next) {
			return nil
		}
		return writeJSONFile(path, next, 0o600)
	})
}

func codexRuntimeSettingsAlias(model string, modelCfg modelconfig.Config) (string, bool) {
	if alias, ok := modelCfg.SettingsAliasForTarget(model); ok {
		return alias, true
	}
	if family, ok := modelconfig.FamilyForModel(model); ok {
		return string(family), true
	}
	return "", false
}
