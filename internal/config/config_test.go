package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHomeDefault(t *testing.T) {
	t.Setenv(EnvHome, "")
	home, err := Home()
	require.NoError(t, err)
	userHome, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(userHome, ".cjv"), home)
}

func TestHomeEnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvHome, dir)
	home, err := Home()
	require.NoError(t, err)
	expected, _ := filepath.Abs(dir)
	assert.Equal(t, expected, home)
}

func TestEnsureDirs(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv(EnvHome, tmp)
	require.NoError(t, EnsureDirs())
	assert.DirExists(t, filepath.Join(tmp, "toolchains"))
	assert.DirExists(t, filepath.Join(tmp, "bin"))
	assert.DirExists(t, filepath.Join(tmp, "downloads"))
}

// --- Tests merged from home_test.go ---

func TestHome_WithCJVHome(t *testing.T) {
	t.Setenv("CJV_HOME", "/custom/cjv")

	home, err := Home()
	require.NoError(t, err)
	assert.Contains(t, home, "custom")
	assert.Contains(t, home, "cjv")
}

func TestHome_FallbackToUserHome(t *testing.T) {
	t.Setenv("CJV_HOME", "")

	home, err := Home()
	require.NoError(t, err)
	assert.NotEmpty(t, home)
	assert.Contains(t, home, ".cjv")
}

// --- Tests merged from home_paths_test.go ---

func TestPathFunctions_ConstructCorrectPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	// SettingsPath() is intentionally decoupled from CJV_HOME — it always
	// lives under the OS user home so that the home path itself can be
	// persisted as a setting. Tests for that live in home_resolve_test.go.
	tests := []struct {
		name   string
		fn     func() (string, error)
		suffix string
	}{
		{"ToolchainsDir", ToolchainsDir, "toolchains"},
		{"BinDir", BinDir, "bin"},
		{"DownloadsDir", DownloadsDir, "downloads"},
		{"DocsDir", DocsDir, "docs"},
		{"StdxDir", StdxDir, "stdx"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fn()
			require.NoError(t, err)
			assert.Equal(t, filepath.Join(home, tt.suffix), got)
		})
	}
}

func TestPerToolchainPathFunctions(t *testing.T) {
	home := t.TempDir()
	t.Setenv(EnvHome, home)

	docs, err := DocsDirFor("lts-1.0.5")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "docs", "lts-1.0.5"), docs)

	stdx, err := StdxDirFor("lts-1.0.5")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "stdx", "lts-1.0.5"), stdx)
}

func TestDefaultSettingsFileCachesByResolvedPath(t *testing.T) {
	// SettingsPath() is now driven by the OS user home, so switch HOME (or
	// USERPROFILE on Windows) to vary the resolved path and confirm the
	// cache keys per resolved path rather than returning a stale instance.
	home1 := t.TempDir()
	home2 := t.TempDir()

	IsolateForTest(t, home1)
	sf1, err := DefaultSettingsFile()
	require.NoError(t, err)
	sf1Again, err := DefaultSettingsFile()
	require.NoError(t, err)
	assert.Same(t, sf1, sf1Again)

	IsolateForTest(t, home2)
	sf2, err := DefaultSettingsFile()
	require.NoError(t, err)
	assert.NotSame(t, sf1, sf2)
	assert.Equal(t, filepath.Join(home2, ".cjv", "settings.toml"), sf2.Path())
}

func TestResetCachedUserHomeDir(t *testing.T) {
	ResetCachedUserHomeDir()
	home, err := cachedUserHomeDir()
	require.NoError(t, err)
	assert.NotEmpty(t, home)
	ResetCachedUserHomeDir()
}

func TestLoadSettings_DefaultsWhenMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	path := filepath.Join(home, "settings.toml")
	settings, err := LoadSettings(path)
	require.NoError(t, err)
	assert.NotNil(t, settings)
}

func TestLoadSettings_LoadsExistingFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	// Write a settings file with a specific default
	s := DefaultSettings()
	s.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, SaveSettings(&s, filepath.Join(home, "settings.toml")))

	settings, err := LoadSettings(filepath.Join(home, "settings.toml"))
	require.NoError(t, err)
	assert.Equal(t, "lts-1.0.5", settings.DefaultToolchain)
}
