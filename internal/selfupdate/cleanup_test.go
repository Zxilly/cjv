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

func TestCleanupOldBinariesRemovesDotOldAndGcFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	binDir := filepath.Join(home, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	managed := filepath.Join(binDir, proxy.CjvBinaryName())
	require.NoError(t, os.WriteFile(managed, []byte("current"), 0o755))

	dotOld := filepath.Join(binDir, "."+proxy.CjvBinaryName()+".old")
	gcFile := filepath.Join(binDir, gcTestName(proxy.CjvBinaryName()))
	require.NoError(t, os.WriteFile(dotOld, []byte("old"), 0o755))
	require.NoError(t, os.WriteFile(gcFile, []byte("gc"), 0o755))

	CleanupOldBinaries()

	assert.FileExists(t, managed)
	assert.NoFileExists(t, dotOld)
	assert.NoFileExists(t, gcFile)
}

func gcTestName(base string) string {
	ext := filepath.Ext(base)
	stem := base[:len(base)-len(ext)]
	return stem + "-gc-deadbeef" + ext
}
