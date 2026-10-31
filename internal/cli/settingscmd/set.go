package settingscmd

import (
	"fmt"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/spf13/cobra"
)

func newSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: i18n.T("SetCmdShort", nil),
	}
	cmd.AddCommand(newSetAutoSelfUpdateCommand(), newSetAutoInstallCommand(), newSetDefaultHostCommand(), newSetHomeCommand())
	return cmd
}

func newSetAutoSelfUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:       "auto-self-update <enable|disable|check>",
		Short:     i18n.T("SetAutoSelfUpdateShort", nil),
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{config.AutoSelfUpdateEnable, config.AutoSelfUpdateDisable, config.AutoSelfUpdateCheck},
		RunE: func(cmd *cobra.Command, args []string) error {
			val := args[0]
			if !config.ValidAutoSelfUpdate(val) {
				return fmt.Errorf(i18n.T("SetInvalidAutoUpdate", nil), val)
			}
			return updateSetting(cmd.OutOrStdout(), "auto-self-update", val, config.SettingsUpdate{AutoSelfUpdate: &val})
		},
	}
}

func newSetAutoInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:       "auto-install <true|false>",
		Short:     i18n.T("SetAutoInstallShort", nil),
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"true", "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			val := args[0]
			if val != "true" && val != "false" {
				return fmt.Errorf(i18n.T("SetInvalidBool", nil), val)
			}
			newVal := val == "true"
			return updateSetting(cmd.OutOrStdout(), "auto-install", val, config.SettingsUpdate{AutoInstall: &newVal})
		},
	}
}

func newSetDefaultHostCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "default-host <goos-goarch>",
		Short: i18n.T("SetDefaultHostShort", nil),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			val := args[0]
			if _, err := sdktarget.CurrentHostTuple(val); err != nil {
				return fmt.Errorf(i18n.T("SetInvalidHost", nil), val, err)
			}
			return updateSetting(cmd.OutOrStdout(), "default-host", val, config.SettingsUpdate{DefaultHost: &val})
		},
	}
}
