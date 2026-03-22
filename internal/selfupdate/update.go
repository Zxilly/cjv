package selfupdate

import (
	"context"
	"fmt"
	"net/url"
	"runtime"
	"strings"

	"github.com/Zxilly/cjv/internal/i18n"
	go_selfupdate "github.com/creativeprojects/go-selfupdate"
)

// PlaceholderURL is the sentinel URL injected at build time for mirror variants
// that do not support auto-update. Must match the value in .goreleaser.yml.
const PlaceholderURL = "https://mirror.placeholder.invalid/cjv/releases"

// Update checks for and applies a self-update.
// updateURL: GitHub releases URL (e.g., "https://github.com/owner/cjv/releases")
// currentVersion: the current version string
func Update(ctx context.Context, updateURL, currentVersion string) error {
	if updateURL == "" || updateURL == PlaceholderURL {
		fmt.Println(i18n.T("MirrorNoAutoUpdate", nil))
		return nil
	}

	// Development builds have no meaningful version to compare against,
	// so skip the network round-trip entirely.
	if currentVersion == "dev" {
		fmt.Println(i18n.T("AlreadyUpToDate", i18n.MsgData{"Version": currentVersion}))
		return nil
	}

	source, err := go_selfupdate.NewGitHubSource(go_selfupdate.GitHubConfig{})
	if err != nil {
		return err
	}

	updater, err := go_selfupdate.NewUpdater(go_selfupdate.Config{
		Source:  source,
		Filters: []string{fmt.Sprintf("cjv_%s_%s", runtime.GOOS, runtime.GOARCH)},
	})
	if err != nil {
		return err
	}

	latest, found, err := updater.DetectLatest(ctx, go_selfupdate.ParseSlug(extractSlug(updateURL)))
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if !found || latest.LessOrEqual(currentVersion) {
		fmt.Println(i18n.T("AlreadyUpToDate", i18n.MsgData{"Version": currentVersion}))
		return nil
	}

	fmt.Println(i18n.T("UpdateFound", i18n.MsgData{
		"Current": currentVersion,
		"Latest":  latest.Version(),
	}))

	managedExe, err := ManagedExecutablePath()
	if err != nil {
		return err
	}

	if err := updater.UpdateTo(ctx, latest, managedExe); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Println(i18n.T("UpdateApplied", i18n.MsgData{"Version": latest.Version()}))
	return nil
}

// extractSlug extracts an "owner/repo" slug from a full GitHub URL.
// If rawURL is already in "owner/repo" format, it is returned as-is.
func extractSlug(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" {
		return rawURL
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	return rawURL
}
