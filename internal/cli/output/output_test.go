package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
)

type fixture struct {
	A int    `json:"a"`
	B string `json:"b"`
}

func (f fixture) Text() string { return f.B }

func TestRender_JSONMode_EmitsCompactJSON(t *testing.T) {
	t.Cleanup(func() { SetJSONMode(false) })
	SetJSONMode(true)

	var buf bytes.Buffer
	requireNoError(t, RenderTo(&buf, fixture{A: 1, B: "hello"}))
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
	t.Cleanup(func() { SetJSONMode(false) })
	SetJSONMode(false)

	var buf bytes.Buffer
	requireNoError(t, RenderTo(&buf, fixture{B: "hello"}))
	out := buf.String()
	if out != "hello\n" {
		t.Fatalf("expected %q, got %q", "hello\n", out)
	}
}

func TestRender_TextMode_EmptyTextProducesNoOutput(t *testing.T) {
	t.Cleanup(func() { SetJSONMode(false) })
	SetJSONMode(false)

	var buf bytes.Buffer
	requireNoError(t, RenderTo(&buf, fixture{B: ""}))
	out := buf.String()
	if out != "" {
		t.Fatalf("expected empty, got %q", out)
	}
}

func TestRenderError_MapsCjverrTypesToCodes(t *testing.T) {
	t.Cleanup(func() { SetJSONMode(false) })
	SetJSONMode(true)

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
		requireSameError(t, RenderErrorTo(&stdout, &stderr, tc.err), tc.err)
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
	t.Cleanup(func() { SetJSONMode(false) })
	SetJSONMode(true)

	var stdout, stderr bytes.Buffer
	err := &cjverr.ExitCodeError{Code: 2}
	requireSameError(t, RenderErrorTo(&stdout, &stderr, err), err)
	out := stdout.String()
	if out != "" {
		t.Fatalf("ExitCodeError should not produce envelope output, got %q", out)
	}
}

func TestRenderError_TextModeSilent(t *testing.T) {
	t.Cleanup(func() { SetJSONMode(false) })
	SetJSONMode(false)

	var stdout, stderr bytes.Buffer
	err := &cjverr.NoToolchainConfiguredError{}
	requireSameError(t, RenderErrorTo(&stdout, &stderr, err), err)
	out := stdout.String()
	if out != "" {
		t.Fatalf("text mode should not write to stdout, got %q", out)
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
