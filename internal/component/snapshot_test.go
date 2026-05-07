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

	snap, err := TakeSnapshot(roots, []Name{Stdx})
	require.NoError(t, err)
	defer snap.Cleanup() //nolint:errcheck

	require.NoError(t, os.Remove(filepath.Join(stdxDir, "dynamic", "libfoo.so")))
	require.NoError(t, cleanupComponentMeta(tcDir, Stdx))

	require.NoError(t, snap.Restore())

	assert.True(t, IsInstalled(tcDir, Stdx))
	data, err := os.ReadFile(filepath.Join(stdxDir, "dynamic", "libfoo.so"))
	require.NoError(t, err)
	assert.Equal(t, "old", string(data))
}

func TestSnapshotRestoreRemovesPathsThatDidNotExist(t *testing.T) {
	roots := Roots{TcDir: t.TempDir(), StdxDir: filepath.Join(t.TempDir(), "stdx"), DocsDir: t.TempDir()}

	snap, err := TakeSnapshot(roots, []Name{Stdx})
	require.NoError(t, err)
	defer snap.Cleanup() //nolint:errcheck

	require.NoError(t, os.MkdirAll(filepath.Join(roots.StdxDir, "dynamic"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(roots.StdxDir, "dynamic", "libfoo.so"), []byte("new"), 0o644))
	require.NoError(t, WriteManifest(roots.TcDir, Stdx, []string{"dynamic/libfoo.so"}))

	require.NoError(t, snap.Restore())

	assert.NoDirExists(t, roots.StdxDir)
	assert.False(t, IsInstalled(roots.TcDir, Stdx))
}

func TestCopyTreeCopiesSingleFile(t *testing.T) {
	src := filepath.Join(t.TempDir(), "source.txt")
	dst := filepath.Join(t.TempDir(), "nested", "copy.txt")
	require.NoError(t, os.WriteFile(src, []byte("content"), 0o644))

	require.NoError(t, copyTree(src, dst))

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "content", string(got))
}

func TestTakeSnapshotRejectsUnknownComponent(t *testing.T) {
	roots := Roots{TcDir: t.TempDir(), StdxDir: t.TempDir(), DocsDir: t.TempDir()}

	snap, err := TakeSnapshot(roots, []Name{Name("unknown")})

	require.Error(t, err)
	assert.Nil(t, snap)
}

func TestSnapshotCleanupHandlesNilAndEmptySnapshot(t *testing.T) {
	require.NoError(t, (*Snapshot)(nil).Cleanup())
	require.NoError(t, (&Snapshot{}).Cleanup())
}

func TestCopyTreeCopiesDirectoryTree(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "dst")
	require.NoError(t, os.MkdirAll(filepath.Join(src, "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "nested", "file.txt"), []byte("tree"), 0o644))

	require.NoError(t, copyTree(src, dst))

	got, err := os.ReadFile(filepath.Join(dst, "nested", "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "tree", string(got))
}

func TestCopyTreeCopiesSymlinkWhenSupported(t *testing.T) {
	srcDir := t.TempDir()
	target := filepath.Join(srcDir, "target.txt")
	link := filepath.Join(srcDir, "link.txt")
	require.NoError(t, os.WriteFile(target, []byte("target"), 0o644))
	if err := os.Symlink("target.txt", link); err != nil {
		t.Skipf("symlink creation requires privileges on this platform: %v", err)
	}

	dst := filepath.Join(t.TempDir(), "link-copy.txt")
	require.NoError(t, copyTree(link, dst))

	gotTarget, err := os.Readlink(dst)
	require.NoError(t, err)
	assert.Equal(t, "target.txt", gotTarget)
}
