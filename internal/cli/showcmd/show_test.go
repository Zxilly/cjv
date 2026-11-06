package showcmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/clitest"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstalledListDistinguishesIdentityAndRelease(t *testing.T) {
	i18n.Init("en")
	t.Cleanup(func() { i18n.Init("en") })
	home := t.TempDir()
	configtest.Isolate(t, home)
	for _, name := range []string{"lts", "lts-1.0.5"} {
		require.NoError(t, toolchain.WriteInstallation(filepath.Join(home, "toolchains", name), toolchain.Installation{Release: "lts-1.0.5"}))
	}
	text, err := clitest.Execute(t, clitest.Root(Register), "show", "installed")
	require.NoError(t, err)
	assert.Contains(t, text, "lts → lts-1.0.5 (tracking)")
	assert.Contains(t, text, "lts-1.0.5 (pinned)")
	text, err = clitest.Execute(t, clitest.Root(Register), "show", "installed", "--json")
	require.NoError(t, err)
	var result InstalledResult
	require.NoError(t, json.Unmarshal([]byte(text), &result))
	require.Len(t, result.Toolchains, 2)
	assert.Equal(t, "lts", result.Toolchains[0].Name)
	assert.Equal(t, "lts-1.0.5", result.Toolchains[0].Release)
	assert.Equal(t, "tracking", result.Toolchains[0].Kind)
	assert.Equal(t, "pinned", result.Toolchains[1].Kind)
}

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
	i18n.Init("en")
	t.Cleanup(func() { i18n.Init("en") })
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

func TestShowStatusLocalized(t *testing.T) {
	t.Cleanup(func() { i18n.Init("en") })
	for _, lang := range []string{"zh-CN", "en"} {
		t.Run(lang, func(t *testing.T) {
			i18n.Init(lang)
			setDefault(t, "lts-1.0.5", false)
			out, err := clitest.Execute(t, clitest.Root(Register), "show", "active")
			require.NoError(t, err)
			label := "not installed"
			source := "source:"
			if lang == "zh-CN" {
				label, source = "未安装", "来源:"
				assert.NotContains(t, out, "not installed")
				assert.NotContains(t, out, "default toolchain")
				assert.Contains(t, out, "默认工具链")
			}
			assert.Contains(t, out, label)
			out, err = clitest.Execute(t, clitest.Root(Register), "show", "home")
			require.NoError(t, err)
			assert.Contains(t, out, source)
			out, err = clitest.Execute(t, clitest.Root(Register), "show", "active", "--json")
			require.NoError(t, err)
			var result ActiveResult
			require.NoError(t, json.Unmarshal([]byte(out), &result))
			assert.Equal(t, "default toolchain", result.Source)
			assert.False(t, result.Installed)
		})
	}
}

func TestShowSourceLabelsLocalized(t *testing.T) {
	i18n.Init("zh-CN")
	t.Cleanup(func() { i18n.Init("en") })
	for source, want := range map[string]string{
		"environment (CJV_TOOLCHAIN)": "环境变量 CJV_TOOLCHAIN",
		"directory override":          "目录覆盖", "default toolchain": "默认工具链",
		"command line": "命令行", "unknown": "未知", "cangjie-sdk.toml": "cangjie-sdk.toml",
		"env": "环境变量 CJV_HOME", "persisted": "用户设置", "default": "默认目录",
	} {
		assert.Equal(t, want, localizedSource(source), source)
	}
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
		out, err = clitest.Execute(t, clitest.Root(Register), "show", "installed", "--json")
		require.NoError(t, err)
		var result InstalledResult
		require.NoError(t, json.Unmarshal([]byte(out), &result))
		assert.NotNil(t, result.Toolchains)
		assert.Empty(t, result.Toolchains)
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
