package settings

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/Zxilly/cjv/internal/cli/output"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

func newDefaultCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "default [toolchain]",
		Short: i18n.T("DefaultCmdShort", nil),
		Long:  i18n.T("DefaultCmdLong", nil),
		Args:  cobra.MaximumNArgs(1),
		RunE:  runDefault,
	}
}

func runDefault(cmd *cobra.Command, args []string) error {
	if err := toolchain.RecoverHomeContext(cmd.Context()); err != nil {
		return err
	}
	jsonMode, _ := cmd.Flags().GetBool("json")
	quiet, _ := cmd.Flags().GetBool("quiet")
	renderer := &output.Renderer{JSON: jsonMode, Quiet: quiet}
	if len(args) == 0 {
		if jsonMode {
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
	parsed, err := toolchain.ParseToolchainName(name)
	if err != nil {
		return err
	}
	if err := ensureActiveToolchainName(name, parsed); err != nil {
		return err
	}
	normalizedName := parsed.String()

	if _, findErr := toolchain.FindInstalled(parsed); findErr != nil {
		if !errors.Is(findErr, os.ErrNotExist) {
			return findErr
		}
		sink := progress.Discard
		if !jsonMode && !quiet {
			sink = progress.NewText(cmd.OutOrStdout(), cmd.ErrOrStderr())
		}
		if err := lifecycle.Install(cmd.Context(), lifecycle.InstallRequest{Toolchain: normalizedName}, lifecycle.Options{Progress: sink}); err != nil {
			return err
		}
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

func ensureActiveToolchainName(input string, parsed toolchain.ToolchainName) error {
	if parsed.Target == "" {
		return nil
	}
	hostName := toolchain.ToolchainName{
		Channel: parsed.Channel,
		Version: parsed.Version,
	}.String()
	return fmt.Errorf("target variant %q cannot be used as an active toolchain; use host toolchain %q and configure targets instead", input, hostName)
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
