package lifecycle_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for the three link forms. The shared placement pipeline (recovery,
// staging, validation, transactional swap, rollback) is covered with every
// acquisition in resolved_install_test.go; these tests cover what each
// acquisition adds: the CI bundle layout and bundled stdx for URL and local
// archives, the never-deleted source for local archives, and the in-place
// reference for directories.

// zipBytes builds an in-memory zip from name->content entries.
func zipBytes(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for n, content := range files {
		fw, err := w.Create(n)
		require.NoError(t, err)
		_, err = fw.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return buf.Bytes()
}

// hostSDKTarget returns a plausible CI SDK target name for the current OS. The
// cross-OS guard reads the binary magic, not this name, so any value works.
func hostSDKTarget() string {
	switch runtime.GOOS {
	case "windows":
		return "windows-x64"
	case "darwin":
		return "mac-aarch64"
	default:
		return "linux-x64"
	}
}

// magicForOS returns the leading bytes of an executable that targets goos
// (ELF / PE / Mach-O), so the cross-OS guard sees the intended OS.
func magicForOS(goos string) []byte {
	switch goos {
	case "linux":
		return []byte{0x7f, 'E', 'L', 'F'}
	case "windows":
		return []byte{'M', 'Z', 0x90, 0x00}
	default: // darwin
		return []byte{0xCF, 0xFA, 0xED, 0xFE}
	}
}

// foreignMagic returns a binary magic for a GOOS different from the host.
func foreignMagic() []byte {
	if runtime.GOOS == "linux" {
		return magicForOS("windows")
	}
	return magicForOS("linux")
}

// sdkInnerArchive returns a zip whose single top-level cangjie/ dir holds a cjc
// binary with the host OS's magic, mirroring the CI SDK archive layout.
func sdkInnerArchive(t *testing.T) []byte {
	t.Helper()
	return zipBytes(t, map[string][]byte{
		"cangjie/bin/" + sdktools.PlatformBinaryName("cjc"): magicForOS(runtime.GOOS),
	})
}

// stdxInnerArchiveWith returns a stdx zip whose single top-level dir holds
// dynamic/<lib> and static/<lib>, so tests can vary the file set across installs.
func stdxInnerArchiveWith(t *testing.T, lib string) []byte {
	t.Helper()
	return zipBytes(t, map[string][]byte{
		"any_cjnative/dynamic/" + lib: []byte("d"),
		"any_cjnative/static/" + lib:  []byte("s"),
	})
}

// ciBundle returns the outer CI archive: an inner SDK archive plus, when stdx
// is non-nil, an inner stdx archive.
func ciBundle(t *testing.T, sdk, stdx []byte) []byte {
	t.Helper()
	files := map[string][]byte{"cangjie-sdk-" + hostSDKTarget() + "-1.0.0.zip": sdk}
	if stdx != nil {
		files["cangjie-stdx-"+hostSDKTarget()+"-1.0.0.0.0.1.zip"] = stdx
	}
	return zipBytes(t, files)
}

// serveBytes serves the given bytes over HTTP and returns the URL.
func serveBytes(t *testing.T, body []byte) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// writeArchive writes body to a fresh temp file and returns its path.
func writeArchive(t *testing.T, body []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "sdk.zip")
	require.NoError(t, os.WriteFile(p, body, 0o644))
	return p
}

func linkHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvDistServer, "")
	return home
}

func linkURL(t *testing.T, name string, body []byte, sha256 string, force, noStdx bool) error {
	t.Helper()
	return lifecycle.InstallToolchainFromURL(context.Background(), name, serveBytes(t, body), sha256, force, noStdx, lifecycle.Options{})
}

func TestLinkURL_SDKAndStdx(t *testing.T) {
	home := linkHome(t)
	require.NoError(t, linkURL(t, "my-sdk", ciBundle(t, sdkInnerArchive(t), stdxInnerArchiveWith(t, "libfoo")), "", false, false))

	// Toolchain is a real (owned) directory, not a symlink.
	info, err := os.Lstat(filepath.Join(home, "toolchains", "my-sdk"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Zero(t, info.Mode()&os.ModeSymlink, "URL toolchain must be a real directory, not a symlink")
	assert.FileExists(t, filepath.Join(home, "toolchains", "my-sdk", "bin", sdktools.PlatformBinaryName("cjc")))

	// Proxy links created.
	assert.FileExists(t, filepath.Join(home, "bin", sdktools.CjvBinaryName()))
	assert.FileExists(t, filepath.Join(home, "bin", sdktools.PlatformBinaryName("cjc")))

	// Bundled stdx installed and manifest written.
	assert.DirExists(t, filepath.Join(home, "stdx", "my-sdk", "dynamic"))
	assert.DirExists(t, filepath.Join(home, "stdx", "my-sdk", "static"))
	assert.FileExists(t, filepath.Join(home, "toolchains", "my-sdk", ".cjv", "components", "manifest-stdx"))

	// The default toolchain is never changed by a link.
	_, settings, err := config.LoadDefaultSettings()
	require.NoError(t, err)
	assert.Empty(t, settings.DefaultToolchain, "URL install must not change the default toolchain")

	// The downloads area keeps no residue on success.
	entries, err := os.ReadDir(filepath.Join(home, "downloads"))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLinkURL_SDKOnlyAndNoStdx(t *testing.T) {
	home := linkHome(t)
	require.NoError(t, linkURL(t, "my-sdk", ciBundle(t, sdkInnerArchive(t), nil), "", false, false))
	assert.FileExists(t, filepath.Join(home, "toolchains", "my-sdk", "bin", sdktools.PlatformBinaryName("cjc")))
	assert.NoDirExists(t, filepath.Join(home, "stdx", "my-sdk"))

	require.NoError(t, linkURL(t, "no-stdx", ciBundle(t, sdkInnerArchive(t), stdxInnerArchiveWith(t, "libfoo")), "", false, true))
	assert.FileExists(t, filepath.Join(home, "toolchains", "no-stdx", "bin", sdktools.PlatformBinaryName("cjc")))
	assert.NoDirExists(t, filepath.Join(home, "stdx", "no-stdx"), "--no-stdx must skip the bundled stdx")
}

func TestLinkURL_BareArchiveFallback(t *testing.T) {
	home := linkHome(t)
	// URL points directly at a bare SDK archive (no inner cangjie-sdk-* wrapper):
	// the served zip itself has the single top-level cangjie/ dir.
	require.NoError(t, linkURL(t, "bare", sdkInnerArchive(t), "", false, false))
	assert.FileExists(t, filepath.Join(home, "toolchains", "bare", "bin", sdktools.PlatformBinaryName("cjc")))
	assert.NoDirExists(t, filepath.Join(home, "stdx", "bare"))
}

func TestLinkURL_NoSDKArchiveRejected(t *testing.T) {
	home := linkHome(t)
	outer := zipBytes(t, map[string][]byte{"README": []byte("nothing"), "a/x": []byte("a"), "b/y": []byte("b")})
	require.Error(t, linkURL(t, "empty", outer, "", false, false))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "empty"))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "empty.staging"))
}

func TestLinkURL_SHA256Mismatch(t *testing.T) {
	home := linkHome(t)
	const wrong = "0000000000000000000000000000000000000000000000000000000000000000"
	require.Error(t, linkURL(t, "my-sdk", ciBundle(t, sdkInnerArchive(t), nil), wrong, false, false))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "my-sdk"))
}

func TestLinkURL_ForceReinstall(t *testing.T) {
	home := linkHome(t)
	url := serveBytes(t, ciBundle(t, sdkInnerArchive(t), nil))
	link := func(force bool) error {
		return lifecycle.InstallToolchainFromURL(context.Background(), "my-sdk", url, "", force, false, lifecycle.Options{})
	}
	require.NoError(t, link(false))
	err := link(false)
	var already *cjverr.ToolchainAlreadyInstalledError
	require.ErrorAs(t, err, &already, "second install without --force must fail")
	require.NoError(t, link(true), "second install with --force must succeed")
	assert.FileExists(t, filepath.Join(home, "toolchains", "my-sdk", "bin", sdktools.PlatformBinaryName("cjc")))
}

func TestLinkURL_CrossOSRejected(t *testing.T) {
	home := linkHome(t)
	foreign := zipBytes(t, map[string][]byte{"cangjie/bin/cjc": foreignMagic()})
	tests := map[string][]byte{
		// The cjc binary targets a different OS than the host (detected via magic).
		"nested": ciBundle(t, foreign, nil),
		// A bare SDK archive whose cjc targets another OS is rejected the same way.
		"bare": foreign,
	}
	if runtime.GOOS != "darwin" {
		// A macOS SDK ships a Mach-O cjc; it must be rejected on a non-darwin host.
		tests["mac"] = zipBytes(t, map[string][]byte{
			"cangjie-sdk-mac-aarch64-1.0.0.zip": zipBytes(t, map[string][]byte{"cangjie/bin/cjc": magicForOS("darwin")}),
		})
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			err := linkURL(t, name, body, "", false, false)
			require.Error(t, err)
			assert.Contains(t, err.Error(), runtime.GOOS)
			assert.NoDirExists(t, filepath.Join(home, "toolchains", name))
		})
	}
}

func TestLinkURL_StdxMissingDirsRollsBack(t *testing.T) {
	home := linkHome(t)
	// A stdx archive whose stripped contents do NOT yield dynamic/ + static/.
	badStdx := zipBytes(t, map[string][]byte{"wrong_layout/foo": []byte("x")})
	require.Error(t, linkURL(t, "my-sdk", ciBundle(t, sdkInnerArchive(t), badStdx), "", false, false))

	// SDK is kept (committed before stdx), but the half-written stdx is rolled back.
	assert.FileExists(t, filepath.Join(home, "toolchains", "my-sdk", "bin", sdktools.PlatformBinaryName("cjc")))
	assert.NoDirExists(t, filepath.Join(home, "stdx", "my-sdk"))
	assert.NoFileExists(t, filepath.Join(home, "toolchains", "my-sdk", ".cjv", "components", "manifest-stdx"))
}

func TestLinkURL_ForceReinstallWithStdx(t *testing.T) {
	home := linkHome(t)
	require.NoError(t, linkURL(t, "my-sdk", ciBundle(t, sdkInnerArchive(t), stdxInnerArchiveWith(t, "libold")), "", false, false))
	assert.FileExists(t, filepath.Join(home, "stdx", "my-sdk", "dynamic", "libold"))

	// Force-reinstall with a stdx bundle whose file set changed (libold -> libnew).
	require.NoError(t, linkURL(t, "my-sdk", ciBundle(t, sdkInnerArchive(t), stdxInnerArchiveWith(t, "libnew")), "", true, false))

	assert.FileExists(t, filepath.Join(home, "toolchains", "my-sdk", "bin", sdktools.PlatformBinaryName("cjc")))
	assert.FileExists(t, filepath.Join(home, "toolchains", "my-sdk", ".cjv", "components", "manifest-stdx"))
	assert.FileExists(t, filepath.Join(home, "stdx", "my-sdk", "dynamic", "libnew"))
	// The dropped library must not be left orphaned on the stdx search path.
	assert.NoFileExists(t, filepath.Join(home, "stdx", "my-sdk", "dynamic", "libold"))
	assert.NoFileExists(t, filepath.Join(home, "stdx", "my-sdk", "static", "libold"))
}

func TestLinkZip_MaterializesOwnedToolchainAndKeepsSource(t *testing.T) {
	home := linkHome(t)
	src := writeArchive(t, ciBundle(t, sdkInnerArchive(t), stdxInnerArchiveWith(t, "libfoo")))
	require.NoError(t, lifecycle.InstallToolchainFromZip(context.Background(), "my-sdk", src, "", false, false, lifecycle.Options{}))

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

	// A second link without force fails; with force it succeeds.
	err = lifecycle.InstallToolchainFromZip(context.Background(), "my-sdk", src, "", false, false, lifecycle.Options{})
	var already *cjverr.ToolchainAlreadyInstalledError
	require.ErrorAs(t, err, &already)
	require.NoError(t, lifecycle.InstallToolchainFromZip(context.Background(), "my-sdk", src, "", true, false, lifecycle.Options{}))
	assert.FileExists(t, src)
}

func TestLinkZip_ScratchesUnderDownloadsNotBesideArchive(t *testing.T) {
	home := linkHome(t)
	src := writeArchive(t, ciBundle(t, sdkInnerArchive(t), stdxInnerArchiveWith(t, "libfoo")))
	srcDir := filepath.Dir(src)
	// Where the OS enforces it, make the archive's directory read-only so any
	// attempt to scratch beside it fails loudly instead of leaving residue.
	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		require.NoError(t, os.Chmod(srcDir, 0o555))
		t.Cleanup(func() { _ = os.Chmod(srcDir, 0o755) })
	}

	require.NoError(t, lifecycle.InstallToolchainFromZip(context.Background(), "my-sdk", src, "", false, false, lifecycle.Options{}))
	assert.FileExists(t, filepath.Join(home, "toolchains", "my-sdk", "bin", sdktools.PlatformBinaryName("cjc")))

	// Nothing was extracted next to the user's archive, and no scratch dir
	// survives under downloads either.
	entries, err := os.ReadDir(srcDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	leftovers, err := filepath.Glob(filepath.Join(home, "downloads", ".cjv-link-*"))
	require.NoError(t, err)
	assert.Empty(t, leftovers)
}

func TestLinkZip_SHA256MismatchKeepsArchive(t *testing.T) {
	home := linkHome(t)
	src := writeArchive(t, ciBundle(t, sdkInnerArchive(t), nil))
	const wrong = "0000000000000000000000000000000000000000000000000000000000000000"
	require.Error(t, lifecycle.InstallToolchainFromZip(context.Background(), "my-sdk", src, wrong, false, false, lifecycle.Options{}))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "my-sdk"))
	assert.FileExists(t, src, "a rejected archive must not be deleted")
}

func sdkDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cjcPath := filepath.Join(dir, "bin", sdktools.PlatformBinaryName("cjc"))
	require.NoError(t, os.MkdirAll(filepath.Dir(cjcPath), 0o755))
	require.NoError(t, os.WriteFile(cjcPath, []byte("stub"), 0o755))
	return dir
}

func TestLinkToolchainDir_ReferencesInPlaceAndFinalizes(t *testing.T) {
	home := linkHome(t)
	target := sdkDir(t)
	require.NoError(t, lifecycle.LinkToolchainDir("local-sdk", target))

	// Finalized like an installed toolchain.
	assert.FileExists(t, filepath.Join(home, "bin", sdktools.CjvBinaryName()))
	assert.FileExists(t, filepath.Join(home, "bin", sdktools.PlatformBinaryName("cjc")))

	// A directory is referenced in place (symlink/junction), not copied: a file
	// added to the source afterwards is visible through the link. This holds for
	// both symlinks and Windows junctions, unlike a ModeSymlink bit check.
	require.NoError(t, os.WriteFile(filepath.Join(target, "marker.txt"), []byte("x"), 0o644))
	assert.FileExists(t, filepath.Join(home, "toolchains", "local-sdk", "marker.txt"))

	err := lifecycle.LinkToolchainDir("local-sdk", target)
	var already *cjverr.ToolchainAlreadyInstalledError
	require.ErrorAs(t, err, &already, "an existing toolchain is never replaced by a link")
}

func TestLinkToolchainDir_RejectsDirectoryWithoutCompiler(t *testing.T) {
	home := linkHome(t)
	err := lifecycle.LinkToolchainDir("not-sdk", t.TempDir())
	require.Error(t, err)
	assert.NoFileExists(t, filepath.Join(home, "toolchains", "not-sdk"))
}

func TestLinkToolchainDir_RemovesLinkWhenFinalizeFails(t *testing.T) {
	home := linkHome(t)
	finalizeErr := errors.New("proxy refresh failed")
	lifecycle.SetAfterFinalizeHook(t, func() error { return finalizeErr })
	err := lifecycle.LinkToolchainDir("local-sdk", sdkDir(t))
	require.ErrorIs(t, err, finalizeErr)
	_, statErr := os.Lstat(filepath.Join(home, "toolchains", "local-sdk"))
	assert.True(t, errors.Is(statErr, os.ErrNotExist), "a link whose finalization failed must not remain")
}
