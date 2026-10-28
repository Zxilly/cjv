package initcmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// initConsole routes the wizard through the command's writers so embedded
// callers and tests receive the same output as a terminal invocation.
type initConsole struct {
	out io.Writer
	err io.Writer
}

func newInitConsole(cmd *cobra.Command) initConsole {
	return initConsole{out: cmd.OutOrStdout(), err: cmd.ErrOrStderr()}
}

func (c initConsole) println(a ...any) {
	_, _ = fmt.Fprintln(c.out, a...)
}

func (c initConsole) printf(format string, a ...any) {
	_, _ = fmt.Fprintf(c.out, format, a...)
}

func (c initConsole) paragraph(text string) {
	_, _ = io.WriteString(c.out, renderInitText(text, writerIsTerminal(c.out)))
}

// The wizard's translated paragraphs use only inline code spans. Preserve
// their text and explicit line breaks without loading a Markdown renderer
// and syntax highlighters for languages the wizard never displays.
func renderInitText(text string, terminal bool) string {
	codeStyle := color.New(color.FgCyan)
	if !terminal {
		codeStyle.DisableColor()
	}
	var rendered strings.Builder
	for {
		prefix, rest, found := strings.Cut(text, "`")
		rendered.WriteString(prefix)
		if !found {
			break
		}
		code, suffix, closed := strings.Cut(rest, "`")
		if !closed {
			rendered.WriteByte('`')
			rendered.WriteString(rest)
			break
		}
		rendered.WriteString(codeStyle.Sprint(code))
		text = suffix
	}
	// Keep the existing 100-column paragraph layout with two columns of
	// margin on each side. ANSI-aware wrapping also handles wide CJK text.
	wrapped := ansi.Wrap(rendered.String(), 96, "")
	return "\n  " + strings.ReplaceAll(wrapped, "\n", "\n  ") + "\n\n"
}

func writerIsTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fd := f.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
