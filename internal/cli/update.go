package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/cjverr"
	clisettings "github.com/Zxilly/cjv/internal/cli/settings"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/selfupdate"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/Zxilly/cjv/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var noSelfUpdate bool

func init() {
	updateCmd.Flags().BoolVar(&noSelfUpdate, "no-self-update", false, "Don't check for cjv self-updates")
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update [toolchain]",
	Short: "Update installed toolchains",
	Long:  "Update a specific toolchain or all installed toolchains to their latest versions.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runUpdate,
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	toolchain.CleanupStagingDirs()

	if len(args) == 1 {
		return updateSingle(ctx, args[0])
	}

	sf, settings, err := updateAll(ctx)

	// Self-update check and cache cleanup run regardless of updateAll errors.
	// updateAll may partially succeed (some toolchains updated, others failed),
	// and we still want housekeeping to proceed. The original error is returned
	// at the end of this function.
	//
	// Reload settings from the cached SettingsFile to pick up mutations made
	// by reinstallChannel through this same SettingsFile instance
	// (e.g. default_toolchain changes). Note: this does not re-read from disk.
	if sf != nil {
		if reloaded, loadErr := sf.Load(); loadErr == nil {
			settings = reloaded
		}
	}
	if !noSelfUpdate && settings != nil && settings.AutoSelfUpdate != config.AutoSelfUpdateDisable {
		selfupdate.CleanupOldBinaries()

		switch settings.AutoSelfUpdate {
		case config.AutoSelfUpdateEnable:
			if _, err := selfupdate.EnsureManagedExecutable(); err != nil {
				slog.Warn("failed to bootstrap managed cjv binary for self-update", "error", err)
			} else if err := selfupdate.Update(ctx, updateURL, version); err != nil {
				slog.Warn("self-update failed", "error", err)
			} else if err := proxy.CreateAllProxyLinks(); err != nil {
				slog.Warn("failed to refresh proxies after self-update", "error", err)
			}
		default:
			if settings.AutoSelfUpdate != config.AutoSelfUpdateCheck {
				slog.Warn("unknown auto_self_update value, treating as check", "value", settings.AutoSelfUpdate)
			}
			fmt.Printf("\n  cjv %s\n", version)
		}
	}

	// Clean up download cache
	if n := cleanDownloadCache(); n > 0 {
		slog.Debug("cleaned download cache", "removed", n)
	}

	return err
}

func updateSingle(ctx context.Context, input string) error {
	name, err := toolchain.ParseToolchainName(input)
	if err != nil {
		return err
	}
	if name.IsCustom() {
		return fmt.Errorf("cannot update custom toolchain '%s'", input)
	}

	// If channel-only (e.g. "lts"), find the installed version for that channel
	if name.IsChannelOnly() {
		installed, err := findInstalledForChannel(name.Channel)
		if err != nil {
			return err
		}
		if installed == "" {
			return &cjverr.ToolchainNotInstalledError{Name: input}
		}
		sf, settings, err := clisettings.LoadSettings()
		if err != nil {
			return err
		}
		return reinstallChannel(ctx, name.Channel, installed, settings, sf, nil)
	}

	// Specific version — just install it (InstallToolchainWithOptions handles "already installed")
	return InstallToolchainWithOptions(ctx, input, false)
}

func updateAll(ctx context.Context) (*config.SettingsFile, *config.Settings, error) {
	installed, err := toolchain.ListInstalled()
	if err != nil {
		return nil, nil, err
	}
	if len(installed) == 0 {
		fmt.Println(i18n.T("NoToolchainsInstalled", nil))
		return nil, nil, nil
	}

	sf, settings, err := clisettings.LoadSettings()
	if err != nil {
		return nil, nil, err
	}

	// Lazily fetch manifest only if there are non-nightly channels to update.
	var manifest *dist.Manifest
	var manifestErr error

	var errs []error
	for _, name := range installed {
		parsed, err := toolchain.ParseToolchainName(name)
		if err != nil {
			slog.Warn("skipping toolchain", "name", name, "error", err)
			continue
		}
		if parsed.IsCustom() || parsed.Channel == toolchain.UnknownChannel {
			continue // skip custom/linked toolchains
		}
		if parsed.Channel != toolchain.Nightly && manifest == nil && manifestErr == nil {
			fmt.Println(i18n.T("FetchingManifest", nil))
			manifest, manifestErr = fetchManifest(ctx, settings.ManifestURL)
			if manifestErr != nil {
				slog.Warn("manifest fetch failed; skipping non-nightly updates", "error", manifestErr)
			}
		}
		if manifestErr != nil && parsed.Channel != toolchain.Nightly {
			continue // manifest unavailable, can't resolve non-nightly
		}

		// Reload settings from the cached SettingsFile so each iteration sees
		// the latest state saved by reinstallChannel through this same instance.
		// This does not re-read from disk.
		settings, err = sf.Load()
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err := reinstallChannel(ctx, parsed.Channel, name, settings, sf, manifest); err != nil {
			slog.Warn("failed to update toolchain", "name", name, "error", err)
			errs = append(errs, err)
		}
	}
	return sf, settings, errors.Join(errs...)
}

func reinstallChannel(ctx context.Context, channel toolchain.Channel, currentName string, settings *config.Settings, sf *config.SettingsFile, manifest *dist.Manifest) error {
	resolved, err := resolveAndLocate(ctx, toolchain.ToolchainName{Channel: channel}, settings, manifest)
	if err != nil {
		return err
	}

	if resolved.Name == currentName {
		color.Green(i18n.T("AlreadyUpToDate", i18n.MsgData{
			"Version": currentName,
		}))
		return nil
	}

	fmt.Println(i18n.T("UpdateFound", i18n.MsgData{
		"Current": currentName,
		"Latest":  resolved.Name,
	}))

	if err := installResolved(ctx, resolved, settings, sf, false); err != nil {
		return err
	}

	// Update default if it pointed to the old version
	if settings.DefaultToolchain == currentName {
		settings.DefaultToolchain = resolved.Name
	}

	// Update any overrides that referenced the old version
	for dir, tc := range settings.Overrides {
		if tc == currentName {
			settings.Overrides[dir] = resolved.Name
		}
	}

	if err := sf.Save(settings); err != nil {
		return err
	}

	tcDir, err := config.ToolchainsDir()
	if err != nil {
		return err
	}
	oldDir := filepath.Join(tcDir, currentName)
	if err := utils.RemoveAllRetry(oldDir); err != nil {
		slog.Warn("failed to remove old toolchain", "name", currentName, "error", err)
		fmt.Fprintf(os.Stderr, "\n%s\n", i18n.T("OldToolchainRemoveWarning", i18n.MsgData{"Dir": oldDir}))
	}

	return nil
}

func findInstalledForChannel(channel toolchain.Channel) (string, error) {
	// Use FindInstalled which performs proper semver-based sorting
	// to return the latest installed version for the channel.
	dir, err := toolchain.FindInstalled(toolchain.ToolchainName{Channel: channel})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return filepath.Base(dir), nil
}

