//go:build !windows

package fsops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateLinkUsesRelativeSymlinkOnUnix(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "cjv")
	dst := filepath.Join(dir, "cjc")
	require.NoError(t, os.WriteFile(src, []byte("binary"), 0o755))
	probe := filepath.Join(dir, "symlink-probe")
	if err := os.Symlink("cjv", probe); err != nil {
		t.Skipf("symlinks unavailable on this filesystem: %v", err)
	}
	require.NoError(t, os.Remove(probe))
	require.NoError(t, CreateLink(src, dst))
	info, err := os.Lstat(dst)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&os.ModeSymlink)
	target, err := os.Readlink(dst)
	require.NoError(t, err)
	assert.Equal(t, "cjv", target)
}
