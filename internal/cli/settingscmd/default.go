package settingscmd

import (
	"context"
	"fmt"
	"io"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

func newDefaultCommand(ctx *clictx.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "default [toolchain]",
		Short: i18n.T("DefaultCmdShort", nil),
		Long:  i18n.T("DefaultCmdLong", nil),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDefault(ctx, cmd, args)
		},
	}
}

func runDefault(ctx *clictx.Context, cmd *cobra.Command, args []string) error {
	if len(args) == 0 || args[0] == "none" {
		if err := toolchain.RecoverHomeContext(cmd.Context()); err != nil {
			return err
		}
	}
	renderer := ctx.Output
	if len(args) == 0 {
		if renderer.IsJSON() {
			_, settings, err := config.LoadDefaultSettings()
			if err != nil {
				return err
			}
			return renderer.RenderTo(cmd.OutOrStdout(), defaultResult{Toolchain: settings.DefaultToolchain})
		}
		return showDefault(cmd.OutOrStdout())
	}

	name := args[0]

	sf, err := config.DefaultSettingsFile()
	if err != nil {
		return err
	}

	if name == "none" {
		empty := ""
		if _, err := sf.Update(config.SettingsUpdate{DefaultToolchain: &empty}); err != nil {
			return err
		}
		return renderer.RenderTo(cmd.OutOrStdout(), defaultResult{action: "clear"})
	}

	// Validate and normalize toolchain name.
	// Accept both standard names (lts, sts-1.0) and custom/linked names (my-sdk).
	parsed, err := toolchain.ParseActiveName(name)
	if err != nil {
		return err
	}
	normalizedName := parsed.String()

	// Choosing a default installs a missing toolchain but, unlike install,
	// leaves PATH configuration alone.
	opts := lifecycle.Options{Progress: renderer.ProgressTo(cmd.OutOrStdout(), cmd.ErrOrStderr())}
	if _, _, err := toolchain.PrepareActive(cmd.Context(), normalizedName, func(ctx context.Context) error {
		return lifecycle.Install(ctx, lifecycle.InstallRequest{Toolchain: normalizedName}, opts)
	}); err != nil {
		return err
	}

	if _, err := sf.Update(config.SettingsUpdate{DefaultToolchain: &normalizedName}); err != nil {
		return err
	}

	return renderer.RenderTo(cmd.OutOrStdout(), defaultResult{Toolchain: normalizedName, action: "set"})
}

type defaultResult struct {
	Toolchain string `json:"default_toolchain"`
	action    string
}

func (r defaultResult) Text() string {
	if r.action == "clear" {
		return i18n.T("DefaultCleared", nil)
	}
	return i18n.T("ToolchainSetDefault", i18n.MsgData{"Name": r.Toolchain})
}

func showDefault(w io.Writer) error {
	_, settings, err := config.LoadDefaultSettings()
	if err != nil {
		return err
	}
	if settings.DefaultToolchain == "" {
		_, err := fmt.Fprintln(w, i18n.T("NoDefaultToolchain", nil))
		return err
	}
	_, err = fmt.Fprintln(w, i18n.T("CurrentDefault", i18n.MsgData{
		"Name": settings.DefaultToolchain,
	}))
	return err
}
