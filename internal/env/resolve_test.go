package env_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/resolve"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findEnvValue searches for a key in a []string{"KEY=val"} slice (case-insensitive on Windows).
func findEnvValue(envList []string, key string) (string, bool) {
	for _, e := range envList {
		k, v, ok := strings.Cut(e, "=")
		if !ok {
			continue
		}
		if runtime.GOOS == "windows" {
			if strings.EqualFold(k, key) {
				return v, true
			}
		} else if k == key {
			return v, true
		}
	}
	return "", false
}

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

	// BuildProxyEnv always sets CJV_TOOLCHAIN to the resolved toolchain name
	val, ok := findEnvValue(result, "CJV_TOOLCHAIN")
	assert.True(t, ok, "CJV_TOOLCHAIN should be set")
	assert.Contains(t, val, "lts-1.0.5")
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

	// Should use sts-1.0.3, not the default lts-1.0.5
	val, ok := findEnvValue(result, "CJV_TOOLCHAIN")
	assert.True(t, ok, "CJV_TOOLCHAIN should be set")
	assert.Contains(t, val, "sts-1.0.3")
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

func TestResolveRuntimeEnv_ToolchainFileTargetsTriggerAutoInstall(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("CJV_HOME", home)
	t.Setenv("CJV_TOOLCHAIN", "")
	config.ResetDefaultSettingsFileCache()
	config.ResetCachedUserHomeDir()
	t.Chdir(cwd)

	setupFakeToolchain(t, home, "sts-2.0.0")
	require.NoError(t, os.WriteFile(filepath.Join(cwd, config.ToolchainFileName), []byte(`[toolchain]
channel = "sts"
targets = ["ohos"]
`), 0o644))

	settings := config.DefaultSettings()
	settings.AutoInstall = true
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	oldAutoInstall := resolve.AutoInstallFunc
	var gotInput string
	var gotTargets []string
	resolve.AutoInstallFunc = func(ctx context.Context, input string, targets []string) error {
		gotInput = input
		gotTargets = append([]string(nil), targets...)
		key, err := dist.CurrentPlatformKeyWithTarget(settings.DefaultHost, "ohos")
		require.NoError(t, err)
		name := toolchain.ToolchainName{Channel: toolchain.STS, Version: "2.0.0", PlatformKey: key}.String()
		require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", name), 0o755))
		return nil
	}
	t.Cleanup(func() { resolve.AutoInstallFunc = oldAutoInstall })

	_, err := env.ResolveRuntimeEnv(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, "sts-2.0.0", gotInput)
	assert.Equal(t, []string{"ohos"}, gotTargets)
}
