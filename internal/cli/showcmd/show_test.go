package showcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/clitest"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setDefault isolates a home whose settings select name as the default.
func setDefault(t *testing.T, name string, installed bool) {
	t.Helper()
	home := t.TempDir()
	configtest.Isolate(t, home)
	t.Chdir(t.TempDir())
	if installed {
		require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", name), 0o755))
	}
	settings := config.DefaultSettings()
	settings.DefaultToolchain = name
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
}

func TestRunShowActive(t *testing.T) {
	t.Run("installed", func(t *testing.T) {
		setDefault(t, "lts-1.0.5", true)
		out, err := clitest.CommandOutput(t, func(cmd *cobra.Command) error {
			return newTestCommands().runShowActive(cmd, nil)
		})
		require.NoError(t, err)
		assert.Contains(t, out, "lts-1.0.5")
		assert.NotContains(t, out, "not installed")
	})
	t.Run("configured but not installed", func(t *testing.T) {
		setDefault(t, "lts-1.0.5", false)
		out, err := clitest.CommandOutput(t, func(cmd *cobra.Command) error {
			return newTestCommands().runShowActive(cmd, nil)
		})
		require.NoError(t, err, "uninstalled toolchains are shown with an annotation")
		assert.Contains(t, out, "lts-1.0.5 (not installed)")
	})
	t.Run("none configured", func(t *testing.T) {
		setDefault(t, "", false)
		t.Setenv("CJV_TOOLCHAIN", "")
		_, err := clitest.CommandOutput(t, func(cmd *cobra.Command) error {
			return newTestCommands().runShowActive(cmd, nil)
		})
		assert.Error(t, err, "should error when no toolchain is active")
	})
}

func TestRunShowInstalled(t *testing.T) {
	t.Run("lists toolchains", func(t *testing.T) {
		home := t.TempDir()
		configtest.Isolate(t, home)
		for _, name := range []string{"lts-1.0.5", "sts-2.0.0"} {
			require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", name), 0o755))
		}
		out, err := clitest.CommandOutput(t, func(cmd *cobra.Command) error {
			return RunShowInstalled(newTestCommands().ctx, cmd)
		})
		require.NoError(t, err)
		assert.Contains(t, out, "lts-1.0.5")
		assert.Contains(t, out, "sts-2.0.0")
	})
	t.Run("no toolchains", func(t *testing.T) {
		configtest.Isolate(t, t.TempDir())
		out, err := clitest.CommandOutput(t, func(cmd *cobra.Command) error {
			return RunShowInstalled(newTestCommands().ctx, cmd)
		})
		require.NoError(t, err, "an empty home is reported, not an error")
		assert.Contains(t, out, i18n.T("NoToolchainsInstalled", nil))
	})
}

func TestRunShowDefaultWithDefault(t *testing.T) {
	setDefault(t, "lts-1.0.5", true)
	out, err := clitest.CommandOutput(t, func(cmd *cobra.Command) error {
		return newTestCommands().runShowDefault(cmd, nil)
	})
	require.NoError(t, err)
	assert.Contains(t, out, "lts-1.0.5")
}

func TestRunShowDefaultNoToolchains(t *testing.T) {
	c := newTestCommands()
	configtest.Isolate(t, t.TempDir())

	var stdout, stderr bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	require.NoError(t, c.runShowDefault(cmd, nil))
	assert.Contains(t, stderr.String(), (&cjverr.NoToolchainConfiguredError{}).Error(),
		"the missing-default note reaches the command's err writer")
	assert.Contains(t, stdout.String(), i18n.T("NoToolchainsInstalled", nil))

	stdout.Reset()
	stderr.Reset()
	c.ctx.Output.SetJSONMode(true)
	require.NoError(t, c.runShowDefault(cmd, nil))
	assert.Empty(t, stderr.String(), "JSON mode keeps the note off stderr")
}
