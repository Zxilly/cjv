//go:build unix

package fsops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoveOwnedTreePreservesLinksAndExternalModes(t *testing.T) {
	base := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	sibling := filepath.Join(base, "keep")
	owned := filepath.Join(base, "owned")
	for _, dir := range []string{outside, sibling, owned} {
		require.NoError(t, os.Mkdir(dir, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "data"), []byte("keep"), 0o444))
	}
	t.Cleanup(func() {
		for _, dir := range []string{outside, sibling, owned} {
			_ = os.Chmod(dir, 0o700)
		}
	})
	if hardlinksAvailable(t, base) {
		require.NoError(t, os.Link(filepath.Join(outside, "data"), filepath.Join(owned, "hardlink")))
	} else {
		t.Log("filesystem policy denies hard links; still checking symlinks and external modes")
	}
	require.NoError(t, os.Symlink(outside, filepath.Join(owned, "external")))
	require.NoError(t, os.Symlink("../keep", filepath.Join(owned, "internal")))
	for _, dir := range []string{outside, sibling, owned} {
		require.NoError(t, os.Chmod(dir, 0o555))
	}
	root, err := os.OpenRoot(base)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, root.Close()) })
	require.NoError(t, RemoveOwnedTree(root, "owned"))
	require.NoDirExists(t, owned)
	for _, dir := range []string{outside, sibling} {
		info, err := os.Stat(dir)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o555), info.Mode().Perm())
		info, err = os.Stat(filepath.Join(dir, "data"))
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o444), info.Mode().Perm())
		data, err := os.ReadFile(filepath.Join(dir, "data"))
		require.NoError(t, err)
		require.Equal(t, "keep", string(data))
	}
}

func TestRemoveOwnedTreeRejectsAncestorLinksAndRoot(t *testing.T) {
	base := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(base, "keep", "child"), 0o755))
	require.NoError(t, os.Symlink("keep", filepath.Join(base, "alias")))
	root, err := os.OpenRoot(base)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, root.Close()) })
	for _, path := range []string{"alias/child", ".", "", "..", "../keep", base} {
		require.Error(t, RemoveOwnedTree(root, path), path)
	}
	require.DirExists(t, filepath.Join(base, "keep", "child"))
	require.NoError(t, RemoveOwnedTree(root, "alias"), "removing the link itself is allowed")
	require.DirExists(t, filepath.Join(base, "keep", "child"))
	require.NoError(t, RemoveOwnedTree(root, "missing/child"))
}
