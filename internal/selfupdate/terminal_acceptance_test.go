package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// This opt-in harness exercises the real cooked terminal prompt in tmux. The
// signed feed and package-manager calls stay local; it never installs software.
func TestUpdateTerminalAcceptance(t *testing.T) {
	scenario := os.Getenv("CLAUDODEX_UPDATE_TERMINAL_ACCEPTANCE")
	if scenario == "" {
		t.Skip("opt-in tmux acceptance harness")
	}
	m, _, _ := managerFixture(t)
	m.cfg.Stdin = os.Stdin
	m.cfg.Stderr = os.Stderr
	m.interactive = updateInteractive
	m.prompt = promptConsent
	if strings.HasPrefix(scenario, "typed-ahead") {
		m.prompt = func(ctx context.Context, out io.Writer) (bool, error) {
			fmt.Fprintln(out, "Acceptance: type ahead now; the question follows shortly.")
			select {
			case <-ctx.Done():
				return false, ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
			return promptConsent(ctx, out)
		}
	}
	if scenario == "brew-cancel" {
		m.command = func(ctx context.Context, _ string, _ []string, _ []string, _ bool) ([]byte, error) {
			fmt.Fprintln(os.Stderr, "Acceptance package manager is waiting; Ctrl-C cancels.")
			<-ctx.Done()
			return nil, ctx.Err()
		}
	}
	m.exec = func(string, []string, []string) error {
		fmt.Fprintln(os.Stderr, "ACCEPTANCE: updated version selected in this invocation.")
		return nil
	}
	result, err := m.BeforeLaunch(context.Background(), Request{Args: managerArgs()})
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if !result.Reexecuted {
		fmt.Fprintln(os.Stderr, "ACCEPTANCE: current version continues.")
	}
}
