package launcher

import (
	"github.com/bassner/claudodex/internal/modelconfig"

	"os"
	"strings"
)

func ForcePassThrough(args []string) bool {
	return len(args) > 0 && args[0] == "--"
}

func ClaudodexCommand(args []string) (string, []string, bool) {
	if len(args) == 0 || args[0] == "--" {
		return "", nil, false
	}
	if len(args[0]) >= 4 && args[0][:4] == "clx:" {
		return args[0], args[1:], true
	}
	return "", nil, false
}

func DirectClaudeFastPath(args []string) bool {
	if len(args) != 1 {
		return false
	}
	return args[0] == "--help" || args[0] == "--version"
}

func RewriteClaudeModelArgs(args []string) []string {
	return RewriteClaudeModelArgsWithConfig(args, modelconfig.Default())
}

func RewriteClaudeModelArgsWithConfig(args []string, models modelconfig.Config) []string {
	out := append([]string(nil), args...)
	for i := 0; i < len(out); i++ {
		switch out[i] {
		case "--model", "--fallback-model":
			if i+1 < len(out) {
				out[i+1] = claudeRuntimeModel(out[i+1], models)
				i++
			}
		default:
			for _, prefix := range []string{"--model=", "--fallback-model="} {
				if value, ok := strings.CutPrefix(out[i], prefix); ok {
					out[i] = prefix + claudeRuntimeModel(value, models)
					break
				}
			}
		}
	}
	return out
}

func DisableClaudeChrome(args []string) []string {
	out := make([]string, 0, len(args)+1)
	hasNoChrome := false
	for _, arg := range args {
		switch arg {
		case "--chrome":
			continue
		case "--no-chrome":
			if hasNoChrome {
				continue
			}
			hasNoChrome = true
		}
		out = append(out, arg)
	}
	if hasNoChrome {
		return out
	}
	return append([]string{"--no-chrome"}, out...)
}

func explicitModelArg(args []string) (string, bool) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--model":
			if i+1 < len(args) {
				return args[i+1], true
			}
			i++
		default:
			if value, ok := strings.CutPrefix(args[i], "--model="); ok {
				return value, true
			}
		}
	}
	return "", false
}

func claudeRuntimeModel(model string, models modelconfig.Config) string {
	return modelconfig.WithLongContext(models.RuntimeModel(model))
}

func DetectInteractive(args []string, stdin, stdout *os.File) bool {
	if hasAutomationMode(args) {
		return false
	}
	return isTerminal(stdin) && isTerminal(stdout)
}

func hasAutomationMode(args []string) bool {
	for i, arg := range args {
		switch arg {
		case "-p", "--print":
			return true
		case "--output-format":
			if i+1 < len(args) && args[i+1] != "text" {
				return true
			}
		default:
			if value, ok := strings.CutPrefix(arg, "--output-format="); ok && value != "text" {
				return true
			}
		}
	}
	return false
}

func isTerminal(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
