package dist

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/utils"
)

const DefaultNightlyBaseURL = "https://gitcode.com/Cangjie/nightly_build/releases/download"

// GitCodeTokenHeader is the HTTP header used for GitCode API authentication.
const GitCodeTokenHeader = "PRIVATE-TOKEN"

// DefaultNightlyAPIURL is the GitCode GET .../releases/latest endpoint for the nightly_build repo.
const DefaultNightlyAPIURL = "https://api.gitcode.com/api/v5/repos/Cangjie/nightly_build/releases/latest"

// MaxResponseSize limits HTTP response body reads to prevent memory exhaustion.
const MaxResponseSize = 10 << 20 // 10 MB

var (
	httpClient     *http.Client
	httpClientOnce sync.Once
)

// HTTPClient returns the shared HTTP client with proper timeout and User-Agent.
// The client is lazily initialized so that CJV_DOWNLOAD_TIMEOUT can be set
// via t.Setenv before the first call in tests.
func HTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		httpClient = newHTTPClient()
	})
	return httpClient
}

func newHTTPClient() *http.Client {
	timeout := 180 * time.Second
	if s := os.Getenv(config.EnvDownloadTimeout); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			timeout = time.Duration(n) * time.Second
		}
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &uaTransport{
			base: http.DefaultTransport,
			ua:   "cjv/" + utils.Version(),
		},
	}
}

// uaTransport adds a User-Agent header to all requests.
type uaTransport struct {
	base http.RoundTripper
	ua   string
}

func (t *uaTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		r2 := new(http.Request)
		*r2 = *req
		r2.Header = r2.Header.Clone()
		r2.Header.Set("User-Agent", t.ua)
		req = r2
	}
	return t.base.RoundTrip(req)
}

func NightlyDownloadURL(baseURL, version, goos, goarch string) (string, error) {
	filename, err := NightlyFilename(goos, goarch, version)
	if err != nil {
		return "", err
	}
	return nightlyDownloadURL(baseURL, version, filename)
}

func NightlyDownloadURLForPlatform(baseURL, version, platformKey string) (string, error) {
	filename, err := NightlyFilenameForPlatform(platformKey, version)
	if err != nil {
		return "", err
	}
	return nightlyDownloadURL(baseURL, version, filename)
}

func nightlyDownloadURL(baseURL, version, filename string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid nightly base URL: %w", err)
	}
	base = base.JoinPath(version, filename)
	return base.String(), nil
}

func parseSHA256(content string) string {
	digest := strings.TrimSpace(content)
	if len(digest) != 64 {
		return ""
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return ""
	}
	return strings.ToLower(digest)
}

func FetchNightlySHA256(ctx context.Context, assetURL string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL+".sha256", nil)
	if err != nil {
		return ""
	}
	resp, err := HTTPClient().Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort cleanup

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize))
	if err != nil {
		return ""
	}
	return parseSHA256(string(body))
}

// gitCodeRelease matches the JSON object returned by GitCode GET .../releases/latest
// (nightly_build responses include many assets; only tag_name is required for cjv).
type gitCodeRelease struct {
	TagName         string                `json:"tag_name"`
	TargetCommitish string                `json:"target_commitish"`
	Prerelease      bool                  `json:"prerelease"`
	Name            string                `json:"name"`
	Body            string                `json:"body"`
	Author          gitCodeReleaseAuthor  `json:"author"`
	CreatedAt       string                `json:"created_at"`
	Assets          []gitCodeReleaseAsset `json:"assets"`
}

type gitCodeReleaseAuthor struct {
	ID        string `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Type      string `json:"type"`
	URL       string `json:"url"`
}

type gitCodeReleaseAsset struct {
	BrowserDownloadURL string `json:"browser_download_url"`
	Name               string `json:"name"`
	Type               string `json:"type"` // e.g. "source", "attach"
}

// FetchLatestNightly queries the GitCode releases/latest API and returns the
// tag_name of the repository's latest release.
// apiURL should be the full latest endpoint URL (e.g. DefaultNightlyAPIURL).
// apiKey is the GitCode API access token; required for authentication.
func FetchLatestNightly(ctx context.Context, apiURL, apiKey string) (string, error) {
	if apiKey == "" {
		return "", &cjverr.GitCodeAPIKeyRequiredError{}
	}
	u, err := url.Parse(apiURL)
	if err != nil {
		return "", fmt.Errorf("invalid nightly API URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create nightly request: %w", err)
	}
	req.Header.Set(GitCodeTokenHeader, apiKey)
	resp, err := HTTPClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to query nightly versions: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort cleanup

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to query nightly versions: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize))
	if err != nil {
		return "", err
	}

	var release gitCodeRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("failed to parse nightly release: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("nightly release has empty tag_name")
	}

	return release.TagName, nil
}
