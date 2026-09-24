package reachable

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func envScriptNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"env.ps1", "env.bat"}
	}
	return []string{"env"}
}

func TestEnsureEstablishesManagedBinaryAndProxyLinks(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvNoPathSetup, "1")

	require.NoError(t, Ensure(Policy{}))

	binDir := filepath.Join(home, "bin")
	assert.FileExists(t, filepath.Join(binDir, sdktools.CjvBinaryName()))
	for _, tool := range sdktools.AllProxyTools() {
		assert.FileExists(t, filepath.Join(binDir, sdktools.PlatformBinaryName(tool)), tool)
	}
	for _, name := range envScriptNames() {
		assert.NoFileExists(t, filepath.Join(home, name), "env scripts are written only on request")
	}
}

func TestEnsureWritesEnvScriptsOnRequest(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvNoPathSetup, "1")

	require.NoError(t, Ensure(Policy{EnvScripts: true}))

	for _, name := range envScriptNames() {
		assert.FileExists(t, filepath.Join(home, name))
	}
}

func TestEnsureKeepsOrReplacesManagedBinaryPerPolicy(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvNoPathSetup, "1")
	binDir := filepath.Join(home, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	managed := filepath.Join(binDir, sdktools.CjvBinaryName())
	require.NoError(t, os.WriteFile(managed, []byte("stale"), 0o755))

	require.NoError(t, Ensure(Policy{}))
	data, err := os.ReadFile(managed)
	require.NoError(t, err)
	assert.Equal(t, "stale", string(data), "an existing managed binary is kept by default")

	require.NoError(t, Ensure(Policy{ForceManagedBinary: true}))
	data, err = os.ReadFile(managed)
	require.NoError(t, err)
	assert.NotEqual(t, "stale", string(data), "ForceManagedBinary replaces it with the running executable")
	assert.FileExists(t, filepath.Join(binDir, sdktools.PlatformBinaryName("cjc")))
}
