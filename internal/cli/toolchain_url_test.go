package cli

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// foreignOS returns a GOOS different from the host plus a matching binary magic.
func foreignOS() (string, []byte) {
	if runtime.GOOS == "linux" {
		return "windows", magicForOS("windows")
	}
	return "linux", magicForOS("linux")
}

// sdkInnerArchive returns a zip whose single top-level cangjie/ dir holds a cjc
// binary with the host OS's magic, mirroring the CI SDK archive layout.
func sdkInnerArchive(t *testing.T) []byte {
	t.Helper()
	return zipBytes(t, map[string][]byte{
		"cangjie/bin/" + proxy.PlatformBinaryName("cjc"): magicForOS(runtime.GOOS),
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

// stdxInnerArchive returns a stdx zip mirroring the CI stdx archive layout.
func stdxInnerArchive(t *testing.T) []byte {
	t.Helper()
	return stdxInnerArchiveWith(t, "libfoo")
}

// serveBytes serves the given bytes over HTTP and returns the URL.
func serveBytes(t *testing.T, body []byte) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func resetLinkFlags(t *testing.T) {
	t.Helper()
	linkSHA256 = ""
	linkForce = false
	linkNoStdx = false
	for _, n := range []string{"sha256", "force", "no-stdx"} {
		if f := toolchainLinkCmd.Flags().Lookup(n); f != nil {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		}
	}
}

func linkURL(t *testing.T, name, url string) error {
	t.Helper()
	return toolchainLinkCmd.RunE(toolchainLinkCmd, []string{name, url})
}

func TestToolchainLinkURL_SDKAndStdx(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	resetLinkFlags(t)

	outer := zipBytes(t, map[string][]byte{
		"cangjie-sdk-" + hostSDKTarget() + "-1.0.0.zip":        sdkInnerArchive(t),
		"cangjie-stdx-" + hostSDKTarget() + "-1.0.0.0.0.1.zip": stdxInnerArchive(t),
	})
	url := serveBytes(t, outer)

	require.NoError(t, linkURL(t, "my-sdk", url))

	// Toolchain is a real (owned) directory, not a symlink.
	info, err := os.Lstat(filepath.Join(home, "toolchains", "my-sdk"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Zero(t, info.Mode()&os.ModeSymlink, "URL toolchain must be a real directory, not a symlink")
	assert.FileExists(t, filepath.Join(home, "toolchains", "my-sdk", "bin", proxy.PlatformBinaryName("cjc")))

	// Proxy links created.
	assert.FileExists(t, filepath.Join(home, "bin", proxy.CjvBinaryName()))
	assert.FileExists(t, filepath.Join(home, "bin", proxy.PlatformBinaryName("cjc")))

	// Bundled stdx installed and manifest written.
	assert.DirExists(t, filepath.Join(home, "stdx", "my-sdk", "dynamic"))
	assert.DirExists(t, filepath.Join(home, "stdx", "my-sdk", "static"))
	assert.FileExists(t, filepath.Join(home, "toolchains", "my-sdk", ".cjv", "components", "manifest-stdx"))
}

func TestToolchainLinkURL_SDKOnly(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	resetLinkFlags(t)

	outer := zipBytes(t, map[string][]byte{
		"cangjie-sdk-" + hostSDKTarget() + "-1.0.0.zip": sdkInnerArchive(t),
	})
	require.NoError(t, linkURL(t, "my-sdk", serveBytes(t, outer)))

	assert.FileExists(t, filepath.Join(home, "toolchains", "my-sdk", "bin", proxy.PlatformBinaryName("cjc")))
	assert.NoDirExists(t, filepath.Join(home, "stdx", "my-sdk"))
}

func TestToolchainLinkURL_BareArchiveFallback(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	resetLinkFlags(t)

	// URL points directly at a bare SDK archive (no inner cangjie-sdk-* wrapper):
	// the served zip itself has the single top-level cangjie/ dir.
	bare := zipBytes(t, map[string][]byte{
		"cangjie/bin/" + proxy.PlatformBinaryName("cjc"): magicForOS(runtime.GOOS),
	})
	require.NoError(t, linkURL(t, "bare", serveBytes(t, bare)))

	assert.FileExists(t, filepath.Join(home, "toolchains", "bare", "bin", proxy.PlatformBinaryName("cjc")))
	assert.NoDirExists(t, filepath.Join(home, "stdx", "bare"))
}

func TestToolchainLinkURL_NoStdxFlag(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	resetLinkFlags(t)
	linkNoStdx = true

	outer := zipBytes(t, map[string][]byte{
		"cangjie-sdk-" + hostSDKTarget() + "-1.0.0.zip":        sdkInnerArchive(t),
		"cangjie-stdx-" + hostSDKTarget() + "-1.0.0.0.0.1.zip": stdxInnerArchive(t),
	})
	require.NoError(t, linkURL(t, "my-sdk", serveBytes(t, outer)))

	assert.FileExists(t, filepath.Join(home, "toolchains", "my-sdk", "bin", proxy.PlatformBinaryName("cjc")))
	assert.NoDirExists(t, filepath.Join(home, "stdx", "my-sdk"), "--no-stdx must skip the bundled stdx")
}

func TestToolchainLinkURL_SHA256Mismatch(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	resetLinkFlags(t)
	linkSHA256 = "0000000000000000000000000000000000000000000000000000000000000000"

	outer := zipBytes(t, map[string][]byte{
		"cangjie-sdk-" + hostSDKTarget() + "-1.0.0.zip": sdkInnerArchive(t),
	})
	require.Error(t, linkURL(t, "my-sdk", serveBytes(t, outer)))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "my-sdk"))
}

func TestToolchainLinkURL_ForceReinstall(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	resetLinkFlags(t)

	outer := zipBytes(t, map[string][]byte{
		"cangjie-sdk-" + hostSDKTarget() + "-1.0.0.zip": sdkInnerArchive(t),
	})
	url := serveBytes(t, outer)

	require.NoError(t, linkURL(t, "my-sdk", url))
	require.Error(t, linkURL(t, "my-sdk", url), "second install without --force must fail")

	linkForce = true
	require.NoError(t, linkURL(t, "my-sdk", url), "second install with --force must succeed")
}

func TestToolchainLinkURL_ReservedName(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	resetLinkFlags(t)

	// Reserved channel name is rejected before any download.
	require.Error(t, linkURL(t, "lts", "https://example.invalid/sdk.zip"))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "lts"))
}

func TestToolchainLinkURL_CrossOSRejected(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	resetLinkFlags(t)

	// The cjc binary targets a different OS than the host (detected via magic).
	_, foreignMagic := foreignOS()
	sdk := zipBytes(t, map[string][]byte{"cangjie/bin/cjc": foreignMagic})
	outer := zipBytes(t, map[string][]byte{
		"cangjie-sdk-" + hostSDKTarget() + "-1.0.0.zip": sdk,
	})
	require.Error(t, linkURL(t, "cross", serveBytes(t, outer)))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "cross"))
}

func TestToolchainLinkURL_BareArchiveCrossOSRejected(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	resetLinkFlags(t)

	// A bare SDK archive (no inner cangjie-sdk-*) whose cjc targets another OS
	// must still be rejected — the magic check covers the bare path too.
	_, foreignMagic := foreignOS()
	bare := zipBytes(t, map[string][]byte{"cangjie/bin/cjc": foreignMagic})
	require.Error(t, linkURL(t, "barecross", serveBytes(t, bare)))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "barecross"))
}

func TestToolchainLinkURL_CrossOSRejectedMacBinary(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("a Mach-O binary is native on darwin")
	}
	home := t.TempDir()
	config.IsolateForTest(t, home)
	resetLinkFlags(t)

	// A macOS SDK ships a Mach-O cjc; it must be rejected on a non-darwin host.
	sdk := zipBytes(t, map[string][]byte{"cangjie/bin/cjc": magicForOS("darwin")})
	outer := zipBytes(t, map[string][]byte{
		"cangjie-sdk-mac-aarch64-1.0.0.zip": sdk,
	})
	require.Error(t, linkURL(t, "mac", serveBytes(t, outer)))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "mac"))
}

func TestToolchainLinkURL_DefaultToolchainUnchanged(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	resetLinkFlags(t)

	outer := zipBytes(t, map[string][]byte{
		"cangjie-sdk-" + hostSDKTarget() + "-1.0.0.zip": sdkInnerArchive(t),
	})
	require.NoError(t, linkURL(t, "my-sdk", serveBytes(t, outer)))

	_, settings, err := lifecycle.LoadSettings()
	require.NoError(t, err)
	assert.Empty(t, settings.DefaultToolchain, "URL install must not change the default toolchain")
}

func TestToolchainLinkURL_StdxMissingDirsRollsBack(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	resetLinkFlags(t)

	// A stdx archive whose stripped contents do NOT yield dynamic/ + static/.
	badStdx := zipBytes(t, map[string][]byte{"wrong_layout/foo": []byte("x")})
	outer := zipBytes(t, map[string][]byte{
		"cangjie-sdk-" + hostSDKTarget() + "-1.0.0.zip":        sdkInnerArchive(t),
		"cangjie-stdx-" + hostSDKTarget() + "-1.0.0.0.0.1.zip": badStdx,
	})
	require.Error(t, linkURL(t, "my-sdk", serveBytes(t, outer)))

	// SDK is kept (committed before stdx), but the half-written stdx is rolled back.
	assert.FileExists(t, filepath.Join(home, "toolchains", "my-sdk", "bin", proxy.PlatformBinaryName("cjc")))
	assert.NoDirExists(t, filepath.Join(home, "stdx", "my-sdk"))
	assert.NoFileExists(t, filepath.Join(home, "toolchains", "my-sdk", ".cjv", "components", "manifest-stdx"))
}

func TestToolchainLinkURL_ForceReinstallWithStdx(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	resetLinkFlags(t)

	tgt := hostSDKTarget()
	first := zipBytes(t, map[string][]byte{
		"cangjie-sdk-" + tgt + "-1.0.0.zip":        sdkInnerArchive(t),
		"cangjie-stdx-" + tgt + "-1.0.0.0.0.1.zip": stdxInnerArchiveWith(t, "libold"),
	})
	require.NoError(t, linkURL(t, "my-sdk", serveBytes(t, first)))
	assert.FileExists(t, filepath.Join(home, "stdx", "my-sdk", "dynamic", "libold"))

	// Force-reinstall with a stdx bundle whose file set changed (libold -> libnew).
	linkForce = true
	second := zipBytes(t, map[string][]byte{
		"cangjie-sdk-" + tgt + "-1.1.0.zip":        sdkInnerArchive(t),
		"cangjie-stdx-" + tgt + "-1.1.0.0.0.1.zip": stdxInnerArchiveWith(t, "libnew"),
	})
	require.NoError(t, linkURL(t, "my-sdk", serveBytes(t, second)))

	assert.FileExists(t, filepath.Join(home, "toolchains", "my-sdk", "bin", proxy.PlatformBinaryName("cjc")))
	assert.FileExists(t, filepath.Join(home, "toolchains", "my-sdk", ".cjv", "components", "manifest-stdx"))
	assert.FileExists(t, filepath.Join(home, "stdx", "my-sdk", "dynamic", "libnew"))
	// The dropped library must not be left orphaned on the stdx search path.
	assert.NoFileExists(t, filepath.Join(home, "stdx", "my-sdk", "dynamic", "libold"))
	assert.NoFileExists(t, filepath.Join(home, "stdx", "my-sdk", "static", "libold"))
}

func TestToolchainLinkURL_FlagOnLocalPathRejected(t *testing.T) {
	home := t.TempDir()
	target := t.TempDir()
	config.IsolateForTest(t, home)
	resetLinkFlags(t)
	t.Cleanup(func() { resetLinkFlags(t) })

	cjcPath := filepath.Join(target, "bin", proxy.PlatformBinaryName("cjc"))
	require.NoError(t, os.MkdirAll(filepath.Dir(cjcPath), 0o755))
	require.NoError(t, os.WriteFile(cjcPath, []byte("stub"), 0o755))

	// Explicitly setting a URL-only flag with a local path is rejected.
	require.NoError(t, toolchainLinkCmd.Flags().Set("no-stdx", "true"))
	require.Error(t, linkURL(t, "local-sdk", target))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "local-sdk"))
}
