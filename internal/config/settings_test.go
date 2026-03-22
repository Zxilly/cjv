package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsDefault(t *testing.T) {
	s := DefaultSettings()
	assert.Equal(t, "", s.DefaultToolchain)
	assert.Equal(t, DefaultManifestURL, s.ManifestURL)
	assert.Equal(t, "check", s.AutoSelfUpdate)
	assert.True(t, s.AutoInstall)
}

func TestSettingsRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "settings.toml")

	s := DefaultSettings()
	s.DefaultToolchain = "lts-1.0.5"
	s.Overrides = map[string]string{
		"/home/user/project": "nightly-1.1.0-alpha.20260306010001",
	}

	require.NoError(t, SaveSettings(&s, path))

	loaded, err := LoadSettings(path)
	require.NoError(t, err)
	assert.Equal(t, "lts-1.0.5", loaded.DefaultToolchain)
	assert.Equal(t, "nightly-1.1.0-alpha.20260306010001", loaded.Overrides["/home/user/project"])
	assert.Equal(t, DefaultManifestURL, loaded.ManifestURL)
}

func TestLoadSettingsMissing(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "nonexistent.toml")
	s, err := LoadSettings(path)
	require.NoError(t, err) // missing file returns defaults
	assert.Equal(t, DefaultSettings(), *s)
}

func TestLoadSettingsCreatesOverrideMap(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "settings.toml")

	// Write a settings file with no overrides section
	require.NoError(t, os.WriteFile(path, []byte(`default_toolchain = "lts"
manifest_url = "https://example.com"
auto_self_update = "check"
auto_install = true
`), 0o644))

	s, err := LoadSettings(path)
	require.NoError(t, err)
	assert.NotNil(t, s.Overrides)
}

// --- Tests merged from settings_edge_test.go ---

func TestLoadSettings_MalformedToml(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")
	require.NoError(t, os.WriteFile(path, []byte("[broken\nno closing"), 0o644))

	_, err := LoadSettings(path)
	assert.Error(t, err, "malformed TOML should return error")
}

func TestLoadSettings_EmptyManifestURLRestoredToDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")
	content := "manifest_url = \"\"\ndefault_toolchain = \"lts-1.0.5\"\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	s, err := LoadSettings(path)
	require.NoError(t, err)
	assert.NotEmpty(t, s.ManifestURL, "empty manifest_url should be restored to default")
	assert.Equal(t, "lts-1.0.5", s.DefaultToolchain)
}

func TestLoadSettings_NilOverridesInitialized(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")
	content := "default_toolchain = \"lts\"\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	s, err := LoadSettings(path)
	require.NoError(t, err)
	assert.NotNil(t, s.Overrides, "Overrides map should be initialized even when not in file")
}

func TestSave_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")
	s := DefaultSettings()
	s.DefaultToolchain = "sts-2.0.0"
	s.Overrides["C:\\project"] = "sts-2.0.0"
	require.NoError(t, SaveSettings(&s, path))

	loaded, err := LoadSettings(path)
	require.NoError(t, err)
	assert.Equal(t, "sts-2.0.0", loaded.DefaultToolchain)
	assert.Equal(t, "sts-2.0.0", loaded.Overrides["C:\\project"])
}

func TestLoadSettings_MissingVersion_DefaultsToV1(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")
	require.NoError(t, os.WriteFile(path, []byte("default_toolchain = \"lts-1.0.5\"\n"), 0o644))

	s, err := LoadSettings(path)
	require.NoError(t, err)
	assert.Equal(t, 1, s.Version)
}

func TestLoadSettings_FutureVersion_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")
	require.NoError(t, os.WriteFile(path, []byte("version = 999\n"), 0o644))

	_, err := LoadSettings(path)
	assert.Error(t, err, "future version should return error")
}

func TestSave_PersistsVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")
	s := DefaultSettings()
	require.NoError(t, SaveSettings(&s, path))

	loaded, err := LoadSettings(path)
	require.NoError(t, err)
	assert.Equal(t, 1, loaded.Version)
}

// --- Tests merged from input_validation_test.go (ValidAutoSelfUpdate) ---

func TestValidAutoSelfUpdate_AcceptsDefinedValues(t *testing.T) {
	assert.True(t, ValidAutoSelfUpdate("enable"))
	assert.True(t, ValidAutoSelfUpdate("disable"))
	assert.True(t, ValidAutoSelfUpdate("check"))
}

func TestValidAutoSelfUpdate_IsCaseSensitive(t *testing.T) {
	// TOML values are case-sensitive. "Enable" != "enable".
	for _, bad := range []string{"Enable", "DISABLE", "Check", "ENABLE"} {
		assert.False(t, ValidAutoSelfUpdate(bad), "%q should be rejected (case-sensitive)", bad)
	}
}

func TestValidAutoSelfUpdate_RejectsArbitraryStrings(t *testing.T) {
	for _, bad := range []string{"", "on", "off", "true", "false", "yes", "1", "enabled"} {
		assert.False(t, ValidAutoSelfUpdate(bad), "%q should be rejected", bad)
	}
}

// --- Tests merged from input_validation_test.go (LoadSettings via SettingsPath) ---

func TestLoadSettings_ReturnsDefaultsWhenNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)

	path, err := SettingsPath()
	require.NoError(t, err)
	settings, err := LoadSettings(path)
	require.NoError(t, err, "missing settings file should not be an error")
	assert.NotNil(t, settings, "should return defaults, not nil")
}

func TestLoadSettings_PathMatchesSettingsPath(t *testing.T) {
	// SettingsPath() should return a consistent path that points to settings.toml.
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)

	path, err := SettingsPath()
	require.NoError(t, err)
	assert.Contains(t, path, "settings.toml")
}
