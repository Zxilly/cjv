package runcmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clitest"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/resolve"
	"github.com/Zxilly/cjv/internal/sdktools"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for runWhich — shows the full path to a tool binary.

func TestRunWhichFindsTool(t *testing.T) {
	c := newTestCommands()
	home := t.TempDir()
	cwd := t.TempDir()
	configtest.Isolate(t, home)
	require.NoError(t, config.EnsureDirs())

	t.Chdir(cwd)

	// Install a toolchain
	server := testutil.ValidMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
	clitest.InstallToolchain(t, c.ctx, "lts")

	out, err := clitest.CommandOutput(t, func(cmd *cobra.Command) error {
		return c.runWhich(cmd, []string{"cjc"})
	})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "toolchains", "lts", "bin", sdktools.PlatformBinaryName("cjc")), strings.TrimSpace(out))
}

func TestRunWhichNoActiveToolchain(t *testing.T) {
	c := newTestCommands()
	home := t.TempDir()
	cwd := t.TempDir()
	configtest.Isolate(t, home)
	t.Setenv("CJV_TOOLCHAIN", "")

	t.Chdir(cwd)

	settings := config.DefaultSettings()
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	cmd := &cobra.Command{}
	err := c.runWhich(cmd, []string{"cjc"})
	assert.Error(t, err, "should error when no toolchain is active")
}

func TestRunWhichToolchainFileTargetsTriggerAutoInstall(t *testing.T) {
	c := newTestCommands()
	home := t.TempDir()
	cwd := t.TempDir()
	configtest.Isolate(t, home)
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

	cmd := &cobra.Command{}
	err := c.runWhich(cmd, []string{"cjc"})
	require.NoError(t, err)
	assert.Equal(t, "sts", gotInput)
	assert.Equal(t, []string{"ohos"}, gotTargets)
}
