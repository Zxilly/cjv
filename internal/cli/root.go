// Package cli assembles cjv's command tree. Each command group lives in its
// own package and registers itself on the root; this package owns the root
// command, localized help, and Execute.
package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/cli/componentcmd"
	"github.com/Zxilly/cjv/internal/cli/initcmd"
	"github.com/Zxilly/cjv/internal/cli/runcmd"
	"github.com/Zxilly/cjv/internal/cli/selfmgmt"
	"github.com/Zxilly/cjv/internal/cli/settings"
	"github.com/Zxilly/cjv/internal/cli/showcmd"
	"github.com/Zxilly/cjv/internal/cli/toolchaincmd"
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

// application is one CLI execution: its shared context and command tree.
type application struct {
	ctx     *clictx.Context
	rootCmd *cobra.Command
}

func newApplication(ver, updURL string) *application {
	ctx := clictx.New(ver, updURL)
	root := newRootCommand(ctx)
	ctx.ConfigureRoot(root)
	for _, register := range []func(*cobra.Command, *clictx.Context){
		showcmd.Register,
		componentcmd.Register,
		runcmd.Register,
		initcmd.Register,
		toolchaincmd.Register,
	} {
		register(root, ctx)
	}
	settings.RegisterCommands(root)
	root.AddCommand(selfmgmt.NewSelfCommand(ver, updURL, ctx.Output))
	configureCobraHelp(root)
	return &application{ctx: ctx, rootCmd: root}
}

// Execute runs a fresh command tree and renders its error once.
func Execute(ver, updURL string) error {
	return newApplication(ver, updURL).execute(os.Args[1:])
}

func (app *application) execute(args []string) error {
	return app.ctx.Execute(args)
}

func newRootCommand(ctx *clictx.Context) *cobra.Command {
	root := &cobra.Command{
		Use:           "cjv",
		Short:         i18n.T("RootCmdShort", nil),
		Long:          i18n.T("RootCmdLong", nil),
		Version:       ctx.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			active, resolveErr := showcmd.GatherActive(cmd.Context(), ctx.Selector)
			installed, err := toolchain.ListInstalled()
			if err != nil {
				return err
			}
			activeName := ""
			if active != nil {
				activeName = active.Name
			}
			// Only ignore "no toolchain configured" — propagate real errors
			if resolveErr != nil && !errors.As(resolveErr, new(*cjverr.NoToolchainConfiguredError)) {
				slog.Warn("failed to resolve active toolchain", "error", resolveErr)
			}
			return ctx.Output.RenderTo(clictx.Out(cmd), rootResult{Active: activeName, Installed: installed})
		},
	}
	root.SetVersionTemplate(color.CyanString("cjv {{.Version}}") + "\n")
	return root
}
