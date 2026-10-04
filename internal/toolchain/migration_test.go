package toolchain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLegacyMigrationPreservesSelectorsAndIndependentComponentRoots(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())
	settings := config.DefaultSettings()
	settings.DefaultToolchain = "sts-1.0.0"
	settings.Overrides["project"] = "sts"
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	require.NoError(t, sf.Save(&settings))
	before, err := os.ReadFile(sf.Path())
	require.NoError(t, err)
	for _, version := range []string{"sts-1.0.0", "sts-1.10.0", "sts-1.9.0"} {
		for _, root := range []string{"toolchains", "docs", "stdx"} {
			path := filepath.Join(home, root, version)
			require.NoError(t, os.MkdirAll(path, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(path, "user.txt"), []byte(version), 0o644))
		}
	}
	custom := filepath.Join(home, "toolchains", "my-sdk")
	require.NoError(t, os.MkdirAll(custom, 0o755))
	require.NoError(t, RecoverHome())
	installed, err := ReadInstallation(filepath.Join(home, "toolchains", "sts"))
	require.NoError(t, err)
	assert.Equal(t, "sts-1.10.0", installed.Release)
	for _, root := range []string{"toolchains", "docs", "stdx"} {
		oldPath := filepath.Join(home, root, "sts-1.10.0", "user.txt")
		newPath := filepath.Join(home, root, "sts", "user.txt")
		a, err := os.Stat(oldPath)
		require.NoError(t, err)
		b, err := os.Stat(newPath)
		require.NoError(t, err)
		assert.False(t, os.SameFile(a, b), "legacy contents must be copied")
		require.NoError(t, os.WriteFile(newPath, []byte("channel change"), 0o644))
		data, err := os.ReadFile(oldPath)
		require.NoError(t, err)
		assert.Equal(t, "sts-1.10.0", string(data))
	}
	after, err := os.ReadFile(sf.Path())
	require.NoError(t, err)
	assert.Equal(t, before, after)
	require.NoError(t, os.RemoveAll(filepath.Join(home, "toolchains", "sts")))
	require.NoError(t, RecoverHome())
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "sts"), "completed migration must not resurrect an uninstalled channel")
	assert.DirExists(t, custom)
}

func TestLegacyMigrationFailureRollsBackAndRetries(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())
	for _, root := range []string{"toolchains", "docs"} {
		path := filepath.Join(home, root, "sts-1.0.0")
		require.NoError(t, os.MkdirAll(path, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(path, "data"), []byte("legacy"), 0o644))
	}
	// A destination obstruction must roll the already placed SDK back.
	obstruction := filepath.Join(home, "docs", "sts")
	require.NoError(t, os.WriteFile(obstruction, []byte("user file"), 0o644))
	require.Error(t, RecoverHome())
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "sts"))
	assert.NoFileExists(t, filepath.Join(home, ".cjv", "layout-v2"))
	assert.FileExists(t, filepath.Join(home, "toolchains", "sts-1.0.0", "data"))
	require.NoError(t, os.Remove(obstruction))
	require.NoError(t, RecoverHome())
	assert.FileExists(t, filepath.Join(home, "docs", "sts", "data"))
}
