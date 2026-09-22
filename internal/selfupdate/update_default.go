//go:build !mirror

package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"runtime"
	"strings"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func runUpdate(ctx context.Context, updateURL, currentVersion string) (Result, error) {
	slug := extractSlug(updateURL)
	parts := strings.Split(slug, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Result{}, fmt.Errorf("invalid GitHub repository %q", slug)
	}
	data, err := fetchReleaseFile(ctx, fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", slug))
	if err != nil {
		return Result{}, fmt.Errorf("failed to check for updates: %w", err)
	}
	var release githubRelease
	if err := json.Unmarshal(data, &release); err != nil {
		return Result{}, fmt.Errorf("failed to parse latest release: %w", err)
	}
	latest, newer, err := newerReleaseVersion(currentVersion, release.TagName)
	if err != nil {
		return Result{}, err
	}
	if !newer {
		return Result{CurrentVersion: currentVersion, Version: currentVersion, Status: StatusUpToDate}, nil
	}

	assetName := releaseAssetName("cjv", runtime.GOOS, runtime.GOARCH)
	assetURL, checksumURL := "", ""
	for _, asset := range release.Assets {
		switch asset.Name {
		case assetName:
			assetURL = asset.URL
		case "checksums.txt":
			checksumURL = asset.URL
		}
	}
	if assetURL == "" || checksumURL == "" {
		return Result{}, fmt.Errorf("release %s is missing %s or checksums.txt", release.TagName, assetName)
	}
	if err := installReleaseArtifact(ctx, releaseArtifact{
		AssetName:   assetName,
		BinaryName:  platformBinaryName("cjv", runtime.GOOS),
		AssetURL:    assetURL,
		ChecksumURL: checksumURL,
	}); err != nil {
		return Result{}, fmt.Errorf("update failed: %w", err)
	}
	return Result{CurrentVersion: currentVersion, Version: latest, Status: StatusUpdated}, nil
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
