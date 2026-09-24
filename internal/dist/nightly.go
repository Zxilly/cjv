package dist

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/retry"
)

// MaxResponseSize limits HTTP metadata reads.
const MaxResponseSize = 10 << 20 // 10 MB

// AppVersion is the running cjv's version as reported in the User-Agent of
// every request the shared client makes. main sets it from the
// linker-injected version before the first request; "dev" is the fallback.
var AppVersion = "dev"

var (
	httpClient     *http.Client
	httpClientOnce sync.Once
)

// HTTPClient returns the shared HTTP client with timeout and User-Agent.
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
			ua:   "cjv/" + AppVersion,
		},
	}
}

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

var errChecksumSidecarMalformed = errors.New("nightly checksum sidecar is malformed (expected 64 hex chars)")

// FetchNightlySHA256 fetches the optional sha256 sidecar for assetURL. A 404
// represents an upstream release that relies on TLS transport integrity.
func FetchNightlySHA256(ctx context.Context, assetURL string) (string, error) {
	var sha string
	err := retry.Do(getMaxDownloadRetries()+1,
		func(e error) bool { return !errors.Is(e, errChecksumSidecarMalformed) },
		func() error {
			var fetchErr error
			sha, fetchErr = fetchNightlySHA256Once(ctx, assetURL)
			return fetchErr
		})
	return sha, err
}

func fetchNightlySHA256Once(ctx context.Context, assetURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL+".sha256", nil)
	if err != nil {
		return "", err
	}
	resp, err := HTTPClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch nightly checksum: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch nightly checksum: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize))
	if err != nil {
		return "", err
	}
	sha := parseSHA256(string(body))
	if sha == "" {
		return "", errChecksumSidecarMalformed
	}
	return sha, nil
}
