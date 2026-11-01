package startup

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/term"
)

const logo = ` ▗▄▄▖▗▖    ▗▄▖ ▗▖ ▗▖▗▄▄▄   ▗▄▖ ▗▄▄▄  ▗▄▄▄▖▗▖  ▗▖
▐▌   ▐▌   ▐▌ ▐▌▐▌ ▐▌▐▌  █ ▐▌ ▐▌▐▌  █ ▐▌    ▝▚▞▘
▐▌   ▐▌   ▐▛▀▜▌▐▌ ▐▌▐▌  █ ▐▌ ▐▌▐▌  █ ▐▛▀▀▘  ▐▌
▝▚▄▄▖▐▙▄▄▖▐▌ ▐▌▝▚▄▞▘▐▙▄▄▀ ▝▚▄▞▘▐▙▄▄▀ ▐▙▄▄▖▗▞▘▝▚▖`

type Display struct {
	out      io.Writer
	active   bool
	styled   bool
	colored  bool
	stop     context.CancelFunc
	warnings []string
}

// Catch interrupts only during startup. Closing restores the terminal and the
// normal signal handling before Claude Code takes control.
func (d *Display) Context(parent context.Context) context.Context {
	if !d.active || d.stop != nil {
		return parent
	}
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	d.stop = stop
	return ctx
}

func New(out io.Writer, version string, interactive bool) *Display {
	d := &Display{out: out, styled: interactive && os.Getenv("TERM") != "dumb"}
	if file, ok := out.(*os.File); ok && !term.IsTerminal(int(file.Fd())) {
		d.styled = false
	}
	if !d.styled {
		return d
	}
	d.active = true
	d.colored = os.Getenv("NO_COLOR") == ""
	fmt.Fprint(out, "\x1b[?1049h\x1b[H")
	width := 80
	if file, ok := out.(*os.File); ok {
		if columns, _, err := term.GetSize(int(file.Fd())); err == nil {
			width = columns
		}
	}
	if width >= 60 {
		fmt.Fprintln(out, d.color("36", logo))
	}
	fmt.Fprintf(out, "\n  %s v%s\n\n", d.color("1", "Claudodex"), version)
	return d
}

func (d *Display) color(code, text string) string {
	if !d.colored {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func (d *Display) Info(icon, format string, args ...any) {
	if d.active {
		fmt.Fprintf(d.out, "  %s %s %s\n", d.color("34", "[INFO] "), icon, fmt.Sprintf(format, args...))
	}
}

func (d *Display) Warn(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if d.active {
		d.warnings = append(d.warnings, message)
	}
	if d.styled {
		fmt.Fprintf(d.out, "  %s ⚠️  %s\n", d.color("33", "[WARN] "), message)
	} else {
		fmt.Fprintf(d.out, "Claudodex: %s\n", message)
	}
}

func (d *Display) Error(err error) {
	d.Close()
	if d.styled {
		fmt.Fprintf(d.out, "  %s ❌ %v\n", d.color("31", "[ERROR]"), err)
	} else {
		fmt.Fprintln(d.out, err)
	}
}

func (d *Display) Close() {
	if d.active {
		d.active = false
		fmt.Fprint(d.out, "\x1b[?1049l")
		for _, warning := range d.warnings {
			d.Warn("%s", warning)
		}
		d.warnings = nil
	}
	if d.stop != nil {
		d.stop()
		d.stop = nil
	}
}

// Existing startup diagnostics use stderr for warnings. Child stderr stays raw.
func (d *Display) Warnings() io.Writer {
	if !d.active {
		return d.out
	}
	return warningWriter{d}
}

type warningWriter struct{ display *Display }

func (w warningWriter) Write(p []byte) (int, error) {
	for _, line := range strings.Split(strings.TrimRight(string(p), "\n"), "\n") {
		w.display.Warn("%s", strings.TrimPrefix(line, "Claudodex: "))
	}
	return len(p), nil
}
