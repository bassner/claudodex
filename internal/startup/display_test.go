package startup

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestDisplayRestoresTerminalBeforeErrorAndOnlyOnce(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	var output bytes.Buffer
	d := New(&output, "0.5.3", true)
	d.Info("🔎", "Loading models")
	_, _ = d.Warnings().Write([]byte("Claudodex: using cached metadata\n"))
	d.Error(errors.New("model discovery failed"))
	d.Close()
	text := output.String()
	if !strings.Contains(text, logo) || !strings.Contains(text, "v0.5.3") || !strings.Contains(text, "[INFO]") || !strings.Contains(text, "[WARN]") {
		t.Fatalf("missing banner or status: %q", text)
	}
	restore := strings.Index(text, "\x1b[?1049l")
	if restore < 0 || strings.Count(text, "\x1b[?1049l") != 1 || strings.Index(text, "[ERROR]") < restore {
		t.Fatalf("error was hidden or terminal restored more than once: %q", text)
	}
	if strings.LastIndex(text, "[WARN]") < restore || strings.Count(text, "using cached metadata") != 2 {
		t.Fatal("startup warning was lost at handoff or replayed more than once")
	}
	before := output.Len()
	d.Info("🔎", "Should not appear after handoff")
	if output.Len() != before {
		t.Fatal("startup messages continued after handoff")
	}
}

func TestDisplayContextEndsAtHandoffWithoutCancellingRuntime(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	var output bytes.Buffer
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	d := New(&output, "0.5.3", true)
	ctx := d.Context(parent)
	d.Close()
	if ctx.Err() == nil || parent.Err() != nil {
		t.Fatal("startup context did not stop independently of the runtime context")
	}
}

func TestDisplayLeavesAutomationAndDumbTerminalsPlain(t *testing.T) {
	for _, interactive := range []bool{false, true} {
		t.Run("plain", func(t *testing.T) {
			t.Setenv("TERM", "dumb")
			var output bytes.Buffer
			d := New(&output, "0.5.3", interactive)
			d.Info("🔎", "Loading models")
			if output.Len() != 0 {
				t.Fatal("banner polluted automation output")
			}
			d.Error(errors.New("failed"))
			if output.String() != "failed\n" {
				t.Fatalf("error output = %q", output.String())
			}
		})
	}
}
