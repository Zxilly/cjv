//go:build mirror

package selfupdate

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/i18n"
	go_selfupdate "github.com/creativeprojects/go-selfupdate"
)

const mirrorBinaryName = "cjv-mirror"

func runUpdate(ctx context.Context, updateURL, currentVersion string) error {
	base, err := gitCodeReleasesBase(updateURL)
	if err != nil {
		return err
	}

	updater, err := go_selfupdate.NewUpdater(go_selfupdate.Config{
		Source: &gitCodeSource{base: base},
		// Verify the downloaded binary against the release checksums.txt (also
		// published on the GitCode mirror release) before replacing the running
		// executable, so a tampered mirror asset cannot be installed silently.
		Validator: &go_selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
		Filters:   []string{fmt.Sprintf("%s_%s_%s", mirrorBinaryName, runtime.GOOS, runtime.GOARCH)},
	})
	if err != nil {
		return err
	}

	// The repository argument is ignored by gitCodeSource (the base URL is
	// captured directly), but the framework requires a non-zero slug.
	latest, found, err := updater.DetectLatest(ctx, go_selfupdate.NewRepositorySlug("cjv", "mirror"))
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

// gitCodeSource adapts GitCode releases to the go-selfupdate Source interface.
// GitCode has no public release listing API, so the latest tag is detected by
// following the redirect of `<base>/releases/latest`. There is no real "list
// of releases" — only the latest one is exposed to the framework.
type gitCodeSource struct {
	base string
}

func (s *gitCodeSource) ListReleases(ctx context.Context, _ go_selfupdate.Repository) ([]go_selfupdate.SourceRelease, error) {
	tag, err := fetchGitCodeLatestTag(ctx, s.base+"/latest")
	if err != nil {
		return nil, err
	}
	return []go_selfupdate.SourceRelease{newGitCodeSourceRelease(s.base, tag)}, nil
}

func (s *gitCodeSource) DownloadReleaseAsset(ctx context.Context, rel *go_selfupdate.Release, assetID int64) (io.ReadCloser, error) {
	if rel == nil {
		return nil, fmt.Errorf("no release")
	}
	// The framework requests the main archive by rel.AssetID and the validation
	// asset (checksums.txt) by rel.ValidationAssetID. Both arrive here as
	// assetID, so pick the matching URL — otherwise the ChecksumValidator would
	// receive the archive bytes instead of checksums.txt and always fail.
	assetURL := rel.AssetURL
	if assetID == rel.ValidationAssetID && rel.ValidationAssetURL != "" {
		assetURL = rel.ValidationAssetURL
	}
	if assetURL == "" {
		return nil, fmt.Errorf("no asset URL for asset id %d", assetID)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := dist.HTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("download %s: HTTP %d", rel.AssetURL, resp.StatusCode)
	}
	return resp.Body, nil
}

// newGitCodeSourceRelease builds a SourceRelease that exposes the mirror
// archive for the running OS/arch as its only asset. Other platforms aren't
// listed because the running binary can only update to its own platform.
func newGitCodeSourceRelease(base, tag string) go_selfupdate.SourceRelease {
	name := mirrorAssetName(runtime.GOOS, runtime.GOARCH)
	return &gitCodeRelease{
		tag: tag,
		url: base + "/tag/" + tag,
		// Expose checksums.txt alongside the archive so the ChecksumValidator
		// can fetch it. Distinct non-zero IDs let DownloadReleaseAsset tell the
		// archive request apart from the validation-asset request.
		assets: []go_selfupdate.SourceAsset{
			&gitCodeAsset{id: 1, name: name, url: base + "/download/" + tag + "/" + name},
			&gitCodeAsset{id: 2, name: "checksums.txt", url: base + "/download/" + tag + "/checksums.txt"},
		},
	}
}

type gitCodeRelease struct {
	tag, url string
	assets   []go_selfupdate.SourceAsset
}

func (r *gitCodeRelease) GetID() int64              { return 0 }
func (r *gitCodeRelease) GetTagName() string        { return r.tag }
func (r *gitCodeRelease) GetDraft() bool            { return false }
func (r *gitCodeRelease) GetPrerelease() bool       { return false }
func (r *gitCodeRelease) GetPublishedAt() time.Time { return time.Time{} }
func (r *gitCodeRelease) GetReleaseNotes() string   { return "" }
func (r *gitCodeRelease) GetName() string           { return r.tag }
func (r *gitCodeRelease) GetURL() string            { return r.url }
func (r *gitCodeRelease) GetAssets() []go_selfupdate.SourceAsset {
	return r.assets
}

type gitCodeAsset struct {
	id   int64
	name string
	url  string
}

func (a *gitCodeAsset) GetID() int64                  { return a.id }
func (a *gitCodeAsset) GetName() string               { return a.name }
func (a *gitCodeAsset) GetSize() int                  { return 0 }
func (a *gitCodeAsset) GetBrowserDownloadURL() string { return a.url }

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
	if goos == "windows" {
		return fmt.Sprintf("%s_%s_%s.zip", mirrorBinaryName, goos, goarch)
	}
	return fmt.Sprintf("%s_%s_%s.tar.gz", mirrorBinaryName, goos, goarch)
}
