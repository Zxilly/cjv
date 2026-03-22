package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFallback_UserUndefinedFieldsMergedFromFallback(t *testing.T) {
	tmp := t.TempDir()

	// User file: only sets manifest_url
	userPath := filepath.Join(tmp, "user.toml")
	require.NoError(t, os.WriteFile(userPath, []byte(`manifest_url = "https://user.example.com"`), 0o644))

	// Fallback file: sets default_toolchain, auto_install=false, auto_self_update="disable"
	fbPath := filepath.Join(tmp, "fallback.toml")
	require.NoError(t, os.WriteFile(fbPath, []byte(`
default_toolchain = "lts-1.0.0"
auto_install = false
auto_self_update = "disable"
`), 0o644))

	t.Setenv(EnvFallbackSettings, fbPath)

	s, _, err := LoadSettingsWithFallback(userPath)
	require.NoError(t, err)

	// User-defined field preserved
	assert.Equal(t, "https://user.example.com", s.ManifestURL)

	// Undefined fields filled from fallback
	assert.Equal(t, "lts-1.0.0", s.DefaultToolchain)
	assert.False(t, s.AutoInstall)
	assert.Equal(t, "disable", s.AutoSelfUpdate)
}

func TestFallback_UserExplicitBoolNotOverridden(t *testing.T) {
	tmp := t.TempDir()

	// User explicitly sets auto_install = false
	userPath := filepath.Join(tmp, "user.toml")
	require.NoError(t, os.WriteFile(userPath, []byte(`auto_install = false`), 0o644))

	// Fallback wants auto_install = true
	fbPath := filepath.Join(tmp, "fallback.toml")
	require.NoError(t, os.WriteFile(fbPath, []byte(`auto_install = true`), 0o644))

	t.Setenv(EnvFallbackSettings, fbPath)

	s, _, err := LoadSettingsWithFallback(userPath)
	require.NoError(t, err)

	// User's explicit false must NOT be overridden by fallback
	assert.False(t, s.AutoInstall)
}

func TestFallback_MissingFallbackFileSilentlySkipped(t *testing.T) {
	tmp := t.TempDir()

	userPath := filepath.Join(tmp, "user.toml")
	require.NoError(t, os.WriteFile(userPath, []byte(`default_toolchain = "lts"`), 0o644))

	// Point to nonexistent fallback
	t.Setenv(EnvFallbackSettings, filepath.Join(tmp, "nonexistent.toml"))

	s, _, err := LoadSettingsWithFallback(userPath)
	require.NoError(t, err)
	assert.Equal(t, "lts", s.DefaultToolchain)
}

func TestFallback_CorruptedFallbackFileDoesNotBlock(t *testing.T) {
	tmp := t.TempDir()

	userPath := filepath.Join(tmp, "user.toml")
	require.NoError(t, os.WriteFile(userPath, []byte(`default_toolchain = "lts"`), 0o644))

	// Corrupted fallback file
	fbPath := filepath.Join(tmp, "fallback.toml")
	require.NoError(t, os.WriteFile(fbPath, []byte(`[[[invalid toml`), 0o644))

	t.Setenv(EnvFallbackSettings, fbPath)

	s, _, err := LoadSettingsWithFallback(userPath)
	require.NoError(t, err)
	assert.Equal(t, "lts", s.DefaultToolchain)
}

func TestFallback_AllFieldsMerge(t *testing.T) {
	tmp := t.TempDir()

	// Empty user file
	userPath := filepath.Join(tmp, "user.toml")
	require.NoError(t, os.WriteFile(userPath, []byte(``), 0o644))

	// Fallback with all fields
	fbPath := filepath.Join(tmp, "fallback.toml")
	require.NoError(t, os.WriteFile(fbPath, []byte(`
default_toolchain = "nightly"
manifest_url = "https://fallback.example.com"
auto_self_update = "enable"
auto_install = false
`), 0o644))

	t.Setenv(EnvFallbackSettings, fbPath)

	s, _, err := LoadSettingsWithFallback(userPath)
	require.NoError(t, err)

	assert.Equal(t, "nightly", s.DefaultToolchain)
	assert.Equal(t, "https://fallback.example.com", s.ManifestURL)
	assert.Equal(t, "enable", s.AutoSelfUpdate)
	assert.False(t, s.AutoInstall)
}

func TestFallback_UserFileNotExistStillMerges(t *testing.T) {
	tmp := t.TempDir()

	// User file doesn't exist — should get defaults + fallback merge
	userPath := filepath.Join(tmp, "nonexistent_user.toml")

	fbPath := filepath.Join(tmp, "fallback.toml")
	require.NoError(t, os.WriteFile(fbPath, []byte(`
default_toolchain = "stable"
auto_install = false
`), 0o644))

	t.Setenv(EnvFallbackSettings, fbPath)

	s, _, err := LoadSettingsWithFallback(userPath)
	require.NoError(t, err)

	assert.Equal(t, "stable", s.DefaultToolchain)
	assert.False(t, s.AutoInstall)
}

func TestDefaultFallbackPath_EnvOverride(t *testing.T) {
	t.Setenv(EnvFallbackSettings, "/custom/path/settings.toml")
	assert.Equal(t, "/custom/path/settings.toml", DefaultFallbackPath())
}
