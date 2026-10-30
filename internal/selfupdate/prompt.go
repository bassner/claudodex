package selfupdate

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"golang.org/x/term"
)

// PlainNotes limits output and removes terminal commands and invisible direction
// controls. Release notes are displayed as text, never interpreted as commands.
func PlainNotes(notes string) string {
	var out strings.Builder
	lines, count := 0, 0
	for _, r := range notes {
		if count >= 4000 || lines >= 40 {
			out.WriteString("\n[More in the release link below.]\n")
			break
		}
		if r == '\n' {
			lines++
			out.WriteRune(r)
			count++
			continue
		}
		if r == '\t' {
			out.WriteString("    ")
			count += 4
			continue
		}
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf, unicode.Zl, unicode.Zp) {
			continue
		}
		out.WriteRune(r)
		count++
	}
	return strings.TrimSpace(out.String())
}

func updateInteractive(args []string, stdin *os.File, stderr io.Writer) bool {
	errFile, ok := stderr.(*os.File)
	if !ok || stdin == nil || !term.IsTerminal(int(stdin.Fd())) || !term.IsTerminal(int(errFile.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "clx:") {
			return false
		}
		switch arg {
		case "-p", "--print", "--help", "-h", "--version", "-v", "auth", "mcp", "doctor", "install", "update":
			return false
		}
		if strings.HasPrefix(arg, "--print=") || strings.HasPrefix(arg, "--output-format") || strings.HasPrefix(arg, "--input-format") {
			return false
		}
	}
	return true
}

func showOffer(out io.Writer, current string, m Manifest) error {
	_, err := fmt.Fprintf(out, "\nClaudodex %s is available (installed: %s).\n\n%s\n\n%s\n", m.Version, current, PlainNotes(m.ReleaseNotes), ReleaseURL(m.Version))
	return err
}

func promptConsent(ctx context.Context, out io.Writer) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := flushTerminalInput(); err != nil {
		return false, err
	}
	if _, err := fmt.Fprint(out, "Update Claudodex now? [y/N] "); err != nil {
		return false, err
	}
	answer, err := readTerminalLine(ctx)
	fmt.Fprintln(out)
	if err != nil {
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}
