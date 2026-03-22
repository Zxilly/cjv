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

	tests := []struct {
		name   string
		fn     func() (string, error)
		suffix string
	}{
		{"ToolchainsDir", ToolchainsDir, "toolchains"},
		{"BinDir", BinDir, "bin"},
		{"DownloadsDir", DownloadsDir, "downloads"},
		{"SettingsPath", SettingsPath, "settings.toml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fn()
			require.NoError(t, err)
			assert.Equal(t, filepath.Join(home, tt.suffix), got)
		})
	}
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
