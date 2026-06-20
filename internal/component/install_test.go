package component

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/dist"
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

func TestInstall_Stdx_StripsTopLevel(t *testing.T) {
	srvDir := t.TempDir()

	// Mock GitCode endpoint serving a fake stdx zip with one top-level
	// directory we expect Install to strip.
	zipPath := buildZip(t, srvDir, "stdx.zip", map[string]string{
		"cangjie-stdx-linux-aarch64-1.0.5.1/dynamic/libfoo.so": "dynamic-bytes",
		"cangjie-stdx-linux-aarch64-1.0.5.1/static/libfoo.a":   "static-bytes",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
	}))
	defer server.Close()

	tcDir := t.TempDir()
	docsDir := t.TempDir()
	stdxDir := t.TempDir()
	roots := Roots{TcDir: tcDir, DocsDir: docsDir, StdxDir: stdxDir}
	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}
	downloads := t.TempDir()
	mf := stdxComponentManifest(t, toolchain.LTS, "1.0.5", "linux-arm64", server.URL+"/stdx.zip")

	require.NoError(t, Install(context.Background(), roots, tc, Stdx, "linux-arm64", downloads, false, mf))

	assert.FileExists(t, filepath.Join(stdxDir, "dynamic", "libfoo.so"))
	assert.FileExists(t, filepath.Join(stdxDir, "static", "libfoo.a"))

	manifest, err := ReadManifest(tcDir, Stdx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"dynamic/libfoo.so", "static/libfoo.a"}, manifest)

	// Already installed → returns ComponentAlreadyInstalledError when force=false.
	err = Install(context.Background(), roots, tc, Stdx, "linux-arm64", downloads, false, mf)
	var already *cjverr.ComponentAlreadyInstalledError
	assert.ErrorAs(t, err, &already)
}

// stdxComponentManifest builds a manifest whose stdx link for tuple's archive
// platform points at url. tuple is resolved with the same mapping ResolveAssetURL
// uses, so callers pass the SDK tuple (e.g. "linux-arm64") and the manifest is
// keyed by the matching stdx platform token (e.g. "linux-aarch64").
func stdxComponentManifest(t *testing.T, channel toolchain.Channel, version, tuple, url string) *dist.Manifest {
	t.Helper()
	platform, err := stdxPlatform(tuple)
	require.NoError(t, err)
	return manifestWithComponents(channel, version, dist.ComponentSet{
		Stdx: map[string]dist.ComponentInfo{platform: {URL: url}},
	})
}

func TestInstall_ForceDownloadFailureKeepsExistingComponent(t *testing.T) {
	t.Setenv("CJV_MAX_RETRIES", "0")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusInternalServerError)
	}))
	defer server.Close()

	tcDir := t.TempDir()
	docsDir := t.TempDir()
	stdxDir := t.TempDir()
	roots := Roots{TcDir: tcDir, DocsDir: docsDir, StdxDir: stdxDir}
	require.NoError(t, os.MkdirAll(filepath.Join(stdxDir, "dynamic"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stdxDir, "dynamic", "libfoo.so"), []byte("old"), 0o644))
	require.NoError(t, WriteManifest(tcDir, Stdx, []string{"dynamic/libfoo.so"}))

	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}
	mf := stdxComponentManifest(t, toolchain.LTS, "1.0.5", "linux-arm64", server.URL+"/x")
	err := Install(context.Background(), roots, tc, Stdx, "linux-arm64", t.TempDir(), true, mf)

	require.Error(t, err)
	assert.True(t, IsInstalled(tcDir, Stdx))
	assert.FileExists(t, filepath.Join(stdxDir, "dynamic", "libfoo.so"))
	data, readErr := os.ReadFile(filepath.Join(stdxDir, "dynamic", "libfoo.so"))
	require.NoError(t, readErr)
	assert.Equal(t, "old", string(data))
}

func TestInstall_StdxDocs_WritesManifestAndFiles(t *testing.T) {
	tarPath := buildTarGz(t, t.TempDir(), "stdx-docs.tar.gz", map[string]string{
		"libs_stdx/index.html": "stdx docs",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, tarPath)
	}))
	defer server.Close()

	tcDir := t.TempDir()
	roots := Roots{TcDir: tcDir, DocsDir: t.TempDir(), StdxDir: t.TempDir()}
	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}
	mf := manifestWithComponents(toolchain.LTS, "1.0.5", dist.ComponentSet{
		StdxDocs: &dist.ComponentInfo{URL: server.URL + "/stdx-docs.tar.gz"},
	})

	require.NoError(t, Install(context.Background(), roots, tc, StdxDocs, "", t.TempDir(), false, mf))
	assert.FileExists(t, filepath.Join(roots.DocsDir, "stdx", "libs_stdx", "index.html"))
	assert.True(t, IsInstalled(tcDir, StdxDocs))
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
		http.ServeFile(w, r, tarPath)
	}))
	defer server.Close()

	tcDir := t.TempDir()
	roots := Roots{TcDir: tcDir, DocsDir: t.TempDir(), StdxDir: t.TempDir()}
	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}
	mf := manifestWithComponents(toolchain.LTS, "1.0.5", dist.ComponentSet{
		Docs: &dist.ComponentInfo{URL: server.URL + "/docs.tar.gz"},
	})

	require.NoError(t, Install(context.Background(), roots, tc, Docs, "", t.TempDir(), false, mf))

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
		http.ServeFile(w, r, zipPath)
	}))
	defer server.Close()

	tcDir := t.TempDir()
	roots := Roots{TcDir: tcDir, DocsDir: t.TempDir(), StdxDir: t.TempDir()}
	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}
	mf := stdxComponentManifest(t, toolchain.LTS, "1.0.5", "win32-x64", server.URL+"/stdx.zip")

	require.NoError(t, Install(context.Background(), roots, tc, Stdx, "win32-x64", t.TempDir(), false, mf))

	assert.FileExists(t, filepath.Join(roots.StdxDir, "dynamic", "foo.dll"))
	assert.FileExists(t, filepath.Join(roots.StdxDir, "static", "foo.lib"))
}

func TestInstall_DocsDoesNotRequestChecksumSidecar(t *testing.T) {
	t.Setenv("CJV_MAX_RETRIES", "0")
	tarPath := buildTarGz(t, t.TempDir(), "docs.tar.gz", map[string]string{
		"index.html": "docs",
	})
	var shaRequests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if filepath.Ext(r.URL.Path) == ".sha256" {
			shaRequests.Add(1)
			_, _ = w.Write([]byte(strings.Repeat("0", 64)))
			return
		}
		http.ServeFile(w, r, tarPath)
	}))
	defer server.Close()

	tcDir := t.TempDir()
	roots := Roots{TcDir: tcDir, DocsDir: t.TempDir(), StdxDir: t.TempDir()}
	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}
	mf := manifestWithComponents(toolchain.LTS, "1.0.5", dist.ComponentSet{
		Docs: &dist.ComponentInfo{URL: server.URL + "/docs.tar.gz"},
	})

	require.NoError(t, Install(context.Background(), roots, tc, Docs, "", t.TempDir(), false, mf))
	assert.Zero(t, shaRequests.Load())
	assert.True(t, IsInstalled(tcDir, Docs))
}

// TestInstall_AllComponentsWithMockArchives installs every Name returned by
// KnownComponents against mock servers and verifies each lands on disk with a
// manifest. Guards against silent regressions when a new component is added
// to the spec map but its install path is not exercised end-to-end.
func TestInstall_AllComponentsWithMockArchives(t *testing.T) {
	const version = "1.0.5"

	tuple := "linux-arm64"
	stdxVersion := version + ".1"
	stdxArchiveName := "cangjie-stdx-linux-aarch64-" + stdxVersion + ".zip"
	stdxBuilder := func(t *testing.T, dir string) string {
		return buildZip(t, dir, stdxArchiveName, map[string]string{
			"cangjie-stdx-linux-aarch64-" + stdxVersion + "/dynamic/libfoo.so": "dynamic",
			"cangjie-stdx-linux-aarch64-" + stdxVersion + "/static/libfoo.a":   "static",
		})
	}
	if runtime.GOOS == "windows" {
		tuple = "win32-x64"
		stdxArchiveName = "cangjie-stdx-windows-x64-" + stdxVersion + ".zip"
		stdxBuilder = func(t *testing.T, dir string) string {
			return buildZip(t, dir, stdxArchiveName, map[string]string{
				"cangjie-stdx-windows-x64-" + stdxVersion + "/dynamic/foo.dll": "dynamic",
				"cangjie-stdx-windows-x64-" + stdxVersion + "/static/foo.lib":  "static",
			})
		}
	}

	srvDir := t.TempDir()
	stdxArchive := stdxBuilder(t, srvDir)
	docsArchive := buildTarGz(t, srvDir, "cangjie-docs-html-"+version+".tar.gz", map[string]string{
		"index.html": "docs",
	})
	stdxDocsArchive := buildTarGz(t, srvDir, "cangjie-stdx-docs-html-"+version+".1.tar.gz", map[string]string{
		"libs_stdx/index.html": "stdx docs",
	})

	archives := map[string]string{
		stdxArchiveName: stdxArchive,
		"cangjie-docs-html-" + version + ".tar.gz":        docsArchive,
		"cangjie-stdx-docs-html-" + version + ".1.tar.gz": stdxDocsArchive,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if path, ok := archives[filepath.Base(r.URL.Path)]; ok {
			http.ServeFile(w, r, path)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	stdxPlat, err := stdxPlatform(tuple)
	require.NoError(t, err)
	mf := manifestWithComponents(toolchain.LTS, version, dist.ComponentSet{
		Stdx:     map[string]dist.ComponentInfo{stdxPlat: {URL: server.URL + "/download/" + stdxArchiveName}},
		Docs:     &dist.ComponentInfo{URL: server.URL + "/download/cangjie-docs-html-" + version + ".tar.gz"},
		StdxDocs: &dist.ComponentInfo{URL: server.URL + "/download/cangjie-stdx-docs-html-" + version + ".1.tar.gz"},
	})

	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: version}

	stdxLib := "libfoo.so"
	if runtime.GOOS == "windows" {
		stdxLib = "foo.dll"
	}
	expectedFile := map[Name]string{
		Stdx:     filepath.Join("dynamic", stdxLib),
		Docs:     filepath.Join("main", "index.html"),
		StdxDocs: filepath.Join("stdx", "libs_stdx", "index.html"),
	}
	rootDirFor := func(name Name, roots Roots) string {
		switch name {
		case Stdx:
			return roots.StdxDir
		default:
			return roots.DocsDir
		}
	}

	known := KnownComponents()
	require.NotEmpty(t, known)
	for _, name := range known {
		t.Run(string(name), func(t *testing.T) {
			roots := Roots{TcDir: t.TempDir(), DocsDir: t.TempDir(), StdxDir: t.TempDir()}
			pk := ""
			if name == Stdx {
				pk = tuple
			}

			require.NoError(t, Install(context.Background(), roots, tc, name, pk, t.TempDir(), false, mf))
			assert.True(t, IsInstalled(roots.TcDir, name))
			assert.FileExists(t, filepath.Join(rootDirFor(name, roots), expectedFile[name]))

			manifest, err := ReadManifest(roots.TcDir, name)
			require.NoError(t, err)
			assert.NotEmpty(t, manifest)
		})
	}
}

func TestInstallRejectsUnknownAndUnsupportedComponents(t *testing.T) {
	roots := Roots{TcDir: t.TempDir(), DocsDir: t.TempDir(), StdxDir: t.TempDir()}
	tc := toolchain.ToolchainName{Channel: toolchain.STS, Version: "2.0.0"}

	require.Error(t, Install(context.Background(), roots, tc, Name("unknown"), "", t.TempDir(), false, nil))

	ltsOnly := Name("lts-only")
	specs[ltsOnly] = Spec{
		Name:              ltsOnly,
		Location:          InstallLocation{Anchor: AnchorDocs, Subdir: "lts-only"},
		StripTopLevel:     false,
		SupportedChannels: []toolchain.Channel{toolchain.LTS},
	}
	t.Cleanup(func() { delete(specs, ltsOnly) })

	err := Install(context.Background(), roots, tc, ltsOnly, "", t.TempDir(), false, nil)
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
