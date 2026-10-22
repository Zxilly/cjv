//go:build mirror

package selfupdate

import (
	"context"
	"fmt"
	"net/url"
	"runtime"
	"strings"

	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/progress"
)

const mirrorBinaryName = "cjv-mirror"

func latestTag(ctx context.Context, updateURL string) (string, error) {
	base, err := gitCodeReleasesBase(updateURL)
	if err != nil {
		return "", err
	}
	return fetchGitCodeLatestTag(ctx, base+"/latest")
}

func runUpdate(ctx context.Context, updateURL, currentVersion string, sink progress.Sink) (Result, error) {
	base, err := gitCodeReleasesBase(updateURL)
	if err != nil {
		return Result{}, err
	}
	tag, err := fetchGitCodeLatestTag(ctx, base+"/latest")
	if err != nil {
		return Result{}, fmt.Errorf("failed to check for updates: %w", err)
	}
	latest, newer, err := newerReleaseVersion(currentVersion, tag)
	if err != nil {
		return Result{}, err
	}
	if !newer {
		return Result{CurrentVersion: currentVersion, Version: currentVersion, Status: StatusUpToDate}, nil
	}

	assetName := mirrorAssetName(runtime.GOOS, runtime.GOARCH)
	if err := installReleaseArtifact(ctx, releaseArtifact{
		AssetName:   assetName,
		BinaryName:  platformBinaryName(mirrorBinaryName, runtime.GOOS),
		AssetURL:    base + "/download/" + tag + "/" + assetName,
		ChecksumURL: base + "/download/" + tag + "/checksums.txt",
		Progress:    sink,
	}); err != nil {
		return Result{}, fmt.Errorf("update failed: %w", err)
	}
	return Result{CurrentVersion: currentVersion, Version: latest, Status: StatusUpdated}, nil
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
	loc, err := dist.FetchRedirectLocation(ctx, latestURL)
	if err != nil {
		return "", err
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
