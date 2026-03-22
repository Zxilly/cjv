package dist

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Zxilly/cjv/internal/utils"
)

const DefaultNightlyBaseURL = "https://gitcode.com/Cangjie/nightly_build/releases/download"

// DefaultNightlyAPIURL is the releases API endpoint for querying nightly versions.
const DefaultNightlyAPIURL = "https://gitcode.com/api/v1/repos/Cangjie/nightly_build/releases"

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
	if s := os.Getenv("CJV_DOWNLOAD_TIMEOUT"); s != "" {
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
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid nightly base URL: %w", err)
	}
	base = base.JoinPath(version, filename)
	return base.String(), nil
}

func ProbeNightlyVersion(ctx context.Context, baseURL, version, goos, goarch string) (bool, error) {
	url, err := NightlyDownloadURL(baseURL, version, goos, goarch)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false, err
	}
	resp, err := HTTPClient().Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort cleanup

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("failed to probe nightly version: HTTP %d", resp.StatusCode)
	}
}

type gitCodeRelease struct {
	TagName string `json:"tag_name"`
}

// extractNightlyTimestamp returns the numeric timestamp from a nightly tag
// name like "1.1.0-alpha.20260306010001". It extracts the substring after
// the last "." and parses it as an integer. Returns 0 if the tag does not
// contain a valid numeric timestamp suffix.
func extractNightlyTimestamp(tag string) int64 {
	idx := strings.LastIndex(tag, ".")
	if idx < 0 || idx == len(tag)-1 {
		return 0
	}
	ts, err := strconv.ParseInt(tag[idx+1:], 10, 64)
	if err != nil {
		return 0
	}
	return ts
}

// FetchLatestNightly queries the GitCode releases API and returns the
// latest nightly version tag, sorted descending by the embedded timestamp.
// apiURL should be the releases API base (e.g. DefaultNightlyAPIURL).
func FetchLatestNightly(ctx context.Context, apiURL string) (string, error) {
	u, err := url.Parse(apiURL)
	if err != nil {
		return "", fmt.Errorf("invalid nightly API URL: %w", err)
	}
	q := u.Query()
	q.Set("limit", "50")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create nightly request: %w", err)
	}
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

	var releases []gitCodeRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return "", fmt.Errorf("failed to parse nightly version list: %w", err)
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("no nightly versions found")
	}

	// Pre-compute timestamps so MaxFunc doesn't re-parse on every comparison.
	type tagged struct {
		tag string
		ts  int64
	}
	items := make([]tagged, len(releases))
	for i, r := range releases {
		items[i] = tagged{r.TagName, extractNightlyTimestamp(r.TagName)}
	}
	best := slices.MaxFunc(items, func(a, b tagged) int {
		switch {
		case a.ts != 0 && b.ts != 0:
			return cmp.Compare(a.ts, b.ts)
		case a.ts != 0:
			return 1
		case b.ts != 0:
			return -1
		default:
			return cmp.Compare(a.tag, b.tag)
		}
	})

	return best.tag, nil
}
