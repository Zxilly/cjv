package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/output"
	clisettings "github.com/Zxilly/cjv/internal/cli/settings"
	"github.com/Zxilly/cjv/internal/config"
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

type updateEntry struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type updateResult struct {
	Updates       []updateEntry `json:"updates"`
	NoneInstalled bool          `json:"none_installed,omitempty"`
}

func (r updateResult) Text() string { return "" }

type updateOutcome struct {
	settingsFile  *config.SettingsFile
	settings      *config.Settings
	updates       []updateEntry
	noneInstalled bool
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	toolchain.CleanupStagingDirs()

	if len(args) == 1 {
		updates, err := updateSingle(ctx, args[0])
		if err != nil {
			return err
		}
		if !output.IsJSON() {
			return nil
		}
		return output.RenderTo(cmdOutput(cmd), updateResult{Updates: updates})
	}

	outcome, err := updateAll(ctx)

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
			if !output.IsJSON() {
				fmt.Printf("\n  cjv %s\n", version)
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
	if !output.IsJSON() {
		return nil
	}
	return output.RenderTo(cmdOutput(cmd), updateResult{
		Updates:       outcome.updates,
		NoneInstalled: outcome.noneInstalled,
	})
}

func updateSingle(ctx context.Context, input string) ([]updateEntry, error) {
	name, err := toolchain.ParseToolchainName(input)
	if err != nil {
		return nil, err
	}
	if name.IsCustom() {
		return nil, fmt.Errorf("cannot update custom toolchain '%s'", input)
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
		entry, updated, err := reinstallChannelForPlatform(ctx, reinstallRequest{
			Channel:      name.Channel,
			CurrentName:  currentName,
			Settings:     settings,
			SettingsFile: sf,
			Fetcher:      newManifestFetcher(settings.ManifestURL),
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
		entry, updated, err := reinstallChannel(ctx, name.Channel, installed, settings, sf, newManifestFetcher(settings.ManifestURL))
		return updateEntries(entry, updated), err
	}

	// Specific version — just install it (InstallToolchainWithOptions handles "already installed")
	return nil, InstallToolchainWithOptions(ctx, input, false)
}

func updateEntries(entry updateEntry, updated bool) []updateEntry {
	if !updated {
		return nil
	}
	return []updateEntry{entry}
}

func updateAll(ctx context.Context) (updateOutcome, error) {
	installed, err := toolchain.ListInstalled()
	if err != nil {
		return updateOutcome{}, err
	}
	if len(installed) == 0 {
		if !output.IsJSON() {
			fmt.Println(i18n.T("NoToolchainsInstalled", nil))
		}
		return updateOutcome{noneInstalled: true}, nil
	}

	sf, settings, err := clisettings.LoadSettings()
	if err != nil {
		return updateOutcome{}, err
	}
	outcome := updateOutcome{settingsFile: sf, settings: settings}

	fetcher := newManifestFetcher(settings.ManifestURL)
	var manifestWarnOnce sync.Once

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
		if parsed.Channel != toolchain.Nightly {
			if _, err := fetcher.get(ctx); err != nil {
				manifestWarnOnce.Do(func() {
					slog.Warn("manifest fetch failed; skipping non-nightly updates", "error", err)
				})
				continue
			}
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

		entry, updated, err := reinstallChannelForPlatform(ctx, reinstallRequest{
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
	Fetcher      *manifestFetcher
	Target       string
}

func reinstallChannel(ctx context.Context, channel toolchain.Channel, currentName string, settings *config.Settings, sf *config.SettingsFile, fetcher *manifestFetcher) (updateEntry, bool, error) {
	return reinstallChannelForPlatform(ctx, reinstallRequest{
		Channel:      channel,
		CurrentName:  currentName,
		Settings:     settings,
		SettingsFile: sf,
		Fetcher:      fetcher,
	})
}

func reinstallChannelForPlatform(ctx context.Context, req reinstallRequest) (updateEntry, bool, error) {
	resolved, err := resolveAndLocateWithTuple(ctx, toolchain.ToolchainName{Channel: req.Channel}, req.Settings, req.Fetcher, req.Target)
	if err != nil {
		return updateEntry{}, false, err
	}

	if resolved.Name == req.CurrentName {
		if !output.IsJSON() {
			color.Green(i18n.T("AlreadyUpToDate", i18n.MsgData{
				"Version": req.CurrentName,
			}))
		}
		return updateEntry{}, false, nil
	}

	noteStep(i18n.T("UpdateFound", i18n.MsgData{
		"Current": req.CurrentName,
		"Latest":  resolved.Name,
	}))
	update := updateEntry{From: req.CurrentName, To: resolved.Name}

	if req.Target == "" {
		if err := installResolved(ctx, resolved, req.Settings, req.SettingsFile, false); err != nil {
			return updateEntry{}, false, err
		}
	} else {
		if err := installResolvedNoDefault(ctx, resolved, req.Settings, req.SettingsFile, false); err != nil {
			return updateEntry{}, false, err
		}
	}

	if req.Target == "" {
		if req.Settings.DefaultToolchain == req.CurrentName {
			req.Settings.DefaultToolchain = resolved.Name
		}

		for dir, tc := range req.Settings.Overrides {
			if tc == req.CurrentName {
				req.Settings.Overrides[dir] = resolved.Name
			}
		}
	}

	if err := req.SettingsFile.Save(req.Settings); err != nil {
		return updateEntry{}, false, err
	}

	tcDir, err := config.ToolchainsDir()
	if err != nil {
		return updateEntry{}, false, err
	}
	oldDir := filepath.Join(tcDir, req.CurrentName)
	if err := utils.RemoveAllRetry(oldDir); err != nil {
		slog.Warn("failed to remove old toolchain", "name", req.CurrentName, "error", err)
		fmt.Fprintf(os.Stderr, "\n%s\n", i18n.T("OldToolchainRemoveWarning", i18n.MsgData{"Dir": oldDir}))
	}

	return update, true, nil
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
