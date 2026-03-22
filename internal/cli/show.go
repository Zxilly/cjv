package cli

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show active and installed toolchains",
	RunE:  runShowDefault,
}

var showActiveCmd = &cobra.Command{
	Use:   "active",
	Short: "Show the active toolchain",
	RunE:  runShowActive,
}

var showInstalledCmd = &cobra.Command{
	Use:   "installed",
	Short: "List installed toolchains",
	RunE:  runShowInstalled,
}

var showHomeCmd = &cobra.Command{
	Use:   "home",
	Short: "Show CJV_HOME path",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := config.Home()
		if err != nil {
			return err
		}
		fmt.Println(home)
		return nil
	},
}

func runShowDefault(cmd *cobra.Command, args []string) error {
	// Show active + installed
	if err := runShowActive(cmd, args); err != nil {
		// Only ignore "no toolchain configured" — propagate real errors
		var noTC *cjverr.NoToolchainConfiguredError
		if errors.As(err, &noTC) {
			fmt.Fprintln(os.Stderr, err)
		} else {
			return err
		}
	}

	// Show default_host and profile from settings
	sf, sfErr := config.DefaultSettingsFile()
	var settings *config.Settings
	if sfErr == nil {
		settings, sfErr = sf.Load()
	}
	if settings != nil && sfErr == nil {
		host := settings.DefaultHost
		if host == "" {
			host = i18n.T("ShowDefaultHostAuto", nil)
		}
		fmt.Println(i18n.T("ShowDefaultHost", i18n.MsgData{"Host": host}))

		profile := settings.Profile
		if profile == "" {
			profile = i18n.T("ShowProfileDefault", nil)
		}
		fmt.Println(i18n.T("ShowProfile", i18n.MsgData{"Profile": profile}))
	}

	fmt.Println()
	return runShowInstalled(cmd, args)
}

func runShowActive(cmd *cobra.Command, args []string) error {
	_, name, source, err := toolchain.ResolveActiveToolchain()
	if err != nil {
		var notInstalled *cjverr.ToolchainNotInstalledError
		if errors.As(err, &notInstalled) {
			name = notInstalled.Name + " (not installed)"
		} else {
			return err
		}
	}
	fmt.Println(i18n.T("ActiveToolchain", i18n.MsgData{
		"Name":   name,
		"Source": source.String(),
	}))
	return nil
}

func runShowInstalled(cmd *cobra.Command, args []string) error {
	installed, err := toolchain.ListInstalled()
	if err != nil {
		return err
	}
	if len(installed) == 0 {
		fmt.Println(i18n.T("NoToolchainsInstalled", nil))
		return nil
	}
	fmt.Println(i18n.TP("InstalledToolchains", i18n.MsgData{
		"Count": strconv.Itoa(len(installed)),
	}, len(installed)))
	for _, name := range installed {
		fmt.Printf("  %s\n", name)
	}
	return nil
}

func init() {
	showCmd.AddCommand(showActiveCmd)
	showCmd.AddCommand(showInstalledCmd)
	showCmd.AddCommand(showHomeCmd)
	rootCmd.AddCommand(showCmd)
}
