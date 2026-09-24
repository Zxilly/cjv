package component

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotRestoresComponentRootsAndMetadata(t *testing.T) {
	tcDir := t.TempDir()
	stdxDir := t.TempDir()
	roots := Roots{TcDir: tcDir, StdxDir: stdxDir, DocsDir: t.TempDir()}
	require.NoError(t, os.MkdirAll(filepath.Join(stdxDir, "dynamic"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stdxDir, "dynamic", "libfoo.so"), []byte("old"), 0o644))
	require.NoError(t, WriteManifest(tcDir, Stdx, []string{"dynamic/libfoo.so"}))

	snap, err := takeSnapshot(roots, []Name{Stdx})
	require.NoError(t, err)
	defer snap.cleanup() //nolint:errcheck

	require.NoError(t, os.Remove(filepath.Join(stdxDir, "dynamic", "libfoo.so")))
	require.NoError(t, Remove(roots, Stdx))

	require.NoError(t, snap.restore())

	assert.True(t, IsInstalled(tcDir, Stdx))
	data, err := os.ReadFile(filepath.Join(stdxDir, "dynamic", "libfoo.so"))
	require.NoError(t, err)
	assert.Equal(t, "old", string(data))
}

func TestSnapshotRestoreRemovesPathsThatDidNotExist(t *testing.T) {
	roots := Roots{TcDir: t.TempDir(), StdxDir: filepath.Join(t.TempDir(), "stdx"), DocsDir: t.TempDir()}

	snap, err := takeSnapshot(roots, []Name{Stdx})
	require.NoError(t, err)
	defer snap.cleanup() //nolint:errcheck

	require.NoError(t, os.MkdirAll(filepath.Join(roots.StdxDir, "dynamic"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(roots.StdxDir, "dynamic", "libfoo.so"), []byte("new"), 0o644))
	require.NoError(t, WriteManifest(roots.TcDir, Stdx, []string{"dynamic/libfoo.so"}))

	require.NoError(t, snap.restore())

	assert.NoDirExists(t, roots.StdxDir)
	assert.False(t, IsInstalled(roots.TcDir, Stdx))
}

func TestTakeSnapshotRejectsUnknownComponent(t *testing.T) {
	roots := Roots{TcDir: t.TempDir(), StdxDir: t.TempDir(), DocsDir: t.TempDir()}

	snap, err := takeSnapshot(roots, []Name{Name("unknown")})

	require.Error(t, err)
	assert.Nil(t, snap)
}

func TestSnapshotCleanupHandlesNilAndEmptySnapshot(t *testing.T) {
	require.NoError(t, (*snapshot)(nil).cleanup())
	require.NoError(t, (&snapshot{}).cleanup())
}
