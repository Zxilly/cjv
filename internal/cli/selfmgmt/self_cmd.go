package selfmgmt

import (
	"fmt"
	"log/slog"
	"runtime"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/selfupdate"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var uninstallYes bool

// NewSelfCommand creates the "self" command with its subcommands.
// cleanCacheCmd can be nil if no cache cleanup subcommand is needed.
func NewSelfCommand(ver, updURL string, cleanCacheCmd *cobra.Command) *cobra.Command {
	selfCmd := &cobra.Command{
		Use:   "self",
		Short: "Manage the cjv installation itself",
	}

	selfUpdateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update cjv to the latest version",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := selfupdate.EnsureManagedExecutable(); err != nil {
				return err
			}
			selfupdate.CleanupOldBinaries()
			if err := selfupdate.Update(cmd.Context(), updURL, ver); err != nil {
				return err
			}
			return proxy.CreateAllProxyLinks()
		},
	}

	selfUninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall cjv and all installed toolchains",
		RunE: func(cmd *cobra.Command, args []string) error {
			confirm := uninstallYes
			if !confirm {
				err := huh.NewConfirm().
					Title(i18n.T("ConfirmUninstall", nil)).
					Value(&confirm).
					Run()
				if err != nil {
					return err
				}
			}
			if !confirm {
				return nil
			}

			home, err := config.Home()
			if err != nil {
				return err
			}
			managedExe, err := selfupdate.EnsureManagedExecutable()
			if err != nil {
				return err
			}

			// Remove PATH entries from shell configs (Unix)
			if runtime.GOOS != "windows" {
				posix, fish := env.ShellConfigPaths()
				for _, rc := range posix {
					if err := env.RemovePathFromShellConfig(rc); err != nil {
						slog.Warn("failed to clean PATH from shell config", "path", rc, "error", err)
					}
				}
				if fish != "" {
					if err := env.RemovePathFromShellConfig(fish); err != nil {
						slog.Warn("failed to clean PATH from shell config", "path", fish, "error", err)
					}
				}
			} else {
				// Windows: remove from registry
				binDir, err := config.BinDir()
				if err != nil {
					slog.Warn("failed to determine bin directory", "error", err)
				} else if err := env.RemovePathFromWindowsRegistry(binDir); err != nil {
					slog.Warn("failed to clean PATH from registry", "error", err)
				}
			}

			// Remove ~/.cjv/ (platform-specific: on Windows, a detached process
			// handles delayed deletion of the running binary).
			if err := removeHomeDir(home, managedExe); err != nil {
				return err
			}

			fmt.Println(i18n.T("UninstallComplete", nil))
			return nil
		},
	}

	selfUninstallCmd.Flags().BoolVarP(&uninstallYes, "yes", "y", false, "Skip confirmation prompt")
	selfCmd.AddCommand(selfUpdateCmd)
	selfCmd.AddCommand(selfUninstallCmd)
	if cleanCacheCmd != nil {
		selfCmd.AddCommand(cleanCacheCmd)
	}

	return selfCmd
}
