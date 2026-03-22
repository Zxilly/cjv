package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwapInstalledToolchainRollsBackExistingInstallOnTailFailure(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "toolchain")
	staging := dest + ".staging"

	require.NoError(t, os.MkdirAll(dest, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dest, "version.txt"), []byte("old"), 0o644))
	require.NoError(t, os.MkdirAll(staging, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(staging, "version.txt"), []byte("new"), 0o644))

	err := swapInstalledToolchain(staging, dest, true, func() error {
		return errors.New("proxy refresh failed")
	})
	require.Error(t, err)

	data, readErr := os.ReadFile(filepath.Join(dest, "version.txt"))
	require.NoError(t, readErr)
	assert.Equal(t, []byte("old"), data)
	assert.NoFileExists(t, dest+".old")
	assert.NoFileExists(t, staging)
}

func TestSwapInstalledToolchainRemovesNewInstallOnTailFailure(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "toolchain")
	staging := dest + ".staging"

	require.NoError(t, os.MkdirAll(staging, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(staging, "version.txt"), []byte("new"), 0o644))

	err := swapInstalledToolchain(staging, dest, false, func() error {
		return errors.New("settings save failed")
	})
	require.Error(t, err)
	assert.NoFileExists(t, dest)
	assert.NoFileExists(t, staging)
}
