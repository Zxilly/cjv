//go:build windows

package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows/registry"
)

func TestRegistryPathGuardSnapshotsAndReadsPath(t *testing.T) {
	guard, err := SaveRegistryPath()
	require.NoError(t, err)
	require.NotNil(t, guard)
	assert.NotEmpty(t, guard.name)

	_, err = ReadRegistryPath()
	require.NoError(t, err)
}

func TestFindRegistryPathNameUsesStoredCasing(t *testing.T) {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\cjv-test-registry-guard`, registry.ALL_ACCESS)
	require.NoError(t, err)
	defer key.Close() //nolint:errcheck
	t.Cleanup(func() {
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\cjv-test-registry-guard`)
	})
	require.NoError(t, key.SetStringValue("PATH", "value"))

	assert.Equal(t, "PATH", findRegistryPathName(key))
}
