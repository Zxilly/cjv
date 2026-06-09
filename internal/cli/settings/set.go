package settings

import (
	"fmt"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/spf13/cobra"
)

var setCmd = &cobra.Command{
	Use:   "set",
	Short: i18n.T("SetCmdShort", nil),
}

var setAutoSelfUpdateCmd = &cobra.Command{
	Use:       "auto-self-update <enable|disable|check>",
	Short:     i18n.T("SetAutoSelfUpdateShort", nil),
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{config.AutoSelfUpdateEnable, config.AutoSelfUpdateDisable, config.AutoSelfUpdateCheck},
	RunE: func(cmd *cobra.Command, args []string) error {
		val := args[0]
		if !config.ValidAutoSelfUpdate(val) {
			return fmt.Errorf("invalid value %q: must be enable, disable, or check", val)
		}
		return updateSetting("auto-self-update", val, func(s *config.Settings) bool {
			if s.AutoSelfUpdate == val {
				return false
			}
			s.AutoSelfUpdate = val
			return true
		})
	},
}

var setAutoInstallCmd = &cobra.Command{
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
		return updateSetting("auto-install", val, func(s *config.Settings) bool {
			if s.AutoInstall == newVal {
				return false
			}
			s.AutoInstall = newVal
			return true
		})
	},
}

var setDefaultHostCmd = &cobra.Command{
	Use:   "default-host <goos-goarch>",
	Short: i18n.T("SetDefaultHostShort", nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		val := args[0]
		if _, err := dist.CurrentHostTuple(val); err != nil {
			return fmt.Errorf("invalid default-host %q: %w", val, err)
		}
		return updateSetting("default-host", val, func(s *config.Settings) bool {
			if s.DefaultHost == val {
				return false
			}
			s.DefaultHost = val
			return true
		})
	},
}

var setGitCodeAPIKeyCmd = &cobra.Command{
	Use:   "gitcode-api-key <key>",
	Short: i18n.T("SetGitCodeAPIKeyShort", nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		val := args[0]
		// Display a masked value: the token is a secret and updateSetting
		// echoes the display value to stdout (scrollback, CI logs, screen shares).
		return updateSetting("gitcode-api-key", maskSecret(val), func(s *config.Settings) bool {
			if s.GitCodeAPIKey == val {
				return false
			}
			s.GitCodeAPIKey = val
			return true
		})
	},
}

// maskSecret redacts a secret value for display, revealing neither its content
// nor its length.
func maskSecret(s string) string {
	if s == "" {
		return ""
	}
	return "********"
}
