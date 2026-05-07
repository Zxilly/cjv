//go:build windows

package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestProcessNameFindsCurrentProcessAndRejectsMissingPID(t *testing.T) {
	name, err := ProcessName(os.Getpid())
	require.NoError(t, err)
	assert.NotEmpty(t, name)

	_, err = ProcessName(-1)
	require.Error(t, err)
}
