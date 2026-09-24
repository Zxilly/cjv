package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/selfmgmt"
	"github.com/Zxilly/cjv/internal/cli/settings"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
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

// Execute runs a fresh command tree and renders its error once.
func Execute(ver, updURL string) error {
	return newApplication(ver, updURL).execute(os.Args[1:])
}

func (app *application) execute(args []string) error {
	// Unknown commands fail during Cobra's lookup, before flag parsing. Honor
	// leading output flags for those errors without interpreting child arguments.
	for _, arg := range args {
		matched, enabled, err := parseJSONModeFlag(arg)
		if err != nil || !matched {
			break
		}
		app.output.SetJSONMode(enabled)
	}
	app.rootCmd.SetArgs(args)
	err := app.rootCmd.Execute()
	return app.output.RenderErrorTo(app.rootCmd.OutOrStdout(), app.rootCmd.ErrOrStderr(), err)
}

func (app *application) configureRoot() {
	app.rootCmd.Version = app.version
	app.rootCmd.SetVersionTemplate(color.CyanString("cjv {{.Version}}") + "\n")
	app.rootCmd.PersistentFlags().BoolVar(&app.output.JSON, "json", false, i18n.T("RootFlagJSON", nil))
	settings.RegisterCommands(app.rootCmd)
	app.rootCmd.AddCommand(selfmgmt.NewSelfCommand(app.version, app.updateURL, app.output))
	configureCobraHelp(app.rootCmd)
}

func (app *application) initRootCommands() {
	app.rootCmd = &cobra.Command{
		Use:           "cjv",
		Short:         i18n.T("RootCmdShort", nil),
		Long:          i18n.T("RootCmdLong", nil),
		SilenceUsage:  true,
		SilenceErrors: true,
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
			return app.output.RenderTo(cmdOutput(cmd), rootResult{Active: activeName, Installed: installed})
		},
	}
}
