package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/selfmgmt"
	"github.com/Zxilly/cjv/internal/cli/settings"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	updateURL string
	version   string
)

var rootCmd = &cobra.Command{
	Use:   "cjv",
	Short: "Cangjie SDK manager",
	Long:  "cjv is an SDK manager for the Cangjie programming language.",
	RunE: func(cmd *cobra.Command, args []string) error {
		installed, err := toolchain.ListInstalled()
		if err != nil {
			return err
		}
		if len(installed) == 0 {
			fmt.Println(i18n.T("RootNoToolchains", nil))
			return nil
		}

		_, activeName, _, resolveErr := toolchain.ResolveActiveToolchain()
		// Only ignore "no toolchain configured" — propagate real errors
		if resolveErr != nil && !errors.As(resolveErr, new(*cjverr.NoToolchainConfiguredError)) {
			slog.Warn("failed to resolve active toolchain", "error", resolveErr)
		}
		if activeName != "" {
			fmt.Println(i18n.T("RootActiveToolchain", i18n.MsgData{"Name": activeName}))
		}
		fmt.Println(i18n.TP("RootInstalledCount", i18n.MsgData{"Count": strconv.Itoa(len(installed))}, len(installed)))
		for _, name := range installed {
			marker := "  "
			if name == activeName {
				marker = "* "
			}
			fmt.Printf("  %s%s\n", marker, name)
		}
		return nil
	},
}

func Execute(ver, updURL string) error {
	version = ver
	updateURL = updURL
	rootCmd.Version = ver
	rootCmd.SetVersionTemplate(color.CyanString("cjv {{.Version}}") + "\n")

	settings.RegisterCommands(rootCmd)
	rootCmd.AddCommand(selfmgmt.NewSelfCommand(ver, updURL, cleanCacheCmd))

	return rootCmd.Execute()
}
