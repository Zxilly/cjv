package cli

import (
	"errors"
	"fmt"
	"maps"
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

	sf, settings, err := clisettings.LoadSettings()
	if err != nil {
		return err
	}
	rollbackSettings := cloneSettings(settings)

	// Update settings BEFORE removing the directory. If the process is killed
	// between these two steps, an orphaned directory is harmless (cleaned up
	// later), but dangling settings references would break all tool invocations.
	// If deletion fails while this process is still running, roll settings back.
	if err := updateSettingsAfterUninstallLoaded(sf, settings, name); err != nil {
		return err
	}

	if err := utils.RemoveAllRetry(dir); err != nil {
		removeErr := fmt.Errorf("failed to remove toolchain: %w", err)
		if restoreErr := sf.Save(rollbackSettings); restoreErr != nil {
			return errors.Join(removeErr, fmt.Errorf("failed to restore settings: %w", restoreErr))
		}
		return removeErr
	}

	// Per-toolchain docs and stdx live outside the toolchain directory; nuke
	// them too so a fresh reinstall doesn't see stale extras.
	if docsDir, derr := config.DocsDirFor(name); derr == nil {
		_ = utils.RemoveAllRetry(docsDir) //nolint:errcheck // best-effort cleanup
	}
	if stdxDir, serr := config.StdxDirFor(name); serr == nil {
		_ = utils.RemoveAllRetry(stdxDir) //nolint:errcheck // best-effort cleanup
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
	return updateSettingsAfterUninstallLoaded(sf, settings, name)
}

func updateSettingsAfterUninstallLoaded(sf *config.SettingsFile, settings *config.Settings, name string) error {
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
		newDefault, listErr := nextDefaultAfterUninstall(name)
		if listErr != nil {
			return listErr
		}
		settings.DefaultToolchain = newDefault
		changed = true
	}

	if changed {
		return sf.Save(settings)
	}
	return nil
}

func nextDefaultAfterUninstall(name string) (string, error) {
	remaining, listErr := toolchain.ListInstalled()
	if listErr != nil {
		return "", fmt.Errorf("failed to list installed toolchains: %w", listErr)
	}
	idx := slices.IndexFunc(remaining, func(r string) bool {
		if r == name {
			return false
		}
		parsed, err := toolchain.ParseToolchainName(r)
		return err == nil && parsed.PlatformKey == ""
	})
	if idx < 0 {
		return "", nil
	}
	return remaining[idx], nil
}

func cloneSettings(settings *config.Settings) *config.Settings {
	cp := *settings
	cp.Overrides = maps.Clone(settings.Overrides)
	return &cp
}
