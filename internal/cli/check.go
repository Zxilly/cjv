package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/output"
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

type checkEntry struct {
	Name            string `json:"name"`
	Latest          string `json:"latest,omitempty"`
	UpdateAvailable bool   `json:"update_available"`
	NotForPlatform  bool   `json:"not_for_platform,omitempty"`
	Platform        string `json:"platform,omitempty"`
	Error           string `json:"error,omitempty"`
}

type checkResult struct {
	Toolchains    []checkEntry `json:"toolchains"`
	CjvVersion    string       `json:"cjv_version"`
	HasUpdates    bool         `json:"has_updates"`
	NoneInstalled bool         `json:"none_installed,omitempty"`
}

func (r checkResult) Text() string {
	if r.NoneInstalled {
		return i18n.T("NoToolchainsInstalled", nil)
	}
	var b strings.Builder
	for _, e := range r.Toolchains {
		switch {
		case e.Error != "":
			fmt.Fprintf(&b, "  %s: %s\n", e.Name, e.Error)
		case e.NotForPlatform:
			fmt.Fprintf(&b, "  %s: %s\n", e.Name, i18n.T("UpdateAvailableButNotForPlatform", i18n.MsgData{
				"Current":  e.Name,
				"Latest":   e.Latest,
				"Platform": e.Platform,
			}))
		case e.UpdateAvailable:
			b.WriteString(color.YellowString("  %s → %s", e.Name, e.Latest))
			b.WriteByte('\n')
		default:
			b.WriteString(color.GreenString("  %s ✓", e.Name))
			b.WriteByte('\n')
		}
	}
	fmt.Fprintf(&b, "\n  cjv %s\n", r.CjvVersion)
	if !r.HasUpdates {
		b.WriteString(color.GreenString(i18n.T("AllUpToDate", nil)))
		b.WriteByte('\n')
	}
	return b.String()
}

func runCheck(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	installed, err := toolchain.ListInstalled()
	if err != nil {
		return err
	}
	if len(installed) == 0 {
		return output.RenderTo(cmdOutput(cmd), checkResult{NoneInstalled: true, CjvVersion: version})
	}

	_, settings, err := clisettings.LoadSettings()
	if err != nil {
		return err
	}

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

	var manifest *dist.Manifest
	var manifestErr error

	result := checkResult{CjvVersion: version}

	for _, name := range installed {
		parsed, err := toolchain.ParseToolchainName(name)
		if err != nil || parsed.IsCustom() || parsed.Channel == toolchain.UnknownChannel {
			continue
		}

		if parsed.Channel == toolchain.Nightly {
			if nightlyErr != nil {
				result.Toolchains = append(result.Toolchains, checkEntry{Name: name, Error: nightlyErr.Error()})
				continue
			}
			latestName := toolchain.ToolchainName{
				Channel:     toolchain.Nightly,
				Version:     latestNightly,
				PlatformKey: parsed.PlatformKey,
			}.String()
			entry := checkEntry{Name: name, Latest: latestName}
			if latestName != name {
				entry.UpdateAvailable = true
				result.HasUpdates = true
			}
			result.Toolchains = append(result.Toolchains, entry)
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
		infoPlatformKey := platformKey
		if parsed.PlatformKey != "" {
			latestName = toolchain.ToolchainName{Channel: parsed.Channel, Version: latest, PlatformKey: parsed.PlatformKey}.String()
			infoPlatformKey = parsed.PlatformKey
		}
		entry := checkEntry{Name: name, Latest: latestName}
		if latestName != name {
			_, err = manifest.GetDownloadInfo(parsed.Channel, latest, infoPlatformKey)
			if err != nil {
				if _, ok := errors.AsType[*cjverr.VersionNotAvailableError](err); ok {
					entry.NotForPlatform = true
					entry.Platform = infoPlatformKey
				}
				result.Toolchains = append(result.Toolchains, entry)
				continue
			}
			entry.UpdateAvailable = true
			result.HasUpdates = true
		}
		result.Toolchains = append(result.Toolchains, entry)
	}

	return output.RenderTo(cmdOutput(cmd), result)
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
