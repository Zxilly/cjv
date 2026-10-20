package selfupdate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedExecutablePathUsesManagedBinDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	managed := filepath.Join(home, "bin", sdktools.CjvBinaryName())
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

	binaryName := sdktools.CjvBinaryName()
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

	expected := filepath.Join(home, "bin", sdktools.CjvBinaryName())
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

// The managed path is replaced by renaming a fully copied temporary file over
// it, so a failed replacement leaves whatever was there untouched and no
// partial copy behind. A non-empty directory at the managed path makes the
// rename fail on every platform.
func TestForceUpdateManagedExecutablePreservesExistingEntryOnReplaceFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	managed := filepath.Join(home, "bin", sdktools.CjvBinaryName())
	require.NoError(t, os.MkdirAll(managed, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(managed, "keep"), []byte("old-binary"), 0o644))

	_, err := ForceUpdateManagedExecutable()

	require.Error(t, err)
	data, err := os.ReadFile(filepath.Join(managed, "keep"))
	require.NoError(t, err)
	assert.Equal(t, []byte("old-binary"), data)

	entries, err := os.ReadDir(filepath.Dir(managed))
	require.NoError(t, err)
	assert.Len(t, entries, 1, "no temporary copy may be left beside the managed path")
}

func TestEnsureManagedExecutableCopiesCurrentBinary(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	got, err := EnsureManagedExecutable()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "bin", sdktools.CjvBinaryName()), got)
	assert.FileExists(t, got)

	gotAgain, err := EnsureManagedExecutable()
	require.NoError(t, err)
	assert.Equal(t, got, gotAgain)
}
