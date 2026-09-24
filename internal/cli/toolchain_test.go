package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for `toolchain link`: argument validation, the choice between the
// URL, archive and directory forms, and result rendering. What each form
// does to CJV_HOME is lifecycle behaviour, tested in internal/lifecycle.

func sdkDirForLink(t *testing.T) string {
	t.Helper()
	target := t.TempDir()
	cjcPath := filepath.Join(target, "bin", sdktools.PlatformBinaryName("cjc"))
	require.NoError(t, os.MkdirAll(filepath.Dir(cjcPath), 0o755))
	require.NoError(t, os.WriteFile(cjcPath, []byte("stub"), 0o755))
	return target
}

func (app *application) link(t *testing.T, name, target string) error {
	t.Helper()
	return app.toolchainLinkCmd.RunE(app.toolchainLinkCmd, []string{name, target})
}

func TestToolchainLinkCommandCreatesCustomLinkAndProxyLinks(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	target := sdkDirForLink(t)
	config.IsolateForTest(t, home)

	require.NoError(t, app.link(t, "my-sdk", target))
	assert.FileExists(t, filepath.Join(home, "bin", sdktools.CjvBinaryName()))
	assert.FileExists(t, filepath.Join(home, "bin", sdktools.PlatformBinaryName("cjc")))
	_, statErr := os.Lstat(filepath.Join(home, "toolchains", "my-sdk"))
	assert.NoError(t, statErr)

	require.Error(t, app.link(t, "my-sdk", target))
}

func TestToolchainLinkCommandRejectsInvalidInputs(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)

	require.Error(t, app.link(t, "lts", t.TempDir()))
	require.Error(t, app.link(t, "bad/path", t.TempDir()))
	require.Error(t, app.link(t, "my-sdk", filepath.Join(t.TempDir(), "missing")))
	// Reserved channel names are rejected before any download.
	require.Error(t, app.link(t, "lts", "https://example.invalid/sdk.zip"))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "lts"))
}

func TestToolchainLinkCommandRejectsArchiveFlagsOnDirectory(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)

	// Explicitly setting a URL/archive-only flag with a directory is rejected
	// rather than silently ignored.
	require.NoError(t, app.toolchainLinkCmd.Flags().Set("no-stdx", "true"))
	require.Error(t, app.link(t, "local-sdk", sdkDirForLink(t)))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "local-sdk"))
}

func TestToolchainLinkCommandRendersArchiveResultAsJSON(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	sdkData, _ := testutil.MockSDKZip()
	src := filepath.Join(t.TempDir(), "sdk.zip")
	require.NoError(t, os.WriteFile(src, sdkData, 0o644))

	stdout, err := executeWithOutput(t, app, []string{"--json", "toolchain", "link", "my-sdk", src})
	require.NoError(t, err)
	var result toolchainLinkResult
	require.NoError(t, json.Unmarshal([]byte(stdout), &result), stdout)
	assert.Equal(t, "my-sdk", result.Name)
	assert.Equal(t, filepath.Join(home, "toolchains", "my-sdk"), result.Path)
	assert.FileExists(t, src, "the user's archive is kept")
}
