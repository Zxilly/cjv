// Package cli assembles cjv's command tree. Each command group lives in its
// own package and registers itself on the root; this package owns the root
// command, localized help, and Execute.
package cli

import (
	"os"

	"github.com/Zxilly/cjv/internal/cli/cachecmd"
	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/cli/componentcmd"
	"github.com/Zxilly/cjv/internal/cli/initcmd"
	"github.com/Zxilly/cjv/internal/cli/projectcmd"
	"github.com/Zxilly/cjv/internal/cli/runcmd"
	"github.com/Zxilly/cjv/internal/cli/selfcmd"
	"github.com/Zxilly/cjv/internal/cli/settingscmd"
	"github.com/Zxilly/cjv/internal/cli/showcmd"
	"github.com/Zxilly/cjv/internal/cli/toolchaincmd"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

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
		cachecmd.Register,
		showcmd.Register,
		componentcmd.Register,
		runcmd.Register,
		initcmd.Register,
		projectcmd.Register,
		toolchaincmd.Register,
		settingscmd.Register,
		selfcmd.Register,
	} {
		register(root, ctx)
	}
	configureCobraHelp(root)
	configureCommandContracts(root)
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
			return showcmd.RunShow(ctx, cmd)
		},
	}
	root.SetVersionTemplate(color.CyanString("cjv {{.Version}}") + "\n")
	return root
}

func configureCommandContracts(root *cobra.Command) {
	// Commands with positional arguments declare their own validator. All other
	// runnable commands reject stray tokens before any operation can take place.
	var validate func(*cobra.Command)
	validate = func(cmd *cobra.Command) {
		if cmd.Args == nil && cmd.Runnable() {
			cmd.Args = cobra.NoArgs
		}
		for _, child := range cmd.Commands() {
			validate(child)
		}
	}
	validate(root)
	for _, group := range []struct{ id, key string }{
		{"toolchains", "HelpGroupToolchains"}, {"project", "HelpGroupProject"},
		{"execute", "HelpGroupExecute"}, {"inspect", "HelpGroupInspect"}, {"manage", "HelpGroupManage"},
	} {
		root.AddGroup(&cobra.Group{ID: group.id, Title: i18n.T(group.key, nil)})
	}
	for _, cmd := range root.Commands() {
		switch cmd.Name() {
		case "install", "update", "uninstall", "toolchain", "component", "target":
			cmd.GroupID = "toolchains"
		case "default", "override", "pin":
			cmd.GroupID = "project"
		case "exec", "run", "which", "envsetup":
			cmd.GroupID = "execute"
		case "show", "check", "doctor", "doc", "help":
			cmd.GroupID = "inspect"
		default:
			cmd.GroupID = "manage"
		}
	}
	root.SetHelpCommandGroupID("inspect")
	root.SetCompletionCommandGroupID("manage")
}
