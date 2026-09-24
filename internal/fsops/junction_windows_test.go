//go:build windows

package fsops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for SymlinkOrJunction — creates a directory junction on Windows
// (which doesn't require admin privileges unlike symlinks).

func TestSymlinkOrJunction_CreatesLink(t *testing.T) {
	target := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(target, "test.txt"), []byte("data"), 0o644))

	link := filepath.Join(t.TempDir(), "link")
	err := SymlinkOrJunction(target, link)
	if err != nil {
		t.Skipf("junction creation failed (may need privileges): %v", err)
	}

	// Verify the link points to the target
	content, err := os.ReadFile(filepath.Join(link, "test.txt"))
	require.NoError(t, err)
	assert.Equal(t, "data", string(content))
}

func TestSymlinkOrJunctionCreatesUsableDirectoryLink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	link := filepath.Join(root, "link")
	require.NoError(t, os.MkdirAll(target, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(target, "file.txt"), []byte("content"), 0o644))

	require.NoError(t, SymlinkOrJunction(target, link))

	got, err := os.ReadFile(filepath.Join(link, "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content", string(got))
}

func TestCreateJunctionRejectsInvalidLinkPath(t *testing.T) {
	target := t.TempDir()
	err := createJunction(target, string([]byte{'b', 'a', 'd', 0, 'p', 'a', 't', 'h'}))
	require.Error(t, err)
}

func TestCreateJunctionWithRelativeTarget(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	link := filepath.Join(root, "link")
	require.NoError(t, os.MkdirAll(target, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(target, "file.txt"), []byte("content"), 0o644))

	require.NoError(t, createJunction("target", link))

	got, err := os.ReadFile(filepath.Join(link, "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content", string(got))
}
