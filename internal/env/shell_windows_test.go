//go:build windows

package env

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows/registry"
)

func TestFindPathValueNameAndSetPathValue(t *testing.T) {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\cjv-test-shell-windows`, registry.ALL_ACCESS)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\cjv-test-shell-windows`)
	})
	defer key.Close() //nolint:errcheck

	assert.Equal(t, "Path", findPathValueName(key))
	require.NoError(t, setPathValue(key, "PATH", `%USERPROFILE%\bin`))
	assert.Equal(t, "PATH", findPathValueName(key))

	got, valueType, err := key.GetStringValue("PATH")
	require.NoError(t, err)
	assert.Equal(t, uint32(registry.EXPAND_SZ), valueType)
	assert.Equal(t, `%USERPROFILE%\bin`, got)
}

func TestBroadcastSettingChangeDoesNotPanic(t *testing.T) {
	assert.NotPanics(t, broadcastSettingChange)
}

func TestAddAndRemovePathUsingProvidedRegistryKey(t *testing.T) {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\cjv-test-shell-windows-path`, registry.ALL_ACCESS)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\cjv-test-shell-windows-path`)
	})
	defer key.Close() //nolint:errcheck

	require.NoError(t, setPathValue(key, "Path", `%USERPROFILE%\bin`))
	binDir := `C:\cjv-test-bin`
	require.NoError(t, removePathFromRegistryKey(key, binDir))
	require.NoError(t, addPathToRegistryKey(key, binDir))
	require.NoError(t, addPathToRegistryKey(key, binDir))

	got, _, err := key.GetStringValue("Path")
	require.NoError(t, err)
	entries := strings.Split(got, string(os.PathListSeparator))
	var count int
	for _, entry := range entries {
		if strings.EqualFold(entry, binDir) {
			count++
		}
	}
	assert.Equal(t, 1, count)

	require.NoError(t, removePathFromRegistryKey(key, binDir))
	got, _, err = key.GetStringValue("Path")
	require.NoError(t, err)
	assert.NotContains(t, strings.ToLower(got), strings.ToLower(binDir))
	require.NoError(t, removePathFromRegistryKey(key, binDir))
}
