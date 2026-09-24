// Package output renders command results as text or JSON for one CLI execution.
//
// Commands implement the Result interface (a Text() method) and call RenderTo
// to write their result. In JSON mode, the result struct is marshaled
// directly; otherwise, Text() is invoked. The renderer decides the mode once:
// it also picks the progress adapter an operation reports to and renders
// errors, so commands do not branch on the mode for formatting.
package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/progress"
)

// Renderer owns the output mode of one command invocation.
type Renderer struct {
	JSON bool
}

// SetJSONMode enables or disables JSON output for this invocation.
func (r *Renderer) SetJSONMode(on bool) { r.JSON = on }

// IsJSON reports whether JSON output mode is active.
func (r *Renderer) IsJSON() bool { return r.JSON }

// Result is the contract between commands and the renderer.
//
// Text returns the human-readable representation. The renderer adds a trailing
// newline only if the returned string is non-empty and does not already end
// with one. An empty Text() means "no output" in non-JSON mode.
type Result interface {
	Text() string
}

// JSONValuer lets a result use a different payload for JSON than for text.
type JSONValuer interface {
	JSONValue() any
}

// ProgressDriven is embedded by results whose text-mode output was already
// written as progress events while the operation ran (install, component
// add, materialized toolchain links). In text mode the renderer writes
// nothing more for them; in JSON mode the embedding struct is the payload.
type ProgressDriven struct{}

// Text implements Result: progress already said everything.
func (ProgressDriven) Text() string { return "" }

// Progress returns the adapter operations report progress to in the active
// mode: messages on out and the download bar on stderr for humans, nothing
// in JSON mode so stdout stays a single document.
func (renderer *Renderer) Progress(out io.Writer) progress.Sink {
	if renderer.JSON {
		return progress.Discard
	}
	return progress.NewText(out, os.Stderr)
}

// Note writes a human-facing aside to stderr in text mode. JSON consumers
// read the document on stdout and get no aside, so JSON mode writes nothing.
func (renderer *Renderer) Note(stderr io.Writer, msg string) {
	if renderer.JSON {
		return
	}
	_, _ = fmt.Fprintln(stderr, msg)
}

// RenderTo writes r to w in the active output mode.
func (renderer *Renderer) RenderTo(w io.Writer, r Result) error {
	if renderer.JSON {
		payload := any(r)
		if jsonValuer, ok := r.(JSONValuer); ok {
			payload = jsonValuer.JSONValue()
		}
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		return enc.Encode(payload)
	}
	text := r.Text()
	if text == "" {
		return nil
	}
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	_, err := io.WriteString(w, text)
	return err
}

// RenderOutcome writes r for an operation that ended with err, then returns
// err (joined with a render error, if any). Text mode always shows what was
// achieved before the error is printed; JSON mode renders r only when err is
// nil, because the error envelope must be the only document on stdout.
func (renderer *Renderer) RenderOutcome(w io.Writer, r Result, err error) error {
	if err != nil && renderer.JSON {
		return err
	}
	if renderErr := renderer.RenderTo(w, r); renderErr != nil {
		return errors.Join(err, renderErr)
	}
	return err
}

// RenderErrorTo writes err once in the active mode: a JSON error envelope on
// stdout, or "cjv: <message>" on stderr. An ExitCodeError is a transparent
// wrapper used to propagate child process exit codes; it carries no message
// for an end user and produces no output. The original err is returned so
// cobra propagates the exit code unchanged.
func (renderer *Renderer) RenderErrorTo(stdout, stderr io.Writer, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := errors.AsType[*cjverr.ExitCodeError](err); ok {
		return err
	}
	if !renderer.JSON {
		_, _ = fmt.Fprintln(stderr, "cjv:", err)
		return err
	}
	envelope := struct {
		Error errorPayload `json:"error"`
	}{Error: buildErrorPayload(err)}
	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)
	if encErr := enc.Encode(envelope); encErr != nil {
		_, _ = fmt.Fprintln(stderr, "cjv: failed to encode error envelope:", encErr)
	}
	return err
}
