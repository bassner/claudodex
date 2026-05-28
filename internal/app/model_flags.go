package app

import (
	"fmt"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

func extractStartupModelConfig(args []string, base modelconfig.Config) ([]string, modelconfig.Config, error) {
	models := base.Normalize()
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			out = append(out, args[i:]...)
			break
		}
		name, value, hasInlineValue := strings.Cut(arg, "=")
		switch name {
		case "--claudodex-opus-model":
			next, consumed, err := startupFlagValue(args, i, value, hasInlineValue, name)
			if err != nil {
				return nil, modelconfig.Config{}, err
			}
			models.Opus = next
			i += consumed
		case "--claudodex-sonnet-model":
			next, consumed, err := startupFlagValue(args, i, value, hasInlineValue, name)
			if err != nil {
				return nil, modelconfig.Config{}, err
			}
			models.Sonnet = next
			i += consumed
		case "--claudodex-haiku-model":
			next, consumed, err := startupFlagValue(args, i, value, hasInlineValue, name)
			if err != nil {
				return nil, modelconfig.Config{}, err
			}
			models.Haiku = next
			i += consumed
		case "--claudodex-models":
			next, consumed, err := startupFlagValue(args, i, value, hasInlineValue, name)
			if err != nil {
				return nil, modelconfig.Config{}, err
			}
			var mapErr error
			models, mapErr = applyModelMapFlag(models, next)
			if mapErr != nil {
				return nil, modelconfig.Config{}, mapErr
			}
			i += consumed
		default:
			out = append(out, arg)
		}
	}
	return out, models.Normalize(), nil
}

func startupFlagValue(args []string, index int, inlineValue string, hasInlineValue bool, name string) (string, int, error) {
	if hasInlineValue {
		value := strings.TrimSpace(inlineValue)
		if value == "" {
			return "", 0, fmt.Errorf("missing value for %s", name)
		}
		return value, 0, nil
	}
	if index+1 >= len(args) {
		return "", 0, fmt.Errorf("missing value for %s", name)
	}
	value := strings.TrimSpace(args[index+1])
	if value == "" {
		return "", 0, fmt.Errorf("missing value for %s", name)
	}
	return value, 1, nil
}

func applyModelMapFlag(models modelconfig.Config, value string) (modelconfig.Config, error) {
	for _, entry := range strings.Split(value, ",") {
		key, target, ok := strings.Cut(entry, "=")
		if !ok {
			return modelconfig.Config{}, fmt.Errorf("invalid --claudodex-models entry %q; want opus=model,sonnet=model,haiku=model", entry)
		}
		target = strings.TrimSpace(target)
		if target == "" {
			return modelconfig.Config{}, fmt.Errorf("missing model in --claudodex-models entry %q", entry)
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "opus":
			models.Opus = target
		case "sonnet":
			models.Sonnet = target
		case "haiku":
			models.Haiku = target
		default:
			return modelconfig.Config{}, fmt.Errorf("unknown --claudodex-models key %q", key)
		}
	}
	return models, nil
}
