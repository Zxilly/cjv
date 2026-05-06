package component

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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

