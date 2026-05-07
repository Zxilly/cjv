package component

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildTarGz writes a minimal tar.gz with the given file map. Keys are relative
// paths inside the archive; values are file contents.
func buildTarGz(t *testing.T, dir, name string, files map[string]string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	require.NoError(t, err)
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for n, content := range files {
		hdr := &tar.Header{Name: n, Mode: 0o644, Size: int64(len(content))}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())
	return path
}

func buildZip(t *testing.T, dir, name string, files map[string]string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	require.NoError(t, err)
	w := zip.NewWriter(f)
	for n, content := range files {
		fw, err := w.Create(n)
		require.NoError(t, err)
		_, err = fw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	require.NoError(t, f.Close())
	return path
}

func sha256File(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestInstall_Stdx_StripsTopLevel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stdx archives are tar.gz on non-windows; on Windows the SDK ships zip — covered elsewhere")
	}

	srvDir := t.TempDir()

	// Mock GitCode endpoint serving a fake stdx tarball with one top-level
	// directory we expect Install to strip.
	tarPath := buildTarGz(t, srvDir, "stdx.tar.gz", map[string]string{
		"cangjie-stdx-linux-aarch64-1.0.5/dynamic/libfoo.so": "dynamic-bytes",
		"cangjie-stdx-linux-aarch64-1.0.5/static/libfoo.a":   "static-bytes",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if filepath.Ext(r.URL.Path) == ".sha256" {
			_, _ = w.Write([]byte(sha256File(t, tarPath)))
			return
		}
		http.ServeFile(w, r, tarPath)
	}))
	defer server.Close()

	// Override the LTS release base URL so ResolveAssetURL points at our server.
	origBase := DefaultStdxReleaseBaseURL
	stdxReleaseBaseOverride = server.URL
	defer func() { stdxReleaseBaseOverride = origBase }()

	tcDir := t.TempDir()
	docsDir := t.TempDir()
	stdxDir := t.TempDir()
	roots := Roots{TcDir: tcDir, DocsDir: docsDir, StdxDir: stdxDir}
	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}
	downloads := t.TempDir()

	require.NoError(t, Install(context.Background(), roots, tc, Stdx, "linux-arm64", downloads, false))

	assert.FileExists(t, filepath.Join(stdxDir, "dynamic", "libfoo.so"))
	assert.FileExists(t, filepath.Join(stdxDir, "static", "libfoo.a"))

	manifest, err := ReadManifest(tcDir, Stdx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"dynamic/libfoo.so", "static/libfoo.a"}, manifest)

	// Already installed → returns ComponentAlreadyInstalledError when force=false.
	err = Install(context.Background(), roots, tc, Stdx, "linux-arm64", downloads, false)
	var already *cjverr.ComponentAlreadyInstalledError
	assert.ErrorAs(t, err, &already)
}

func TestInstall_ForceDownloadFailureKeepsExistingComponent(t *testing.T) {
	t.Setenv("CJV_MAX_RETRIES", "0")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusInternalServerError)
	}))
	defer server.Close()

	origBase := stdxReleaseBaseOverride
	stdxReleaseBaseOverride = server.URL
	defer func() { stdxReleaseBaseOverride = origBase }()

	tcDir := t.TempDir()
	docsDir := t.TempDir()
	stdxDir := t.TempDir()
	roots := Roots{TcDir: tcDir, DocsDir: docsDir, StdxDir: stdxDir}
	require.NoError(t, os.MkdirAll(filepath.Join(stdxDir, "dynamic"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stdxDir, "dynamic", "libfoo.so"), []byte("old"), 0o644))
	require.NoError(t, WriteManifest(tcDir, Stdx, []string{"dynamic/libfoo.so"}))

	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}
	err := Install(context.Background(), roots, tc, Stdx, "linux-arm64", t.TempDir(), true)

	require.Error(t, err)
	assert.True(t, IsInstalled(tcDir, Stdx))
	assert.FileExists(t, filepath.Join(stdxDir, "dynamic", "libfoo.so"))
	data, readErr := os.ReadFile(filepath.Join(stdxDir, "dynamic", "libfoo.so"))
	require.NoError(t, readErr)
	assert.Equal(t, "old", string(data))
}

func TestInstall_ReleaseComponentRequiresChecksumSidecar(t *testing.T) {
	tarPath := buildTarGz(t, t.TempDir(), "docs.tar.gz", map[string]string{
		"index.html": "docs",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if filepath.Ext(r.URL.Path) == ".sha256" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, tarPath)
	}))
	defer server.Close()

	origBase := docsBundleBaseOverride
	docsBundleBaseOverride = server.URL
	defer func() { docsBundleBaseOverride = origBase }()

	tcDir := t.TempDir()
	roots := Roots{TcDir: tcDir, DocsDir: t.TempDir(), StdxDir: t.TempDir()}
	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}

	err := Install(context.Background(), roots, tc, Docs, "", t.TempDir(), false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum")
	assert.False(t, IsInstalled(tcDir, Docs))
	assert.NoFileExists(t, filepath.Join(roots.DocsDir, "main", "index.html"))
}

func TestInstall_Docs_WritesManifestAndFiles(t *testing.T) {
	tarPath := buildTarGz(t, t.TempDir(), "docs.tar.gz", map[string]string{
		"index.html":              "docs",
		"libs/std/index.html":     "std",
		"assets/searchindex.js":   "search",
		"assets/favicon.svg":      "icon",
		"dev-guide/index.html":    "guide",
		"tools/source/index.html": "tools",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if filepath.Ext(r.URL.Path) == ".sha256" {
			_, _ = w.Write([]byte(sha256File(t, tarPath)))
			return
		}
		http.ServeFile(w, r, tarPath)
	}))
	defer server.Close()

	origBase := docsBundleBaseOverride
	docsBundleBaseOverride = server.URL
	defer func() { docsBundleBaseOverride = origBase }()

	tcDir := t.TempDir()
	roots := Roots{TcDir: tcDir, DocsDir: t.TempDir(), StdxDir: t.TempDir()}
	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}

	require.NoError(t, Install(context.Background(), roots, tc, Docs, "", t.TempDir(), false))

	assert.FileExists(t, filepath.Join(roots.DocsDir, "main", "index.html"))
	assert.FileExists(t, filepath.Join(roots.DocsDir, "main", "libs", "std", "index.html"))
	manifest, err := ReadManifest(tcDir, Docs)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"index.html",
		"libs/std/index.html",
		"assets/searchindex.js",
		"assets/favicon.svg",
		"dev-guide/index.html",
		"tools/source/index.html",
	}, manifest)
}

func TestInstall_Stdx_WindowsZip(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows stdx archives use zip")
	}

	zipPath := buildZip(t, t.TempDir(), "stdx.zip", map[string]string{
		"cangjie-stdx-windows-x64-1.0.5/dynamic/foo.dll": "dynamic",
		"cangjie-stdx-windows-x64-1.0.5/static/foo.lib":  "static",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if filepath.Ext(r.URL.Path) == ".sha256" {
			_, _ = w.Write([]byte(sha256File(t, zipPath)))
			return
		}
		http.ServeFile(w, r, zipPath)
	}))
	defer server.Close()

	origBase := stdxReleaseBaseOverride
	stdxReleaseBaseOverride = server.URL
	defer func() { stdxReleaseBaseOverride = origBase }()

	tcDir := t.TempDir()
	roots := Roots{TcDir: tcDir, DocsDir: t.TempDir(), StdxDir: t.TempDir()}
	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}

	require.NoError(t, Install(context.Background(), roots, tc, Stdx, "win32-x64", t.TempDir(), false))

	assert.FileExists(t, filepath.Join(roots.StdxDir, "dynamic", "foo.dll"))
	assert.FileExists(t, filepath.Join(roots.StdxDir, "static", "foo.lib"))
}

func TestInstall_RejectsChecksumMismatch(t *testing.T) {
	t.Setenv("CJV_MAX_RETRIES", "0")
	tarPath := buildTarGz(t, t.TempDir(), "docs.tar.gz", map[string]string{
		"index.html": "docs",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if filepath.Ext(r.URL.Path) == ".sha256" {
			_, _ = w.Write([]byte(strings.Repeat("0", 64)))
			return
		}
		http.ServeFile(w, r, tarPath)
	}))
	defer server.Close()

	origBase := docsBundleBaseOverride
	docsBundleBaseOverride = server.URL
	defer func() { docsBundleBaseOverride = origBase }()

	tcDir := t.TempDir()
	roots := Roots{TcDir: tcDir, DocsDir: t.TempDir(), StdxDir: t.TempDir()}
	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}

	err := Install(context.Background(), roots, tc, Docs, "", t.TempDir(), false)

	require.Error(t, err)
	var mismatch *cjverr.ChecksumMismatchError
	assert.ErrorAs(t, err, &mismatch)
	assert.False(t, IsInstalled(tcDir, Docs))
}

func TestInstallRejectsUnknownAndUnsupportedComponents(t *testing.T) {
	roots := Roots{TcDir: t.TempDir(), DocsDir: t.TempDir(), StdxDir: t.TempDir()}
	tc := toolchain.ToolchainName{Channel: toolchain.STS, Version: "2.0.0"}

	require.Error(t, Install(context.Background(), roots, tc, Name("unknown"), "", t.TempDir(), false))

	ltsOnly := Name("lts-only")
	specs[ltsOnly] = Spec{
		Name:              ltsOnly,
		Location:          InstallLocation{Anchor: AnchorDocs, Subdir: "lts-only"},
		StripTopLevel:     false,
		SupportedChannels: []toolchain.Channel{toolchain.LTS},
	}
	t.Cleanup(func() { delete(specs, ltsOnly) })

	err := Install(context.Background(), roots, tc, ltsOnly, "", t.TempDir(), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lts-only")
}

func TestMoveStagedFilesErrorBranches(t *testing.T) {
	parentFile := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(parentFile, []byte("file"), 0o644))
	_, err := moveStagedFiles(t.TempDir(), filepath.Join(parentFile, "dest"), []string{"file.txt"})
	require.Error(t, err)

	stageDir := t.TempDir()
	destDir := t.TempDir()
	moved, err := moveStagedFiles(stageDir, destDir, []string{"missing.txt"})
	require.Error(t, err)
	assert.Empty(t, moved)
}
