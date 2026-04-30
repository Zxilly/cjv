package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/resolve"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for runWhich — shows the full path to a tool binary.

func TestRunWhich_FindsTool(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	t.Chdir(cwd)

	// Install a toolchain
	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))
	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))

	cmd := &cobra.Command{}
	err := runWhich(cmd, []string{"cjc"})
	assert.NoError(t, err)
}

func TestRunWhich_NoActiveToolchain(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("CJV_HOME", home)
	t.Setenv("CJV_TOOLCHAIN", "")

	t.Chdir(cwd)

	settings := config.DefaultSettings()
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	cmd := &cobra.Command{}
	err := runWhich(cmd, []string{"cjc"})
	assert.Error(t, err, "should error when no toolchain is active")
}

func TestRunWhich_ToolchainFileTargetsTriggerAutoInstall(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("CJV_HOME", home)
	t.Setenv("CJV_TOOLCHAIN", "")
	require.NoError(t, config.EnsureDirs())
	t.Chdir(cwd)

	hostDir := filepath.Join(home, "toolchains", "sts-2.0.0")
	cjcPath := filepath.Join(hostDir, "bin", "cjc")
	if os.PathSeparator == '\\' {
		cjcPath += ".exe"
	}
	require.NoError(t, os.MkdirAll(filepath.Dir(cjcPath), 0o755))
	require.NoError(t, os.WriteFile(cjcPath, []byte("stub"), 0o755))
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

	cmd := &cobra.Command{}
	err := runWhich(cmd, []string{"cjc"})
	require.NoError(t, err)
	assert.Equal(t, "sts-2.0.0", gotInput)
	assert.Equal(t, []string{"ohos"}, gotTargets)
}
