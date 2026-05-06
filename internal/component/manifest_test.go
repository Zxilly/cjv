package component

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifestRoundTrip(t *testing.T) {
	tcDir := t.TempDir()

	require.NoError(t, WriteManifest(tcDir, Stdx, []string{"dynamic/libfoo.so", "static/libfoo.a"}))

	got, err := ReadManifest(tcDir, Stdx)
	require.NoError(t, err)
	assert.Equal(t, []string{"dynamic/libfoo.so", "static/libfoo.a"}, got)

	installed, err := ListInstalled(tcDir)
	require.NoError(t, err)
	assert.Equal(t, []Name{Stdx}, installed)

	assert.True(t, IsInstalled(tcDir, Stdx))
	assert.False(t, IsInstalled(tcDir, Docs))
}

func TestRemove_DeletesTrackedFilesAndPrunesEmptyDirs(t *testing.T) {
	tcDir := t.TempDir()
	docsDir := t.TempDir()
	stdxDir := t.TempDir()
	roots := Roots{TcDir: tcDir, DocsDir: docsDir, StdxDir: stdxDir}
	require.NoError(t, os.MkdirAll(filepath.Join(stdxDir, "dynamic"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(stdxDir, "static"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stdxDir, "dynamic", "libfoo.so"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(stdxDir, "static", "libfoo.a"), []byte("y"), 0o644))

	require.NoError(t, WriteManifest(tcDir, Stdx, []string{"dynamic/libfoo.so", "static/libfoo.a"}))

	require.NoError(t, Remove(roots, Stdx))

	assert.NoFileExists(t, filepath.Join(stdxDir, "dynamic", "libfoo.so"))
	assert.NoFileExists(t, filepath.Join(stdxDir, "static", "libfoo.a"))
	// Empty subdirs are pruned.
	assert.NoDirExists(t, filepath.Join(stdxDir, "dynamic"))
	assert.NoDirExists(t, filepath.Join(stdxDir, "static"))
	// Manifest gone.
	assert.False(t, IsInstalled(tcDir, Stdx))
}

func TestRemove_DocsAndStdxDocsAreIsolated(t *testing.T) {
	// Each doc component lives under its own subdir (main/, stdx/), so
	// removing one must not touch the other's tree.
	tcDir := t.TempDir()
	docsRoot := t.TempDir()
	roots := Roots{TcDir: tcDir, DocsDir: docsRoot}

	mainDir := filepath.Join(docsRoot, "main")
	stdxDir := filepath.Join(docsRoot, "stdx")
	require.NoError(t, os.MkdirAll(filepath.Join(mainDir, "assets"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(stdxDir, "assets"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "index.html"), []byte("main"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "assets", "style.css"), []byte("m"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(stdxDir, "index.html"), []byte("stdx"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(stdxDir, "assets", "style.css"), []byte("s"), 0o644))

	require.NoError(t, WriteManifest(tcDir, Docs, []string{"index.html", "assets/style.css"}))
	require.NoError(t, WriteManifest(tcDir, StdxDocs, []string{"index.html", "assets/style.css"}))

	require.NoError(t, Remove(roots, StdxDocs))

	// stdx subtree gone, main subtree intact.
	assert.NoFileExists(t, filepath.Join(stdxDir, "index.html"))
	assert.NoFileExists(t, filepath.Join(stdxDir, "assets", "style.css"))
	assert.FileExists(t, filepath.Join(mainDir, "index.html"))
	assert.FileExists(t, filepath.Join(mainDir, "assets", "style.css"))

	assert.True(t, IsInstalled(tcDir, Docs))
	assert.False(t, IsInstalled(tcDir, StdxDocs))
}
