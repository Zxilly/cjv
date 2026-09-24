//go:build windows

package reachable

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// guardedRegistry isolates CJV_HOME and snapshots the user PATH in the
// registry so the tests can exercise real registry writes and leave the
// machine as it was. Like the integration tests, the writes run only on CI
// so a developer's registry is never touched.
func guardedRegistry(t *testing.T) (binDir string) {
	t.Helper()
	if os.Getenv("CI") != "true" {
		t.Skip("registry PATH writes run only on CI")
	}
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvNoPathSetup, "")
	guard, err := testutil.SaveRegistryPath()
	require.NoError(t, err)
	t.Cleanup(guard.Restore)
	return filepath.Join(home, "bin")
}

func registryPathCount(t *testing.T, binDir string) int {
	t.Helper()
	value, err := testutil.ReadRegistryPath()
	require.NoError(t, err)
	count := 0
	for _, entry := range strings.Split(value, string(os.PathListSeparator)) {
		if strings.EqualFold(entry, binDir) {
			count++
		}
	}
	return count
}

func TestConfigurePathAddsBinDirToRegistryOnce(t *testing.T) {
	binDir := guardedRegistry(t)

	ConfigurePath()
	ConfigurePath()

	assert.Equal(t, 1, registryPathCount(t, binDir))
}

func TestConfigurePathSkipsWhenDisabledByEnv(t *testing.T) {
	binDir := guardedRegistry(t)
	t.Setenv(config.EnvNoPathSetup, "1")

	ConfigurePath()

	assert.Equal(t, 0, registryPathCount(t, binDir))
}

func TestRemovePathUndoesConfigurePath(t *testing.T) {
	binDir := guardedRegistry(t)

	ConfigurePath()
	RemovePath()

	assert.Equal(t, 0, registryPathCount(t, binDir))
}

func TestEnsureConfiguresPathOnlyWhenPolicyAsks(t *testing.T) {
	binDir := guardedRegistry(t)

	require.NoError(t, Ensure(Policy{}))
	assert.Equal(t, 0, registryPathCount(t, binDir))

	require.NoError(t, Ensure(Policy{ConfigurePath: true}))
	assert.Equal(t, 1, registryPathCount(t, binDir))
}
