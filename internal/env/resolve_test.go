package env_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFakeToolchain(t *testing.T, home, name string) string {
	t.Helper()
	tcDir := filepath.Join(home, "toolchains", name)
	binDir := filepath.Join(tcDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))

	cjcName := "cjc"
	if runtime.GOOS == "windows" {
		cjcName = "cjc.exe"
	}
	require.NoError(t, os.WriteFile(filepath.Join(binDir, cjcName), []byte("stub"), 0o755))
	return tcDir
}

func TestResolveRuntimeEnv_DefaultToolchain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	t.Setenv("CJV_TOOLCHAIN", "")
	config.ResetDefaultSettingsFileCache()
	config.ResetCachedUserHomeDir()

	setupFakeToolchain(t, home, "lts-1.0.5")

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	result, err := env.ResolveRuntimeEnv(context.Background(), "")
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	found := false
	for _, e := range result {
		if len(e) > 5 && e[:5] == "PATH=" || (runtime.GOOS == "windows" && len(e) > 5 && (e[:5] == "Path=" || e[:5] == "path=")) {
			assert.Contains(t, e, "lts-1.0.5")
			found = true
			break
		}
	}
	assert.True(t, found, "PATH should contain toolchain directory")
}

func TestResolveRuntimeEnv_WithOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	t.Setenv("CJV_TOOLCHAIN", "")
	config.ResetDefaultSettingsFileCache()
	config.ResetCachedUserHomeDir()

	setupFakeToolchain(t, home, "sts-1.0.3")

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	result, err := env.ResolveRuntimeEnv(context.Background(), "sts-1.0.3")
	require.NoError(t, err)

	found := false
	for _, e := range result {
		if len(e) > 5 && e[:5] == "PATH=" || (runtime.GOOS == "windows" && len(e) > 5 && (e[:5] == "Path=" || e[:5] == "path=")) {
			assert.Contains(t, e, "sts-1.0.3")
			found = true
			break
		}
	}
	assert.True(t, found, "PATH should contain overridden toolchain directory")
}

func TestResolveRuntimeEnv_NoToolchain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	t.Setenv("CJV_TOOLCHAIN", "")
	config.ResetDefaultSettingsFileCache()
	config.ResetCachedUserHomeDir()

	settings := config.DefaultSettings()
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	_, err := env.ResolveRuntimeEnv(context.Background(), "")
	assert.Error(t, err)
}
