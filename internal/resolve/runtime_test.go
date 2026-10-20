package resolve_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/config/configtest"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/resolve"
	sdktarget "github.com/Zxilly/cjv/internal/target"
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

func TestActiveRuntimeEnv_DefaultToolchain(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	t.Setenv("CJV_TOOLCHAIN", "")

	setupFakeToolchain(t, home, "lts-1.0.5")

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	result, err := resolve.ActiveRuntimeEnv(context.Background(), "")
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// BuildProxyEnv always sets CJV_TOOLCHAIN to the resolved toolchain name
	val, ok := findEnvValue(result, "CJV_TOOLCHAIN")
	assert.True(t, ok, "CJV_TOOLCHAIN should be set")
	assert.Contains(t, val, "lts-1.0.5")
}

func TestActiveRuntimeEnv_WithOverride(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	t.Setenv("CJV_TOOLCHAIN", "")

	setupFakeToolchain(t, home, "sts-1.0.3")

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	result, err := resolve.ActiveRuntimeEnv(context.Background(), "sts-1.0.3")
	require.NoError(t, err)

	// Should use sts-1.0.3, not the default lts-1.0.5
	val, ok := findEnvValue(result, "CJV_TOOLCHAIN")
	assert.True(t, ok, "CJV_TOOLCHAIN should be set")
	assert.Contains(t, val, "sts-1.0.3")
}

func TestActiveRuntimeEnv_NoToolchain(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	t.Setenv("CJV_TOOLCHAIN", "")

	settings := config.DefaultSettings()
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	_, err := resolve.ActiveRuntimeEnv(context.Background(), "")
	assert.Error(t, err)
}

func TestActiveRuntimeEnv_ToolchainFileTargetsTriggerAutoInstall(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	configtest.Isolate(t, home)
	t.Setenv("CJV_TOOLCHAIN", "")
	t.Chdir(cwd)

	setupFakeToolchain(t, home, "sts-2.0.0")
	require.NoError(t, os.WriteFile(filepath.Join(cwd, config.ToolchainFileName), []byte(`[toolchain]
channel = "sts"
targets = ["ohos"]
`), 0o644))

	settings := config.DefaultSettings()
	settings.AutoInstall = true
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	oldAutoInstall := resolve.AutoInstallFunc
	var gotInput string
	var gotTargets []string
	resolve.AutoInstallFunc = func(ctx context.Context, input string, targets []string) error {
		gotInput = input
		gotTargets = append([]string(nil), targets...)
		key, err := sdktarget.CurrentTargetTuple(settings.DefaultHost, "ohos")
		require.NoError(t, err)
		name := toolchain.ToolchainName{Channel: toolchain.STS, Target: key}.String()
		require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", name), 0o755))
		parsedName, err := toolchain.ParseToolchainName(name)
		require.NoError(t, err)
		if parsedName.Version == "" && parsedName.Target != "" {
			parsedName.Version = "2.0.0"
			require.NoError(t, toolchain.WriteInstallation(filepath.Join(home, "toolchains", name), toolchain.Installation{Release: parsedName.String(), Tuple: parsedName.Target}))
		}
		return nil
	}
	t.Cleanup(func() { resolve.AutoInstallFunc = oldAutoInstall })

	_, err := resolve.ActiveRuntimeEnv(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, "sts", gotInput)
	assert.Equal(t, []string{"ohos"}, gotTargets)
}
