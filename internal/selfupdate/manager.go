package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/bassner/claudodex/internal/auth"
	"github.com/bassner/claudodex/internal/filelock"
	"github.com/bassner/claudodex/internal/launcher"
	"golang.org/x/mod/semver"
	"golang.org/x/term"
)

type Config struct {
	Version       string
	InstallMethod string
	Home          string
	Stdin         *os.File
	Stderr        io.Writer
}

type Request struct {
	// Args is the original, unmodified command line, for an immediate handoff.
	Args   []string
	Manual bool
	Yes    bool
}

type Result struct {
	StatusLineExecutable string
	Reexecuted           bool
}

type Manager struct {
	cfg         Config
	feed        feed
	path        string
	handedOff   bool
	command     commandRunner
	lookPath    func(string) (string, error)
	isHomebrew  func(string) bool
	buildInfo   func(string) (string, string, error)
	pinHelper   func(string, string) (string, error)
	prompt      func(context.Context, io.Writer) (bool, error)
	interactive func([]string, *os.File, io.Writer) bool
	exec        func(string, []string, []string) error
}

func New(cfg Config) *Manager {
	if cfg.Stderr == nil {
		cfg.Stderr = io.Discard
	}
	path, _ := os.Executable()
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	m := &Manager{cfg: cfg, feed: productionFeed(), path: path, lookPath: exec.LookPath, isHomebrew: homebrewInstall, buildInfo: installedBuild, pinHelper: launcher.PinStatusLineExecutable, prompt: promptConsent, interactive: updateInteractive, exec: replaceProcess}
	m.command = m.runCommand
	m.handedOff = consumeHandoff(cfg.Home, path, cfg.Version)
	return m
}

func newer(offered, current string) bool {
	o, err := StableVersion(offered)
	if err != nil {
		return false
	}
	c, err := StableVersion(current)
	if err != nil {
		return false
	}
	return semver.Compare("v"+o, "v"+c) > 0
}

func (m *Manager) BeforeLaunch(ctx context.Context, r Request) (result Result, err error) {
	if m.handedOff {
		m.handedOff = false
		if r.Manual {
			fmt.Fprintf(m.cfg.Stderr, "Updated Claudodex to %s.\n", m.cfg.Version)
		}
		return result, nil
	}
	if os.Getenv("CLAUDODEX_CHILD") != "" {
		if r.Manual {
			return result, errors.New("run clx:update outside a nested Claudodex session")
		}
		return result, nil
	}
	if !r.Manual && os.Getenv("CLAUDODEX_NO_AUTO_UPDATE") == "1" {
		return result, nil
	}
	loadCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	offered, err := m.feed.load(loadCtx, "")
	cancel()
	if err != nil {
		if r.Manual {
			return result, err
		}
		if errors.Is(err, ErrUnsupportedSchema) {
			fmt.Fprintf(m.cfg.Stderr, "Claudodex: %v\n", err)
		}
		return result, nil
	}
	if !newer(offered.Version, m.cfg.Version) {
		if r.Manual {
			fmt.Fprintln(m.cfg.Stderr, "Claudodex is up to date, or this is a development build. Source builds update manually.")
		}
		return result, nil
	}
	interactive := m.interactive(r.Args, m.cfg.Stdin, m.cfg.Stderr)
	if r.Manual {
		errFile, ok := m.cfg.Stderr.(*os.File)
		interactive = ok && m.cfg.Stdin != nil && term.IsTerminal(int(m.cfg.Stdin.Fd())) && term.IsTerminal(int(errFile.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
	}
	if !interactive && !r.Yes {
		fmt.Fprintf(m.cfg.Stderr, "Claudodex %s is available: %s (run claudodex clx:update).\n", offered.Version, ReleaseURL(offered.Version))
		if r.Manual {
			return result, errors.New("update needs a terminal or explicit clx:update --yes outside a nested session")
		}
		return result, nil
	}
	if !m.isHomebrew(m.path) {
		fmt.Fprintf(m.cfg.Stderr, "Claudodex %s is available: %s\nThis installation updates manually. Preferred macOS install: brew install %s\n", offered.Version, ReleaseURL(offered.Version), Formula)
		if r.Manual {
			return result, errors.New("this build is not owned by the Claudodex Homebrew tap")
		}
		return result, nil
	}
	updateCtx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	asked := false
	ask := func(manifest Manifest) (bool, error) {
		if err := showOffer(m.cfg.Stderr, m.cfg.Version, manifest); err != nil {
			return false, err
		}
		fmt.Fprintf(m.cfg.Stderr, "After Yes: brew update, then brew upgrade --formula --yes %s.\nHomebrew manages required dependencies; a source build may need Go and take longer.\n", Formula)
		if r.Yes {
			if asked {
				return false, errors.New("Homebrew's version changed; rerun clx:update to accept that release")
			}
			asked = true
			return true, nil
		}
		return m.prompt(updateCtx, m.cfg.Stderr)
	}
	yes, err := ask(offered)
	if err != nil || !yes {
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
			return result, err
		}
		fmt.Fprintln(m.cfg.Stderr, "Continuing with the current Claudodex version.")
		return result, nil
	}
	dataDir, err := auth.DataDir(m.cfg.Home)
	if err != nil {
		return result, err
	}
	release, err := filelock.Acquire(updateCtx, filepath.Join(dataDir, "updates", "install.lock"))
	if err != nil {
		return result, err
	}
	defer release()
	// Retain the current helper before Homebrew can remove its old keg.
	result.StatusLineExecutable, err = m.pinHelper(m.cfg.Home, m.path)
	if err != nil {
		return result, fmt.Errorf("retain current statusline helper: %w", err)
	}
	path, err := m.updateHomebrew(updateCtx, offered, ask)
	if err != nil {
		return result, err
	}
	actual, _, err := m.buildInfo(path)
	if err != nil {
		return result, err
	}
	token, cleanup, err := createHandoff(m.cfg.Home, path, actual)
	if err != nil {
		return result, err
	}
	defer cleanup()
	// Never carry an updater lock or temporary signal handler across exec.
	release()
	stop()
	if err := m.exec(path, append([]string{path}, r.Args...), handoffEnvironment(os.Environ(), token)); err != nil {
		return result, fmt.Errorf("start accepted update: %w", err)
	}
	result.Reexecuted = true // Only injected executors return after a successful exec.
	return result, nil
}
