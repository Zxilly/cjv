package toolchaincmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/spf13/cobra"

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

// newTestLink returns the link command; link runs it with name and target.
func newTestLink() *cobra.Command {
	return (&link{ctx: newTestContext()}).command()
}

func runLink(t *testing.T, cmd *cobra.Command, name, target string) error {
	t.Helper()
	return cmd.RunE(cmd, []string{name, target})
}

func TestToolchainLinkCommandCreatesCustomLinkAndProxyLinks(t *testing.T) {
	cmd := newTestLink()
	home := t.TempDir()
	target := sdkDirForLink(t)
	configtest.Isolate(t, home)

	require.NoError(t, runLink(t, cmd, "my-sdk", target))
	assert.FileExists(t, filepath.Join(home, "bin", sdktools.CjvBinaryName()))
	assert.FileExists(t, filepath.Join(home, "bin", sdktools.PlatformBinaryName("cjc")))
	_, statErr := os.Lstat(filepath.Join(home, "toolchains", "my-sdk"))
	assert.NoError(t, statErr)

	require.Error(t, runLink(t, cmd, "my-sdk", target))
}

func TestToolchainLinkCommandRejectsInvalidInputs(t *testing.T) {
	cmd := newTestLink()
	home := t.TempDir()
	configtest.Isolate(t, home)

	require.Error(t, runLink(t, cmd, "lts", t.TempDir()))
	require.Error(t, runLink(t, cmd, "bad/path", t.TempDir()))
	require.Error(t, runLink(t, cmd, "my-sdk", filepath.Join(t.TempDir(), "missing")))
	// Reserved channel names are rejected before any download.
	require.Error(t, runLink(t, cmd, "lts", "https://example.invalid/sdk.zip"))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "lts"))
}

func TestToolchainLinkCommandRejectsArchiveFlagsOnDirectory(t *testing.T) {
	cmd := newTestLink()
	home := t.TempDir()
	configtest.Isolate(t, home)

	// Explicitly setting a URL/archive-only flag with a directory is rejected
	// rather than silently ignored.
	require.NoError(t, cmd.Flags().Set("no-stdx", "true"))
	require.Error(t, runLink(t, cmd, "local-sdk", sdkDirForLink(t)))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "local-sdk"))
}

func TestToolchainLinkCommandRendersArchiveResultAsJSON(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	sdkData, _ := testutil.MockSDKZip()
	src := filepath.Join(t.TempDir(), "sdk.zip")
	require.NoError(t, os.WriteFile(src, sdkData, 0o644))

	stdout, err := execute(t, "--json", "toolchain", "link", "my-sdk", src)
	require.NoError(t, err)
	var result toolchainLinkResult
	require.NoError(t, json.Unmarshal([]byte(stdout), &result), stdout)
	assert.Equal(t, "my-sdk", result.Name)
	assert.Equal(t, filepath.Join(home, "toolchains", "my-sdk"), result.Path)
	assert.FileExists(t, src, "the user's archive is kept")
}

func TestToolchainLinkCommandNormalizesNameInResult(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	stdout, err := execute(t, "--json", "toolchain", "link", " my-sdk/ ", sdkDirForLink(t))
	require.NoError(t, err)
	var result toolchainLinkResult
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	assert.Equal(t, "my-sdk", result.Name)
	assert.FileExists(t, filepath.Join(home, "toolchains", "my-sdk", "bin", sdktools.PlatformBinaryName("cjc")))
}
