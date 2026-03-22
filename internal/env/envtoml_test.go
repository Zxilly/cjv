package env

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvTomlRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "env.toml")

	e := &EnvConfig{
		Vars: map[string]string{
			"CANGJIE_HOME": "/path/to/sdk",
		},
		PathPrepend: PathPrepend{
			Entries: []string{"/path/to/sdk/bin", "/path/to/sdk/tools/bin"},
		},
	}

	require.NoError(t, e.Save(path))

	loaded, err := LoadEnvConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "/path/to/sdk", loaded.Vars["CANGJIE_HOME"])
	assert.Len(t, loaded.PathPrepend.Entries, 2)
}

func TestLoadEnvConfigMissing(t *testing.T) {
	loaded, err := LoadEnvConfig(filepath.Join(t.TempDir(), "nonexistent.toml"))
	require.NoError(t, err)
	assert.NotNil(t, loaded)
	assert.NotNil(t, loaded.Vars)
	assert.Empty(t, loaded.Vars)
}

func TestLoadEnvConfigEmpty(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "env.toml")

	e := &EnvConfig{Vars: make(map[string]string)}
	require.NoError(t, e.Save(path))

	loaded, err := LoadEnvConfig(path)
	require.NoError(t, err)
	assert.NotNil(t, loaded.Vars)
}

func TestLoadToolchainEnv_CapturesOnDemandWhenEnvTomlMissing(t *testing.T) {
	tcDir := t.TempDir()

	var scriptPath string
	var contents string
	if runtime.GOOS == "windows" {
		scriptPath = filepath.Join(tcDir, "envsetup.ps1")
		contents = "$env:CANGJIE_HOME = $PWD.Path"
	} else {
		scriptPath = filepath.Join(tcDir, "envsetup.sh")
		contents = "export CANGJIE_HOME=\"$PWD\"\n"
	}
	require.NoError(t, os.WriteFile(scriptPath, []byte(contents), 0o755))

	cfg := LoadToolchainEnv(context.Background(), tcDir)
	assert.Equal(t, tcDir, cfg.Vars["CANGJIE_HOME"])
}

func TestLoadEnvConfig_NonExistentFileReturnsEmpty(t *testing.T) {
	cfg, err := LoadEnvConfig(filepath.Join(t.TempDir(), "no-such-file.toml"))
	require.NoError(t, err)
	assert.Empty(t, cfg.Vars)
	assert.Empty(t, cfg.PathPrepend.Entries)
}

func TestEnvConfig_SaveAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "env.toml")

	cfg := NewEnvConfig()
	cfg.Vars["CANGJIE_HOME"] = "/opt/sdk"
	cfg.PathPrepend.Entries = []string{"/opt/sdk/bin", "/opt/sdk/tools/bin"}

	require.NoError(t, cfg.Save(path))

	reloaded, err := LoadEnvConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "/opt/sdk", reloaded.Vars["CANGJIE_HOME"])
	assert.Equal(t, []string{"/opt/sdk/bin", "/opt/sdk/tools/bin"}, reloaded.PathPrepend.Entries)
}

func TestLoadEnvConfig_EmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.toml")
	require.NoError(t, os.WriteFile(path, []byte(""), 0o644))

	cfg, err := LoadEnvConfig(path)
	require.NoError(t, err)
	assert.Empty(t, cfg.Vars)
}

func TestEnvConfig_SaveCreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.toml")

	cfg := NewEnvConfig()
	cfg.Vars["KEY"] = "value"
	require.NoError(t, cfg.Save(path))

	assert.FileExists(t, path)
}
