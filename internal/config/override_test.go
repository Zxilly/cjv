package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveOverrideEnvWins(t *testing.T) {
	t.Setenv(EnvToolchain, "nightly")
	s := DefaultSettings()
	s.DefaultToolchain = "lts-1.0.5"

	result, source, err := ResolveToolchain(&s, "/some/dir")
	require.NoError(t, err)
	assert.Equal(t, "nightly", result)
	assert.Equal(t, SourceEnv, source)
}

func TestResolveOverrideDirectoryOverride(t *testing.T) {
	t.Setenv(EnvToolchain, "")
	tmp := t.TempDir()
	s := DefaultSettings()
	s.DefaultToolchain = "lts-1.0.5"
	s.Overrides[NormalizePath(tmp)] = "sts-1.1.0-beta.23"

	result, source, err := ResolveToolchain(&s, tmp)
	require.NoError(t, err)
	assert.Equal(t, "sts-1.1.0-beta.23", result)
	assert.Equal(t, SourceOverride, source)
}

func TestResolveOverrideToolchainFile(t *testing.T) {
	t.Setenv(EnvToolchain, "")
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ToolchainFileName), []byte(`[toolchain]
channel = "nightly-1.1.0-alpha.20260306010001"
`), 0o644)

	s := DefaultSettings()
	s.DefaultToolchain = "lts-1.0.5"

	result, source, err := ResolveToolchain(&s, dir)
	require.NoError(t, err)
	assert.Equal(t, "nightly-1.1.0-alpha.20260306010001", result)
	assert.Equal(t, SourceToolchainFile, source)
}

func TestResolveOverrideDefault(t *testing.T) {
	t.Setenv(EnvToolchain, "")
	tmp := t.TempDir()
	s := DefaultSettings()
	s.DefaultToolchain = "lts-1.0.5"

	result, source, err := ResolveToolchain(&s, tmp)
	require.NoError(t, err)
	assert.Equal(t, "lts-1.0.5", result)
	assert.Equal(t, SourceDefault, source)
}

func TestResolveOverrideNoneConfigured(t *testing.T) {
	t.Setenv(EnvToolchain, "")
	tmp := t.TempDir()
	s := DefaultSettings()

	_, _, err := ResolveToolchain(&s, tmp)
	assert.Error(t, err) // no toolchain configured
}

// --- Tests merged from resolve_edge_test.go ---

func TestResolveToolchain_EnvVarWins(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("CJV_HOME", home)

	settings := DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.0"

	t.Setenv("CJV_TOOLCHAIN", "sts-2.0.0")

	name, source, err := ResolveToolchain(&settings, cwd)
	require.NoError(t, err)
	assert.Equal(t, "sts-2.0.0", name)
	assert.Equal(t, SourceEnv, source)
}

func TestResolveToolchain_DirectoryOverride(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("CJV_HOME", home)

	t.Setenv("CJV_TOOLCHAIN", "")

	normalizedCwd := NormalizePath(cwd)

	settings := DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.0"
	settings.Overrides[normalizedCwd] = "sts-2.0.0"

	name, source, err := ResolveToolchain(&settings, cwd)
	require.NoError(t, err)
	assert.Equal(t, "sts-2.0.0", name)
	assert.Equal(t, SourceOverride, source)
}

func TestResolveToolchain_ToolchainFile(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("CJV_HOME", home)

	t.Setenv("CJV_TOOLCHAIN", "")

	content := "[toolchain]\nchannel = \"sts\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(cwd, "cangjie-sdk.toml"), []byte(content), 0o644))

	settings := DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.0"

	name, source, err := ResolveToolchain(&settings, cwd)
	require.NoError(t, err)
	assert.Equal(t, "sts", name)
	assert.Equal(t, SourceToolchainFile, source)
}

func TestNormalizePath_MakesAbsolute(t *testing.T) {
	normalized := NormalizePath(".")
	assert.True(t, filepath.IsAbs(normalized))
}

func TestEnsureDirs_CreatesAll(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	require.NoError(t, EnsureDirs())

	assert.DirExists(t, filepath.Join(home, "toolchains"))
	assert.DirExists(t, filepath.Join(home, "bin"))
	assert.DirExists(t, filepath.Join(home, "downloads"))
}

// --- Tests merged from input_validation_test.go (OverrideSource.String) ---

func TestOverrideSourceString_AllSourcesHaveDistinctLabels(t *testing.T) {
	tests := []struct {
		source OverrideSource
		want   string
	}{
		{SourceEnv, "environment (CJV_TOOLCHAIN)"},
		{SourceOverride, "directory override"},
		{SourceToolchainFile, "cangjie-sdk.toml"},
		{SourceDefault, "default toolchain"},
		{SourceUnknown, "unknown"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.source.String(),
			"wrong label for OverrideSource %d", tt.source)
	}
}

func TestOverrideSourceString_UndefinedValueFallsToUnknown(t *testing.T) {
	// If new sources are added but String() isn't updated,
	// they should fall through to "unknown" rather than panic.
	assert.Equal(t, "unknown", OverrideSource(99).String())
}
