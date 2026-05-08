package selfupdate

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedExecutablePathUsesManagedBinDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	managed := filepath.Join(home, "bin", proxy.CjvBinaryName())
	require.NoError(t, os.MkdirAll(filepath.Dir(managed), 0o755))
	require.NoError(t, os.WriteFile(managed, []byte("stub"), 0o755))

	got, err := ManagedExecutablePath()
	require.NoError(t, err)
	assert.Equal(t, managed, got)
}

func TestManagedExecutablePathRequiresManagedBinary(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	_, err := ManagedExecutablePath()
	require.Error(t, err)
}

// --- Tests merged from managed_binary_test.go ---

// Tests for ManagedExecutablePath -- locates the managed cjv binary
// under CJV_HOME/bin. Self-update and uninstall need this path to
// know which binary to replace or remove.

func TestManagedExecutablePath_FindsBinary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	binDir := filepath.Join(home, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))

	binaryName := proxy.CjvBinaryName()
	binaryPath := filepath.Join(binDir, binaryName)
	require.NoError(t, os.WriteFile(binaryPath, []byte("stub"), 0o755))

	path, err := ManagedExecutablePath()
	require.NoError(t, err)
	assert.Equal(t, binaryPath, path)
}

func TestManagedExecutablePath_ErrorWhenMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	// bin dir exists but no cjv binary
	require.NoError(t, os.MkdirAll(filepath.Join(home, "bin"), 0o755))

	_, err := ManagedExecutablePath()
	assert.Error(t, err, "should fail when managed binary doesn't exist")
}

func TestForceUpdateManagedExecutable(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	// First call: managed binary does not exist yet; should install it
	got, err := ForceUpdateManagedExecutable()
	require.NoError(t, err)

	expected := filepath.Join(home, "bin", proxy.CjvBinaryName())
	assert.Equal(t, expected, got)

	info, err := os.Stat(got)
	require.NoError(t, err)
	originalSize := info.Size()

	// Overwrite the managed binary with dummy content
	require.NoError(t, os.WriteFile(got, []byte("old"), 0o755))

	// Second call: should overwrite the dummy content with the real binary
	got2, err := ForceUpdateManagedExecutable()
	require.NoError(t, err)
	assert.Equal(t, expected, got2)

	info2, err := os.Stat(got2)
	require.NoError(t, err)
	assert.Equal(t, originalSize, info2.Size(), "managed binary should be restored to original size")
	assert.NotEqual(t, int64(3), info2.Size(), "managed binary should not still be the 3-byte dummy")
}

func TestForceUpdateManagedExecutablePreservesExistingBinaryOnCopyFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	managed := filepath.Join(home, "bin", proxy.CjvBinaryName())
	require.NoError(t, os.MkdirAll(filepath.Dir(managed), 0o755))
	require.NoError(t, os.WriteFile(managed, []byte("old-binary"), 0o755))

	originalCopy := copyManagedExecutableFile
	copyManagedExecutableFile = func(src, dst string, mode os.FileMode) error {
		require.NoError(t, os.WriteFile(dst, []byte("partial"), mode))
		return errors.New("copy failed")
	}
	t.Cleanup(func() {
		copyManagedExecutableFile = originalCopy
	})

	_, err := ForceUpdateManagedExecutable()

	require.Error(t, err)
	data, err := os.ReadFile(managed)
	require.NoError(t, err)
	assert.Equal(t, []byte("old-binary"), data)
}

func TestEnsureManagedExecutableCopiesCurrentBinary(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	got, err := EnsureManagedExecutable()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "bin", proxy.CjvBinaryName()), got)
	assert.FileExists(t, got)

	gotAgain, err := EnsureManagedExecutable()
	require.NoError(t, err)
	assert.Equal(t, got, gotAgain)
}
