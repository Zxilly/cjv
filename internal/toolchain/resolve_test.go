package toolchain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for ResolveActiveToolchain — determines which toolchain the
// user wants to use based on env var, overrides, toolchain file, or default.

func TestResolveActiveToolchain_UsesDefault(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("CJV_HOME", home)

	t.Setenv("CJV_TOOLCHAIN", "")

	t.Chdir(cwd)

	// Create an installed toolchain and set it as default
	tcDir := filepath.Join(home, "toolchains", "lts-1.0.5")
	require.NoError(t, os.MkdirAll(tcDir, 0o755))

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	dir, name, source, err := ResolveActiveToolchain()
	require.NoError(t, err)
	assert.Contains(t, dir, "lts-1.0.5")
	assert.Equal(t, "lts-1.0.5", name)
	assert.Equal(t, config.SourceDefault, source)
}

func TestResolveActiveToolchain_EnvVarOverridesDefault(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("CJV_HOME", home)

	t.Chdir(cwd)

	// Create two toolchains
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.5"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "sts-2.0.0"), 0o755))

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	// CJV_TOOLCHAIN env var should override the default
	t.Setenv("CJV_TOOLCHAIN", "sts-2.0.0")

	_, name, source, err := ResolveActiveToolchain()
	require.NoError(t, err)
	assert.Equal(t, "sts-2.0.0", name,
		"env var should override default toolchain")
	assert.Equal(t, config.SourceEnv, source)
}

func TestResolveActiveToolchain_NoConfig(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("CJV_HOME", home)

	t.Setenv("CJV_TOOLCHAIN", "")

	t.Chdir(cwd)

	// No default, no env var, no toolchain file
	settings := config.DefaultSettings()
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	_, _, _, err := ResolveActiveToolchain()
	assert.Error(t, err, "should error when no toolchain is configured")
}

// --- Tests merged from semver_test.go ---

// Tests for compareSemVer -- ensures correct version ordering across
// standard semver, pre-release, and non-standard version strings.

func TestCompareSemVer_StandardVersions(t *testing.T) {
	assert.Equal(t, 0, compareSemVer("1.0.0", "1.0.0"))
	assert.Equal(t, -1, compareSemVer("1.0.0", "2.0.0"))
	assert.Equal(t, 1, compareSemVer("2.0.0", "1.0.0"))
	assert.Equal(t, -1, compareSemVer("1.0.0", "1.1.0"))
	assert.Equal(t, -1, compareSemVer("1.0.0", "1.0.1"))
}

func TestCompareSemVer_PreRelease(t *testing.T) {
	// Pre-release versions come before release
	assert.Equal(t, -1, compareSemVer("1.0.0-beta.1", "1.0.0"))
	assert.Equal(t, 1, compareSemVer("1.0.0", "1.0.0-beta.1"))
}

func TestCompareSemVer_OneValidOneInvalid(t *testing.T) {
	// A valid semver sorts after an invalid string
	assert.Equal(t, 1, compareSemVer("1.0.0", "not-a-version"))
	assert.Equal(t, -1, compareSemVer("not-a-version", "1.0.0"))
}

func TestCompareSemVer_BothInvalid(t *testing.T) {
	// Falls back to lexicographic string comparison
	assert.Equal(t, -1, compareSemVer("aaa", "bbb"))
	assert.Equal(t, 1, compareSemVer("bbb", "aaa"))
	assert.Equal(t, 0, compareSemVer("same", "same"))
}
