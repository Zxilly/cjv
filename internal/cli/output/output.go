// Package output renders command results as text or JSON for one CLI execution.
//
// Commands implement the Result interface (a Text() method) and call Render
// to write their result to stdout. In JSON mode, the result struct is marshaled
// directly; otherwise, Text() is invoked.
package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
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

// Render writes r to os.Stdout.
func (renderer *Renderer) Render(r Result) error {
	return renderer.RenderTo(os.Stdout, r)
}

// RenderErrorTo writes a JSON error envelope when JSON mode is active.
// In non-JSON mode it does nothing; the caller handles error
// printing via its existing path. In both modes the original err is returned
// so cobra propagates the exit code unchanged.
func (renderer *Renderer) RenderErrorTo(stdout, stderr io.Writer, err error) error {
	if err == nil || !renderer.JSON {
		return err
	}
	// ExitCodeError is a transparent wrapper used to propagate child process
	// exit codes; it carries no semantic information for an end user.
	if _, ok := errors.AsType[*cjverr.ExitCodeError](err); ok {
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

// RenderError writes a JSON error envelope to os.Stdout.
func (renderer *Renderer) RenderError(err error) error {
	return renderer.RenderErrorTo(os.Stdout, os.Stderr, err)
}
