package reachable

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/Zxilly/cjv/internal/selfupdate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func envScriptNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"env.ps1", "env.bat"}
	}
	return []string{"env"}
}

func TestEnsureWaitsBeforeCleaningAnActiveSelfUpdate(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	lock, err := selfupdate.LockManaged(t.Context())
	require.NoError(t, err)
	bin := filepath.Join(home, "bin")
	require.NoError(t, os.MkdirAll(bin, 0o755))
	staged := filepath.Join(bin, "."+sdktools.CjvBinaryName()+".new")
	require.NoError(t, os.WriteFile(staged, []byte("active replacement"), 0o755))
	var ensureErr error
	done := make(chan struct{})
	go func() {
		ensureErr = Ensure(t.Context(), Policy{})
		close(done)
	}()
	defer func() {
		_ = lock.Close()
		<-done
	}()
	select {
	case <-done:
		t.Error("managed setup must wait for the active self-update lock")
	case <-time.After(80 * time.Millisecond):
	}
	assert.FileExists(t, staged, "setup must not delete another updater's staged binary")
	require.NoError(t, lock.Close())
	<-done
	require.NoError(t, ensureErr)
	assert.NoFileExists(t, staged, "interrupted leftovers can be cleaned after owning the lock")
}

func TestEnsureEstablishesManagedBinaryAndProxyLinks(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	t.Setenv(config.EnvNoPathSetup, "1")

	require.NoError(t, Ensure(t.Context(), Policy{}))

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
	configtest.Isolate(t, home)
	t.Setenv(config.EnvNoPathSetup, "1")

	require.NoError(t, Ensure(t.Context(), Policy{EnvScripts: true}))

	for _, name := range envScriptNames() {
		assert.FileExists(t, filepath.Join(home, name))
	}
}

func TestEnsureKeepsOrReplacesManagedBinaryPerPolicy(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	t.Setenv(config.EnvNoPathSetup, "1")
	binDir := filepath.Join(home, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	managed := filepath.Join(binDir, sdktools.CjvBinaryName())
	require.NoError(t, os.WriteFile(managed, []byte("stale"), 0o755))

	require.NoError(t, Ensure(t.Context(), Policy{}))
	data, err := os.ReadFile(managed)
	require.NoError(t, err)
	assert.Equal(t, "stale", string(data), "an existing managed binary is kept by default")

	require.NoError(t, Ensure(t.Context(), Policy{ForceManagedBinary: true}))
	data, err = os.ReadFile(managed)
	require.NoError(t, err)
	assert.NotEqual(t, "stale", string(data), "ForceManagedBinary replaces it with the running executable")
	assert.FileExists(t, filepath.Join(binDir, sdktools.PlatformBinaryName("cjc")))
}
