package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/selfmgmt"
	clisettings "github.com/Zxilly/cjv/internal/cli/settings"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

type updateEntry struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type updateResult struct {
	Updates       []updateEntry          `json:"updates"`
	NoneInstalled bool                   `json:"none_installed,omitempty"`
	SelfUpdate    *selfmgmt.UpdateResult `json:"self_update,omitempty"`
}

func (r updateResult) Text() string { return "" }

type updateOutcome struct {
	settingsFile  *config.SettingsFile
	settings      *config.Settings
	updates       []updateEntry
	noneInstalled bool
}

func (app *application) runUpdate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	toolchain.CleanupStagingDirs()

	if len(args) == 1 {
		updates, err := app.updateSingle(ctx, args[0])
		if err != nil {
			return err
		}
		if !app.output.IsJSON() {
			return nil
		}
		return app.output.RenderTo(cmdOutput(cmd), updateResult{Updates: updates})
	}

	outcome, err := app.updateAll(ctx)

	// Self-update check and cache cleanup run regardless of updateAll errors.
	// updateAll may partially succeed (some toolchains updated, others failed),
	// and we still want housekeeping to proceed. The original error is returned
	// at the end of this function.
	//
	// Reload settings from the cached SettingsFile to pick up mutations made
	// by reinstallChannel through this same SettingsFile instance
	// (e.g. default_toolchain changes). Note: this does not re-read from disk.
	settings := outcome.settings
	if outcome.settingsFile != nil {
		if reloaded, loadErr := outcome.settingsFile.Load(); loadErr == nil {
			settings = reloaded
		}
	}
	var selfUpdate *selfmgmt.UpdateResult
	if !app.noSelfUpdate && settings != nil && settings.AutoSelfUpdate != config.AutoSelfUpdateDisable {
		switch settings.AutoSelfUpdate {
		case config.AutoSelfUpdateEnable:
			selfResult, selfErr := selfmgmt.UpdateManaged(ctx, app.updateURL, app.version)
			if selfResult.Status != "" {
				selfUpdate = &selfResult
			}
			if selfErr != nil {
				slog.Warn("self-update failed", "error", selfErr)
			} else {
				if !app.output.IsJSON() {
					if renderErr := app.output.RenderTo(cmdOutput(cmd), selfResult); renderErr != nil {
						return renderErr
					}
				}
			}

		default:
			if settings.AutoSelfUpdate != config.AutoSelfUpdateCheck {
				slog.Warn("unknown auto_self_update value, treating as check", "value", settings.AutoSelfUpdate)
			}
			if !app.output.IsJSON() {
				fmt.Printf("\n  cjv %s\n", app.version)
			}
		}
	}

	if n, purgeErr := purgeDownloadsDir(); purgeErr != nil {
		slog.Warn("failed to purge downloads dir", "removed", n, "error", purgeErr)
	} else if n > 0 {
		slog.Debug("purged downloads dir", "removed", n)
	}

	if err != nil {
		return err
	}
	if !app.output.IsJSON() {
		return nil
	}
	return app.output.RenderTo(cmdOutput(cmd), updateResult{
		Updates:       outcome.updates,
		NoneInstalled: outcome.noneInstalled,
		SelfUpdate:    selfUpdate,
	})
}

func (app *application) updateSingle(ctx context.Context, input string) ([]updateEntry, error) {
	name, err := toolchain.ParseToolchainName(input)
	if err != nil {
		return nil, err
	}
	if name.IsCustom() {
		return nil, errors.New(i18n.T("UpdateCustomToolchain", i18n.MsgData{"Name": input}))
	}
	if name.Target != "" {
		currentName := name.String()
		if _, err := toolchain.FindInstalled(name); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, &cjverr.ToolchainNotInstalledError{Name: currentName}
			}
			return nil, err
		}
		sf, settings, err := clisettings.LoadSettings()
		if err != nil {
			return nil, err
		}
		fetcher, err := app.newManifestFetcherForSettings(settings)
		if err != nil {
			return nil, err
		}
		entry, updated, err := app.reinstallChannelForPlatform(ctx, reinstallRequest{
			Channel:      name.Channel,
			CurrentName:  currentName,
			Settings:     settings,
			SettingsFile: sf,
			Fetcher:      fetcher,
			Target:       name.Target,
		})
		return updateEntries(entry, updated), err
	}

	// If channel-only (e.g. "lts"), find the installed version for that channel
	if name.IsChannelOnly() {
		installed, err := findInstalledForChannel(name.Channel)
		if err != nil {
			return nil, err
		}
		if installed == "" {
			return nil, &cjverr.ToolchainNotInstalledError{Name: input}
		}
		sf, settings, err := clisettings.LoadSettings()
		if err != nil {
			return nil, err
		}
		fetcher, err := app.newManifestFetcherForSettings(settings)
		if err != nil {
			return nil, err
		}
		entry, updated, err := app.reinstallChannel(ctx, name.Channel, installed, settings, sf, fetcher)
		return updateEntries(entry, updated), err
	}

	// Specific version — just install it (InstallToolchainWithOptions handles "already installed")
	return nil, app.InstallToolchainWithOptions(ctx, input, false)
}

func updateEntries(entry updateEntry, updated bool) []updateEntry {
	if !updated {
		return nil
	}
	return []updateEntry{entry}
}

func (app *application) updateAll(ctx context.Context) (updateOutcome, error) {
	installed, err := toolchain.ListInstalled()
	if err != nil {
		return updateOutcome{}, err
	}
	if len(installed) == 0 {
		if !app.output.IsJSON() {
			fmt.Println(i18n.T("NoToolchainsInstalled", nil))
		}
		return updateOutcome{noneInstalled: true}, nil
	}

	sf, settings, err := clisettings.LoadSettings()
	if err != nil {
		return updateOutcome{}, err
	}
	outcome := updateOutcome{settingsFile: sf, settings: settings}

	fetcher, err := app.newManifestFetcherForSettings(settings)
	if err != nil {
		return updateOutcome{}, err
	}
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
		// Reload settings from the cached SettingsFile so each iteration sees
		// the latest state saved by reinstallChannel through this same instance.
		// This does not re-read from disk.
		settings, err = sf.Load()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		outcome.settings = settings

		entry, updated, err := app.reinstallChannelForPlatform(ctx, reinstallRequest{
			Channel:      parsed.Channel,
			CurrentName:  name,
			Settings:     settings,
			SettingsFile: sf,
			Fetcher:      fetcher,
			Target:       parsed.Target,
		})
		if err != nil {
			slog.Warn("failed to update toolchain", "name", name, "error", err)
			errs = append(errs, err)
			continue
		}
		if updated {
			outcome.updates = append(outcome.updates, entry)
		}
	}
	return outcome, errors.Join(errs...)
}

type reinstallRequest struct {
	Channel      toolchain.Channel
	CurrentName  string
	Settings     *config.Settings
	SettingsFile *config.SettingsFile
	Fetcher      *lifecycle.ManifestFetcher
	Target       string
}

func (app *application) reinstallChannel(ctx context.Context, channel toolchain.Channel, currentName string, settings *config.Settings, sf *config.SettingsFile, fetcher *lifecycle.ManifestFetcher) (updateEntry, bool, error) {
	return app.reinstallChannelForPlatform(ctx, reinstallRequest{
		Channel:      channel,
		CurrentName:  currentName,
		Settings:     settings,
		SettingsFile: sf,
		Fetcher:      fetcher,
	})
}

func (app *application) reinstallChannelForPlatform(ctx context.Context, req reinstallRequest) (updateEntry, bool, error) {
	resolved, err := resolveAndLocate(ctx, toolchain.ToolchainName{Channel: req.Channel}, req.Settings, req.Fetcher, req.Target)
	if err != nil {
		return updateEntry{}, false, err
	}

	updated, err := lifecycle.UpgradeToolchain(ctx, req.CurrentName, resolved, req.SettingsFile, req.Fetcher, app.lifecycleOptions())
	return updateEntry{From: req.CurrentName, To: resolved.Name}, updated, err
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

func (app *application) initUpdateCommands() {
	app.updateCmd = &cobra.Command{
		Use:   "update [toolchain]",
		Short: i18n.T("UpdateCmdShort", nil),
		Long:  i18n.T("UpdateCmdLong", nil),
		Args:  cobra.MaximumNArgs(1),
		RunE:  app.runUpdate,
	}

	app.updateCmd.Flags().BoolVar(&app.noSelfUpdate, "no-self-update", false, i18n.T("UpdateFlagNoSelfUpdate", nil))
	app.rootCmd.AddCommand(app.updateCmd)

}
