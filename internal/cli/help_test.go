package cli

import (
	"bytes"
	"testing"

	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/spf13/cobra"
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
