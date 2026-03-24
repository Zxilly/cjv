package cli

import (
	"errors"
	"fmt"

	"github.com/Zxilly/cjv/internal/cjverr"
	clisettings "github.com/Zxilly/cjv/internal/cli/settings"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for available updates without installing",
	RunE:  runCheck,
}

func runCheck(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	installed, err := toolchain.ListInstalled()
	if err != nil {
		return err
	}
	if len(installed) == 0 {
		fmt.Println(i18n.T("NoToolchainsInstalled", nil))
		return nil
	}

	_, settings, err := clisettings.LoadSettings()
	if err != nil {
		return err
	}

	// Pre-fetch latest nightly once if any nightly toolchain is installed
	var latestNightly string
	var nightlyErr error
	for _, name := range installed {
		parsed, parseErr := toolchain.ParseToolchainName(name)
		if parseErr == nil && parsed.Channel == toolchain.Nightly {
			latestNightly, nightlyErr = dist.FetchLatestNightly(ctx, dist.DefaultNightlyAPIURL, settings.GitCodeAPIKey)
			break
		}
	}

	platformKey, err := dist.CurrentPlatformKey(settings.DefaultHost)
	if err != nil {
		return err
	}

	// Lazy-fetch manifest only when a non-nightly toolchain needs it
	var manifest *dist.Manifest
	var manifestErr error

	hasUpdates := false
	for _, name := range installed {
		parsed, err := toolchain.ParseToolchainName(name)
		if err != nil || parsed.IsCustom() || parsed.Channel == toolchain.UnknownChannel {
			continue
		}

		if parsed.Channel == toolchain.Nightly {
			if nightlyErr != nil {
				fmt.Printf("  %s: %s\n", name, nightlyErr)
				continue
			}
			latestName := toolchain.ToolchainName{Channel: toolchain.Nightly, Version: latestNightly}.String()
			if latestName != name {
				color.Yellow("  %s → %s", name, latestName)
				hasUpdates = true
			} else {
				color.Green("  %s ✓", name)
			}
			continue
		}

		if manifest == nil && manifestErr == nil {
			manifest, manifestErr = fetchManifest(ctx, settings.ManifestURL)
		}
		if manifestErr != nil {
			return manifestErr
		}

		latest, err := manifest.GetLatestVersion(parsed.Channel)
		if err != nil {
			continue
		}

		latestName := toolchain.ToolchainName{Channel: parsed.Channel, Version: latest}.String()
		if latestName != name {
			_, err = manifest.GetDownloadInfo(parsed.Channel, latest, platformKey)
			if err != nil {
				var vnaErr *cjverr.VersionNotAvailableError
				if errors.As(err, &vnaErr) {
					fmt.Printf("  %s: %s\n", name, i18n.T("UpdateAvailableButNotForPlatform", i18n.MsgData{
						"Current":  name,
						"Latest":   latestName,
						"Platform": platformKey,
					}))
				}
				continue
			}
			color.Yellow("  %s → %s", name, latestName)
			hasUpdates = true
		} else {
			color.Green("  %s ✓", name)
		}
	}

	fmt.Printf("\n  cjv %s\n", version)

	if !hasUpdates {
		color.Green(i18n.T("AllUpToDate", nil))
	}

	return nil
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
