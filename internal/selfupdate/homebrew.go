package selfupdate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const Formula = "bassner/tap/claudodex"

type commandRunner func(context.Context, string, []string, []string, bool) ([]byte, error)

func homebrewInstall(path string) bool {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	keg := filepath.Dir(filepath.Dir(resolved))
	formulaDir := filepath.Dir(keg)
	if filepath.Base(resolved) != "claudodex" || filepath.Base(filepath.Dir(resolved)) != "bin" || filepath.Base(formulaDir) != "claudodex" || filepath.Base(filepath.Dir(formulaDir)) != "Cellar" {
		return false
	}
	data, err := os.ReadFile(filepath.Join(keg, "INSTALL_RECEIPT.json"))
	if err != nil || len(data) > 1<<20 {
		return false
	}
	var receipt struct {
		Source struct {
			Tap string `json:"tap"`
		} `json:"source"`
	}
	return json.Unmarshal(data, &receipt) == nil && receipt.Source.Tap == "bassner/tap"
}

func brewEnvironment(env []string) []string {
	out := make([]string, 0, len(env)+3)
	for _, entry := range env {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "HOMEBREW_NO_") || strings.HasPrefix(name, "OPENAI_") || strings.HasPrefix(name, "ANTHROPIC_") || name == "CLAUDE_CODE_OAUTH_TOKEN" || name == "CODEX_API_KEY" || name == handoffEnv {
			continue
		}
		out = append(out, entry)
	}
	return append(out, "HOMEBREW_NO_AUTO_UPDATE=1", "HOMEBREW_NO_INSTALLED_DEPENDENTS_CHECK=1", "HOMEBREW_NO_INSTALL_CLEANUP=1")
}

func (m *Manager) runCommand(ctx context.Context, path string, args, env []string, live bool) ([]byte, error) {
	cmd := exec.Command(path, args...)
	cmd.Env = env
	// A package operation must not consume the prompt piped to Claudodex.
	// A nil Stdin makes os/exec connect the child to the null device.
	var output bytes.Buffer
	cmd.Stdout = &output
	if live {
		cmd.Stdout = m.cfg.Stderr
	}
	cmd.Stderr = m.cfg.Stderr
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return output.Bytes(), err
	case <-ctx.Done():
		_ = cmd.Process.Signal(os.Interrupt)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
		return nil, ctx.Err()
	}
}

func brewVersion(data []byte) (string, error) {
	var info struct {
		Formulae []struct {
			FullName string `json:"full_name"`
			Versions struct {
				Stable string `json:"stable"`
			} `json:"versions"`
		} `json:"formulae"`
	}
	if len(data) > 2<<20 || json.Unmarshal(data, &info) != nil || len(info.Formulae) != 1 || info.Formulae[0].FullName != Formula {
		return "", errors.New("unexpected Homebrew formula metadata")
	}
	return StableVersion(info.Formulae[0].Versions.Stable)
}

func (m *Manager) updateHomebrew(ctx context.Context, offered Manifest, ask func(Manifest) (bool, error)) (string, error) {
	brew, err := m.lookPath("brew")
	if err != nil {
		return "", errors.New("Homebrew was not found; run brew upgrade bassner/tap/claudodex manually")
	}
	env := brewEnvironment(os.Environ())
	if _, err = m.command(ctx, brew, []string{"update"}, env, true); err != nil {
		return "", fmt.Errorf("refresh Homebrew: %w", err)
	}
	data, err := m.command(ctx, brew, []string{"info", "--json=v2", "--formula", Formula}, env, false)
	if err != nil {
		return "", err
	}
	version, err := brewVersion(data)
	if err != nil {
		return "", err
	}
	if version != offered.Version {
		if !newer(version, m.cfg.Version) {
			return "", errors.New("Homebrew does not yet offer the announced update; try again later")
		}
		loadCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		offered, err = m.feed.load(loadCtx, version)
		cancel()
		if err != nil {
			return "", fmt.Errorf("verify changed Homebrew version: %w", err)
		}
		yes, err := ask(offered)
		if err != nil {
			return "", err
		}
		if !yes {
			return "", errors.New("changed Homebrew update declined")
		}
	}
	if _, err = m.command(ctx, brew, []string{"upgrade", "--formula", "--yes", Formula}, env, true); err != nil {
		return "", fmt.Errorf("upgrade Homebrew package: %w", err)
	}
	data, err = m.command(ctx, brew, []string{"--prefix", Formula}, env, false)
	if err != nil {
		return "", err
	}
	prefix := strings.TrimSpace(string(data))
	if !filepath.IsAbs(prefix) || strings.ContainsAny(prefix, "\r\n\x00") {
		return "", errors.New("invalid installed Homebrew prefix")
	}
	path, err := filepath.EvalSymlinks(filepath.Join(prefix, "bin", "claudodex"))
	if err != nil {
		return "", err
	}
	if !m.isHomebrew(path) {
		return "", errors.New("updated executable is outside the expected Homebrew package")
	}
	actual, method, err := m.buildInfo(path)
	if err != nil {
		return "", err
	}
	if actual != offered.Version || method != "homebrew" {
		return "", errors.New("installed Homebrew build differs from the accepted release; current process will continue")
	}
	return path, nil
}
