package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/Zxilly/cjv/internal/cjverr"
	clisettings "github.com/Zxilly/cjv/internal/cli/settings"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/Zxilly/cjv/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(uninstallCmd)
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <toolchain>",
	Short: "Uninstall a Cangjie SDK toolchain",
	Args:  cobra.ExactArgs(1),
	RunE:  runUninstall,
}

func runUninstall(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Validate name (path traversal, empty, +prefix, etc.)
	if _, err := toolchain.ParseToolchainName(name); err != nil {
		return err
	}

	tcDir, err := config.ToolchainsDir()
	if err != nil {
		return err
	}

	dir := filepath.Join(tcDir, name)
	if _, err := os.Stat(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &cjverr.ToolchainNotInstalledError{Name: name}
		}
		return err
	}

	// Update settings BEFORE removing the directory. If the process is killed
	// between these two steps, an orphaned directory is harmless (cleaned up
	// later), but dangling settings references would break all tool invocations.
	if err := updateSettingsAfterUninstall(name); err != nil {
		return err
	}

	if err := utils.RemoveAllRetry(dir); err != nil {
		return fmt.Errorf("failed to remove toolchain: %w", err)
	}

	color.Green(i18n.T("ToolchainUninstalled", i18n.MsgData{
		"Name": name,
	}))
	return nil
}

// updateSettingsAfterUninstall updates default toolchain and cleans up
// overrides that referenced the uninstalled toolchain.
func updateSettingsAfterUninstall(name string) error {
	sf, settings, err := clisettings.LoadSettings()
	if err != nil {
		return err
	}

	changed := false

	// Clean up overrides referencing the uninstalled toolchain
	for dir, tc := range settings.Overrides {
		if tc == name {
			delete(settings.Overrides, dir)
			changed = true
		}
	}

	// Update default if it pointed to the uninstalled toolchain
	if settings.DefaultToolchain == name {
		remaining, listErr := toolchain.ListInstalled()
		if listErr != nil {
			return fmt.Errorf("failed to list installed toolchains: %w", listErr)
		}
		newDefault := ""
		idx := slices.IndexFunc(remaining, func(r string) bool {
			return r != name
		})
		if idx >= 0 {
			newDefault = remaining[idx]
		}
		settings.DefaultToolchain = newDefault
		changed = true
	}

	if changed {
		return sf.Save(settings)
	}
	return nil
}
