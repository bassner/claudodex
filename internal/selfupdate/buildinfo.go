package selfupdate

import (
	"debug/buildinfo"
	"errors"
	"strings"
	"unicode"
)

// installedBuild reads recorded linker settings without executing the candidate.
func installedBuild(path string) (version, method string, err error) {
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	for _, setting := range info.Settings {
		if setting.Key != "-ldflags" {
			continue
		}
		args, splitErr := splitFlags(setting.Value)
		if splitErr != nil {
			return "", "", splitErr
		}
		for i := 0; i < len(args); i++ {
			value := ""
			if args[i] == "-X" && i+1 < len(args) {
				i++
				value = args[i]
			} else if strings.HasPrefix(args[i], "-X=") {
				value = strings.TrimPrefix(args[i], "-X=")
			}
			if v, ok := strings.CutPrefix(value, "main.version="); ok {
				version = v
			}
			if v, ok := strings.CutPrefix(value, "main.installMethod="); ok {
				method = v
			}
		}
	}
	if _, err := StableVersion(version); err != nil {
		return "", "", errors.New("installed package has no stable recorded version")
	}
	return version, method, nil
}

func splitFlags(s string) ([]string, error) {
	var args []string
	var word strings.Builder
	var quote rune
	started, escape := false, false
	for _, r := range s {
		if escape {
			word.WriteRune(r)
			escape = false
			started = true
			continue
		}
		if r == '\\' && quote != '\'' {
			escape = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				word.WriteRune(r)
			}
			started = true
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			started = true
			continue
		}
		if unicode.IsSpace(r) {
			if started {
				args = append(args, word.String())
				word.Reset()
				started = false
			}
			continue
		}
		word.WriteRune(r)
		started = true
	}
	if quote != 0 || escape {
		return nil, errors.New("invalid recorded build flags")
	}
	if started {
		args = append(args, word.String())
	}
	return args, nil
}
