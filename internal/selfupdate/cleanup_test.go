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

func TestCleanupOldBinariesRemovesDotOldAndGcFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	binDir := filepath.Join(home, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	managed := filepath.Join(binDir, sdktools.CjvBinaryName())
	require.NoError(t, os.WriteFile(managed, []byte("current"), 0o755))

	dotOld := filepath.Join(binDir, "."+sdktools.CjvBinaryName()+".old")
	gcFile := filepath.Join(binDir, gcTestName(sdktools.CjvBinaryName()))
	require.NoError(t, os.WriteFile(dotOld, []byte("old"), 0o755))
	require.NoError(t, os.WriteFile(gcFile, []byte("gc"), 0o755))

	CleanupOldBinaries()

	assert.FileExists(t, managed)
	assert.NoFileExists(t, dotOld)
	assert.NoFileExists(t, gcFile)
}

func TestCleanupOldBinariesRecoversInterruptedSwap(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)
	bin := filepath.Join(home, "bin")
	require.NoError(t, os.MkdirAll(bin, 0o755))
	managed := filepath.Join(bin, sdktools.CjvBinaryName())
	old := filepath.Join(bin, "."+sdktools.CjvBinaryName()+".old")
	staged := filepath.Join(bin, "."+sdktools.CjvBinaryName()+".new")
	require.NoError(t, os.WriteFile(old, []byte("previous executable"), 0o755))
	require.NoError(t, os.WriteFile(staged, []byte("unfinished replacement"), 0o755))
	CleanupOldBinaries()
	data, err := os.ReadFile(managed)
	require.NoError(t, err)
	assert.Equal(t, "previous executable", string(data))
	assert.NoFileExists(t, old)
	assert.NoFileExists(t, staged)
}

func gcTestName(base string) string {
	ext := filepath.Ext(base)
	stem := base[:len(base)-len(ext)]
	return stem + "-gc-deadbeef" + ext
}
