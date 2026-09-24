package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/progress"
)

type fixture struct {
	A int    `json:"a"`
	B string `json:"b"`
}

func (f fixture) Text() string { return f.B }

func TestRender_JSONMode_EmitsCompactJSON(t *testing.T) {
	var renderer Renderer
	renderer.SetJSONMode(true)

	var buf bytes.Buffer
	requireNoError(t, renderer.RenderTo(&buf, fixture{A: 1, B: "hello"}))
	out := buf.String()

	var got fixture
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &got); err != nil {
		t.Fatalf("invalid JSON %q: %v", out, err)
	}
	if got.A != 1 || got.B != "hello" {
		t.Fatalf("unexpected payload: %+v", got)
	}
}

func TestRender_TextMode_CallsTextAndAppendsNewline(t *testing.T) {
	var renderer Renderer
	renderer.SetJSONMode(false)

	var buf bytes.Buffer
	requireNoError(t, renderer.RenderTo(&buf, fixture{B: "hello"}))
	out := buf.String()
	if out != "hello\n" {
		t.Fatalf("expected %q, got %q", "hello\n", out)
	}
}

func TestRender_TextMode_EmptyTextProducesNoOutput(t *testing.T) {
	var renderer Renderer
	renderer.SetJSONMode(false)

	var buf bytes.Buffer
	requireNoError(t, renderer.RenderTo(&buf, fixture{B: ""}))
	out := buf.String()
	if out != "" {
		t.Fatalf("expected empty, got %q", out)
	}
}

func TestRenderError_MapsCjverrTypesToCodes(t *testing.T) {
	var renderer Renderer
	renderer.SetJSONMode(true)

	cases := []struct {
		err      error
		wantCode cjverr.ErrorCode
	}{
		{&cjverr.ToolchainNotInstalledError{Name: "lts-1.0.5"}, cjverr.ErrorCodeToolchainNotInstalled},
		{&cjverr.NoToolchainConfiguredError{}, cjverr.ErrorCodeNoToolchainConfigured},
		{&cjverr.UnknownComponentError{Name: "foo"}, cjverr.ErrorCodeUnknownComponent},
		{&cjverr.UnsupportedForJSONError{Command: "exec"}, cjverr.ErrorCodeUnsupportedForJSON},
	}

	for _, tc := range cases {
		var stdout, stderr bytes.Buffer
		requireSameError(t, renderer.RenderErrorTo(&stdout, &stderr, tc.err), tc.err)
		out := stdout.String()

		var env struct {
			Error errorPayload `json:"error"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &env); err != nil {
			t.Fatalf("invalid JSON for %T: %v (%q)", tc.err, err, out)
		}
		if env.Error.Code != tc.wantCode {
			t.Errorf("for %T: got code %q, want %q", tc.err, env.Error.Code, tc.wantCode)
		}
		if env.Error.Message == "" {
			t.Errorf("for %T: empty message", tc.err)
		}
		if env.Error.Details == nil {
			t.Errorf("for %T: details should be non-nil object", tc.err)
		}
	}
}

func TestRenderError_ExitCodeErrorIsPassthrough(t *testing.T) {
	var renderer Renderer
	renderer.SetJSONMode(true)

	var stdout, stderr bytes.Buffer
	err := &cjverr.ExitCodeError{Code: 2}
	requireSameError(t, renderer.RenderErrorTo(&stdout, &stderr, err), err)
	out := stdout.String()
	if out != "" {
		t.Fatalf("ExitCodeError should not produce envelope output, got %q", out)
	}
}

func TestRenderError_TextModeWritesMessageToStderr(t *testing.T) {
	var renderer Renderer
	renderer.SetJSONMode(false)

	var stdout, stderr bytes.Buffer
	err := &cjverr.NoToolchainConfiguredError{}
	requireSameError(t, renderer.RenderErrorTo(&stdout, &stderr, err), err)
	if out := stdout.String(); out != "" {
		t.Fatalf("text mode should not write to stdout, got %q", out)
	}
	if got, want := stderr.String(), "cjv: "+err.Error()+"\n"; got != want {
		t.Fatalf("expected %q on stderr, got %q", want, got)
	}

	stderr.Reset()
	exit := &cjverr.ExitCodeError{Code: 2}
	requireSameError(t, renderer.RenderErrorTo(&stdout, &stderr, exit), exit)
	if got := stderr.String(); got != "" {
		t.Fatalf("ExitCodeError carries no message, got %q", got)
	}
}

func TestNote_TextModeOnly(t *testing.T) {
	var renderer Renderer
	var stderr bytes.Buffer

	renderer.SetJSONMode(false)
	renderer.Note(&stderr, "aside")
	if got, want := stderr.String(), "aside\n"; got != want {
		t.Fatalf("expected %q on stderr, got %q", want, got)
	}

	stderr.Reset()
	renderer.SetJSONMode(true)
	renderer.Note(&stderr, "aside")
	if got := stderr.String(); got != "" {
		t.Fatalf("JSON mode writes no aside, got %q", got)
	}
}

func TestRenderOutcome_JSONModeLeavesErrorEnvelopeAlone(t *testing.T) {
	var renderer Renderer
	renderer.SetJSONMode(true)
	failure := errors.New("partial")

	var buf bytes.Buffer
	requireSameError(t, renderer.RenderOutcome(&buf, fixture{A: 1, B: "done"}, failure), failure)
	if out := buf.String(); out != "" {
		t.Fatalf("JSON mode must not emit a result next to the error envelope, got %q", out)
	}
	requireNoError(t, renderer.RenderOutcome(&buf, fixture{A: 1, B: "done"}, nil))
	if !json.Valid(bytes.TrimSpace(buf.Bytes())) {
		t.Fatalf("expected JSON result, got %q", buf.String())
	}

	renderer.SetJSONMode(false)
	buf.Reset()
	requireSameError(t, renderer.RenderOutcome(&buf, fixture{B: "done"}, failure), failure)
	if out := buf.String(); out != "done\n" {
		t.Fatalf("text mode shows what was achieved before the error, got %q", out)
	}
}

func TestProgressAdapterFollowsMode(t *testing.T) {
	var renderer Renderer
	var out bytes.Buffer
	renderer.SetJSONMode(true)
	if sink := renderer.Progress(&out); sink != progress.Discard {
		t.Fatalf("JSON mode must discard progress, got %T", sink)
	}
	renderer.SetJSONMode(false)
	renderer.Progress(&out).Report(progress.Event{Kind: progress.FetchingManifest})
	if out.Len() == 0 {
		t.Fatal("text mode must render progress on the command writer")
	}
	if got := (ProgressDriven{}).Text(); got != "" {
		t.Fatalf("progress-driven results render no text, got %q", got)
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func requireSameError(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}
