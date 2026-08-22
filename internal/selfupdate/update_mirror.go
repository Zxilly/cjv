//go:build mirror

package selfupdate

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/Zxilly/cjv/internal/i18n"
)

const mirrorBinaryName = "cjv-mirror"

func runUpdate(ctx context.Context, updateURL, currentVersion string) error {
	base, err := gitCodeReleasesBase(updateURL)
	if err != nil {
		return err
	}
	tag, err := fetchGitCodeLatestTag(ctx, base+"/latest")
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	latest, newer, err := newerReleaseVersion(currentVersion, tag)
	if err != nil {
		return err
	}
	if !newer {
		fmt.Println(i18n.T("AlreadyUpToDate", i18n.MsgData{"Version": currentVersion}))
		return nil
	}

	fmt.Println(i18n.T("UpdateFound", i18n.MsgData{
		"Current": currentVersion,
		"Latest":  latest,
	}))

	assetName := mirrorAssetName(runtime.GOOS, runtime.GOARCH)
	if err := installReleaseArtifact(ctx, releaseArtifact{
		AssetName:   assetName,
		BinaryName:  platformBinaryName(mirrorBinaryName, runtime.GOOS),
		AssetURL:    base + "/download/" + tag + "/" + assetName,
		ChecksumURL: base + "/download/" + tag + "/checksums.txt",
	}); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Println(i18n.T("UpdateApplied", i18n.MsgData{"Version": latest}))
	return nil
}

func gitCodeReleasesBase(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("invalid gitcode URL %q", rawURL)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid gitcode slug in %q", rawURL)
	}
	return fmt.Sprintf("%s://%s/%s/%s/releases", u.Scheme, u.Host, parts[0], parts[1]), nil
}

func fetchGitCodeLatestTag(ctx context.Context, latestURL string) (string, error) {
	// http.ErrUseLastResponse stops redirect following without producing an
	// error, leaving the 3xx response with its Location header for inspection.
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, latestURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort cleanup

	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("no redirect location from %s (HTTP %d)", latestURL, resp.StatusCode)
	}

	if locURL, perr := url.Parse(loc); perr == nil && !locURL.IsAbs() {
		if base, berr := url.Parse(latestURL); berr == nil {
			loc = base.ResolveReference(locURL).String()
		}
	}

	tag := tagFromReleaseURL(loc)
	if tag == "" {
		return "", fmt.Errorf("could not parse tag from redirect %q", loc)
	}
	return tag, nil
}

func tagFromReleaseURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "tag" && parts[i+1] != "" {
			return parts[i+1]
		}
	}
	return ""
}

func mirrorAssetName(goos, goarch string) string {
	return releaseAssetName(mirrorBinaryName, goos, goarch)
}
