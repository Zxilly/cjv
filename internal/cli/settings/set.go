package settings

import (
	"fmt"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/i18n"
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
				return fmt.Errorf("invalid value %q: must be enable, disable, or check", val)
			}
			return updateSetting(cmd.OutOrStdout(), "auto-self-update", val, func(s *config.Settings) bool {
				if s.AutoSelfUpdate == val {
					return false
				}
				s.AutoSelfUpdate = val
				return true
			})
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
				return fmt.Errorf("invalid value %q: must be true or false", val)
			}
			newVal := val == "true"
			return updateSetting(cmd.OutOrStdout(), "auto-install", val, func(s *config.Settings) bool {
				if s.AutoInstall == newVal {
					return false
				}
				s.AutoInstall = newVal
				return true
			})
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
			if _, err := dist.CurrentHostTuple(val); err != nil {
				return fmt.Errorf("invalid default-host %q: %w", val, err)
			}
			return updateSetting(cmd.OutOrStdout(), "default-host", val, func(s *config.Settings) bool {
				if s.DefaultHost == val {
					return false
				}
				s.DefaultHost = val
				return true
			})
		},
	}
}
