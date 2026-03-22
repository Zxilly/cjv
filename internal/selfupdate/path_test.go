package selfupdate

import (
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
