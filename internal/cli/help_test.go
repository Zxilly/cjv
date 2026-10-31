package cli

import (
	"bytes"
	"testing"
	"unicode"

	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureCobraHelpLocalizesInstallHelp(t *testing.T) {
	i18n.Init("zh-CN")
	t.Cleanup(func() { i18n.Init("en") })

	root := &cobra.Command{
		Use:   "cjv",
		Short: i18n.T("RootCmdShort", nil),
		Long:  i18n.T("RootCmdLong", nil),
	}
	root.PersistentFlags().Bool("json", false, i18n.T("RootFlagJSON", nil))

	install := &cobra.Command{
		Use:   "install <toolchain>",
		Short: i18n.T("InstallCmdShort", nil),
		Run:   func(cmd *cobra.Command, args []string) {},
	}
	install.Flags().StringSliceP("target", "t", nil, i18n.T("InstallFlagTarget", nil))
	root.AddCommand(install)

	configureCobraHelp(root)

	var stdout bytes.Buffer
	install.SetOut(&stdout)

	require.NoError(t, install.Help())
	help := stdout.String()

	assert.Contains(t, help, "安装仓颉 SDK 工具链")
	assert.Contains(t, help, "用法:")
	assert.Contains(t, help, "cjv install <toolchain> [选项]")
	assert.Contains(t, help, "选项:")
	assert.Contains(t, help, "显示 install 的帮助信息")
	assert.Contains(t, help, "需要附加安装的交叉编译目标后缀")
	assert.Contains(t, help, "全局选项:")
	assert.Contains(t, help, "输出机器可读的 JSON")

	assert.NotContains(t, help, "Usage:")
	assert.NotContains(t, help, "Flags:")
	assert.NotContains(t, help, "help for install")
	assert.NotContains(t, help, "Cross-compilation target")
}

// Check the assembled tree, including Cobra-generated commands, rather than
// just comparing catalogs: hard-coded English never appears in the catalogs.
func TestAllCommandHelpLocalized(t *testing.T) {
	t.Cleanup(func() { i18n.Init("en") })
	for _, lang := range []string{"zh-CN", "en"} {
		t.Run(lang, func(t *testing.T) {
			i18n.Init(lang)
			app := newApplication("dev", "")
			checkText := func(label, text string) {
				t.Helper()
				require.NotEmpty(t, text, label)
				hasChinese := false
				for _, r := range text {
					hasChinese = hasChinese || unicode.Is(unicode.Han, r)
				}
				assert.Equal(t, lang == "zh-CN", hasChinese, "%s: %s", label, text)
			}
			var walk func(*cobra.Command)
			walk = func(cmd *cobra.Command) {
				if cmd.Hidden {
					return
				}
				checkText(cmd.CommandPath()+" short", cmd.Short)
				if cmd.Long != "" {
					checkText(cmd.CommandPath()+" long", cmd.Long)
				}
				checkFlag := func(flag *pflag.Flag) {
					if !flag.Hidden {
						checkText(cmd.CommandPath()+" --"+flag.Name, flag.Usage)
					}
				}
				cmd.LocalFlags().VisitAll(checkFlag)
				cmd.InheritedFlags().VisitAll(checkFlag)
				var out bytes.Buffer
				cmd.SetOut(&out)
				require.NoError(t, cmd.Help())
				expected := cmd.Long
				if expected == "" {
					expected = cmd.Short
				}
				assert.Contains(t, out.String(), expected)
				for _, child := range cmd.Commands() {
					walk(child)
				}
			}
			walk(app.rootCmd)
		})
	}
}

func TestCompletionUsesCommandWriter(t *testing.T) {
	i18n.Init("zh-CN")
	t.Cleanup(func() { i18n.Init("en") })
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		for _, noDescriptions := range []bool{false, true} {
			app := newApplication("dev", "")
			var out bytes.Buffer
			app.rootCmd.SetOut(&out)
			args := []string{"completion", shell}
			if noDescriptions {
				args = append(args, "--no-descriptions")
			}
			require.NoError(t, app.execute(args))
			assert.Contains(t, out.String(), "cjv", shell)
		}
	}
}
