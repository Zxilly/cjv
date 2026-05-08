package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/output"
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
	jsonFlag  bool
)

type rootResult struct {
	Active    string   `json:"active,omitempty"`
	Installed []string `json:"installed"`
}

func (r rootResult) Text() string {
	if len(r.Installed) == 0 {
		return i18n.T("RootNoToolchains", nil)
	}
	var b strings.Builder
	if r.Active != "" {
		b.WriteString(i18n.T("RootActiveToolchain", i18n.MsgData{"Name": r.Active}))
		b.WriteByte('\n')
	}
	b.WriteString(i18n.TP("RootInstalledCount", i18n.MsgData{"Count": strconv.Itoa(len(r.Installed))}, len(r.Installed)))
	b.WriteByte('\n')
	for _, name := range r.Installed {
		marker := "  "
		if name == r.Active {
			marker = "* "
		}
		fmt.Fprintf(&b, "  %s%s\n", marker, name)
	}
	return b.String()
}

var rootCmd = &cobra.Command{
	Use:           "cjv",
	Short:         "Cangjie SDK manager",
	Long:          "cjv is an SDK manager for the Cangjie programming language.",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		output.SetJSONMode(jsonFlag)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		installed, err := toolchain.ListInstalled()
		if err != nil {
			return err
		}
		_, activeName, _, resolveErr := toolchain.ResolveActiveToolchain()
		// Only ignore "no toolchain configured" — propagate real errors
		if resolveErr != nil && !errors.As(resolveErr, new(*cjverr.NoToolchainConfiguredError)) {
			slog.Warn("failed to resolve active toolchain", "error", resolveErr)
		}
		return output.RenderTo(cmdOutput(cmd), rootResult{Active: activeName, Installed: installed})
	},
}

func Execute(ver, updURL string) error {
	version = ver
	updateURL = updURL
	rootCmd.Version = ver
	rootCmd.SetVersionTemplate(color.CyanString("cjv {{.Version}}") + "\n")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "output machine-readable JSON")

	settings.RegisterCommands(rootCmd)
	rootCmd.AddCommand(selfmgmt.NewSelfCommand(ver, updURL, cleanCacheCmd))

	err := rootCmd.Execute()
	if err != nil {
		_ = output.RenderErrorTo(rootCmd.OutOrStdout(), rootCmd.ErrOrStderr(), err)
	}
	return err
}
