//go:build !mirror

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

func runUpdate(ctx context.Context, updateURL, currentVersion string) error {
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

// extractSlug extracts an "owner/repo" slug from a full release URL.
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
