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

func TestListInstalledIgnoresUnknownAndMissingManifests(t *testing.T) {
	tcDir := t.TempDir()
	require.NoError(t, os.MkdirAll(metaPath(tcDir), 0o755))
	require.NoError(t, os.WriteFile(metaPath(tcDir, componentsFile), []byte("docs\nunknown\nstdx\n"), 0o644))
	require.NoError(t, os.WriteFile(metaPath(tcDir, "manifest-docs"), []byte("index.html\n"), 0o644))

	got, err := ListInstalled(tcDir)

	require.NoError(t, err)
	assert.Equal(t, []Name{Docs}, got)
}

func TestRemoveMissingManifestTidiesIndex(t *testing.T) {
	tcDir := t.TempDir()
	roots := Roots{TcDir: tcDir, DocsDir: t.TempDir(), StdxDir: t.TempDir()}
	require.NoError(t, os.MkdirAll(metaPath(tcDir), 0o755))
	require.NoError(t, os.WriteFile(metaPath(tcDir, componentsFile), []byte("docs\n"), 0o644))

	require.NoError(t, Remove(roots, Docs))

	got, err := ListInstalled(tcDir)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestRemoveKeepsFilesClaimedByComponentWithSameRoot(t *testing.T) {
	tcDir := t.TempDir()
	docsRoot := t.TempDir()
	roots := Roots{TcDir: tcDir, DocsDir: docsRoot, StdxDir: t.TempDir()}
	sharedName := Name("docs-copy")
	specs[sharedName] = Spec{
		Name:              sharedName,
		Location:          InstallLocation{Anchor: AnchorDocs, Subdir: "main"},
		StripTopLevel:     false,
		SupportedChannels: specs[Docs].SupportedChannels,
	}
	t.Cleanup(func() { delete(specs, sharedName) })

	mainDir := filepath.Join(docsRoot, "main")
	require.NoError(t, os.MkdirAll(mainDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "shared.css"), []byte("shared"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "docs-only.js"), []byte("docs"), 0o644))
	require.NoError(t, WriteManifest(tcDir, Docs, []string{"shared.css", "docs-only.js"}))
	require.NoError(t, WriteManifest(tcDir, sharedName, []string{"shared.css"}))

	require.NoError(t, Remove(roots, Docs))

	assert.FileExists(t, filepath.Join(mainDir, "shared.css"))
	assert.NoFileExists(t, filepath.Join(mainDir, "docs-only.js"))
	assert.True(t, IsInstalled(tcDir, sharedName))
	assert.False(t, IsInstalled(tcDir, Docs))
}

func TestManifestErrors(t *testing.T) {
	tcDir := t.TempDir()
	require.Error(t, func() error {
		_, err := ReadManifest(tcDir, Docs)
		return err
	}())

	require.NoError(t, os.WriteFile(filepath.Join(tcDir, ".cjv"), []byte("not a directory"), 0o644))
	require.Error(t, WriteManifest(tcDir, Docs, []string{"index.html"}))
	require.Error(t, writeComponentsIndex(tcDir, []Name{Docs}))
}
