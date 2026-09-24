package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests cover the seams that are specific to linking a *local archive*
// (file vs directory detection, never deleting the user's file, verifying it,
// and the flags now accepted on a local path). The shared extraction/stdx core
// is exercised by the URL tests and is not re-tested here.

// writeArchive writes body to a fresh temp file and returns its path.
func writeArchive(t *testing.T, body []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "sdk.zip")
	require.NoError(t, os.WriteFile(p, body, 0o644))
	return p
}

// linkPath runs `toolchain link <name> <path>` against a local path.
func (app *application) linkPath(t *testing.T, name, path string) error {
	t.Helper()
	return app.toolchainLinkCmd.RunE(app.toolchainLinkCmd, []string{name, path})
}

func TestToolchainLinkZip_MaterializesOwnedToolchainAndKeepsSource(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)

	outer := zipBytes(t, map[string][]byte{
		"cangjie-sdk-" + hostSDKTarget() + "-1.0.0.zip":        sdkInnerArchive(t),
		"cangjie-stdx-" + hostSDKTarget() + "-1.0.0.0.0.1.zip": stdxInnerArchive(t),
	})
	src := writeArchive(t, outer)

	require.NoError(t, app.linkPath(t, "my-sdk", src))

	// The toolchain is a real (owned) directory, not a symlink to the archive.
	info, err := os.Lstat(filepath.Join(home, "toolchains", "my-sdk"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Zero(t, info.Mode()&os.ModeSymlink, "archive link must materialize a real directory")
	assert.FileExists(t, filepath.Join(home, "toolchains", "my-sdk", "bin", sdktools.PlatformBinaryName("cjc")))

	// Proxy links created and bundled stdx installed, just like the URL path.
	assert.FileExists(t, filepath.Join(home, "bin", sdktools.PlatformBinaryName("cjc")))
	assert.DirExists(t, filepath.Join(home, "stdx", "my-sdk", "dynamic"))

	// The user's source archive must be left untouched (never moved or deleted).
	assert.FileExists(t, src)
}

func TestToolchainLinkZip_DirectoryStillSymlinks(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	target := t.TempDir()
	config.IsolateForTest(t, home)

	cjcPath := filepath.Join(target, "bin", sdktools.PlatformBinaryName("cjc"))
	require.NoError(t, os.MkdirAll(filepath.Dir(cjcPath), 0o755))
	require.NoError(t, os.WriteFile(cjcPath, []byte("stub"), 0o755))

	require.NoError(t, app.linkPath(t, "local-sdk", target))

	// A directory is referenced in place (symlink/junction), not copied: a file
	// added to the source afterwards is visible through the link. This holds for
	// both symlinks and Windows junctions, unlike a ModeSymlink bit check.
	require.NoError(t, os.WriteFile(filepath.Join(target, "marker.txt"), []byte("x"), 0o644))
	assert.FileExists(t, filepath.Join(home, "toolchains", "local-sdk", "marker.txt"))
}

func TestToolchainLinkZip_SHA256Mismatch(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)

	app.linkSHA256 = "0000000000000000000000000000000000000000000000000000000000000000"

	outer := zipBytes(t, map[string][]byte{
		"cangjie-sdk-" + hostSDKTarget() + "-1.0.0.zip": sdkInnerArchive(t),
	})
	src := writeArchive(t, outer)

	require.Error(t, app.linkPath(t, "my-sdk", src))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "my-sdk"))
	assert.FileExists(t, src, "a rejected archive must not be deleted")
}

func TestToolchainLinkZip_ForceReinstall(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)

	outer := zipBytes(t, map[string][]byte{
		"cangjie-sdk-" + hostSDKTarget() + "-1.0.0.zip": sdkInnerArchive(t),
	})
	src := writeArchive(t, outer)

	require.NoError(t, app.linkPath(t, "my-sdk", src))
	require.Error(t, app.linkPath(t, "my-sdk", src), "second link without --force must fail")

	app.linkForce = true
	require.NoError(t, app.linkPath(t, "my-sdk", src), "second link with --force must succeed")
}
