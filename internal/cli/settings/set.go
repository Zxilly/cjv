package settings

import (
	"fmt"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/spf13/cobra"
)

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "Modify cjv settings",
}

var setAutoSelfUpdateCmd = &cobra.Command{
	Use:       "auto-self-update <enable|disable|check>",
	Short:     "Set auto-self-update behavior",
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
	Short:     "Set whether to auto-install missing toolchains in proxy mode",
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
	Short: "Set the default host platform triple",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		val := args[0]
		if _, err := dist.CurrentPlatformKey(val); err != nil {
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

var setProfileCmd = &cobra.Command{
	Use:   "profile <name>",
	Short: "Set the default installation profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		val := args[0]
		return updateSetting("profile", val, func(s *config.Settings) bool {
			if s.Profile == val {
				return false
			}
			s.Profile = val
			return true
		})
	},
}

var setGitCodeAPIKeyCmd = &cobra.Command{
	Use:   "gitcode-api-key <key>",
	Short: "Set the GitCode API access token for nightly builds",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		val := args[0]
		return updateSetting("gitcode-api-key", val, func(s *config.Settings) bool {
			if s.GitCodeAPIKey == val {
				return false
			}
			s.GitCodeAPIKey = val
			return true
		})
	},
}
