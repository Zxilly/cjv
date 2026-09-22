package settings

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/Zxilly/cjv/internal/i18n"
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
	if len(args) == 0 {
		return showDefault(cmd.OutOrStdout())
	}

	name := args[0]

	sf, settings, err := LoadSettings()
	if err != nil {
		return err
	}

	if name == "none" {
		settings.DefaultToolchain = ""
		if err := sf.Save(settings); err != nil {
			return err
		}
		_, err := fmt.Fprintln(cmd.OutOrStdout(), i18n.T("DefaultCleared", nil))
		return err
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

	// Warn (but don't block) if the toolchain is not installed
	if _, findErr := toolchain.FindInstalled(parsed); findErr != nil {
		slog.Warn("toolchain is not installed", "name", normalizedName)
	}

	settings.DefaultToolchain = normalizedName
	if err := sf.Save(settings); err != nil {
		return err
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), i18n.T("ToolchainSetDefault", i18n.MsgData{
		"Name": normalizedName,
	}))
	return err
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
	_, settings, err := LoadSettings()
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
