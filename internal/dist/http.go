package dist

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
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
	if Offline(req.Context()) {
		return nil, &nonRetriableError{err: &cjverr.NetworkDisabledError{URL: req.URL.Redacted()}}
	}
	if req.Header.Get("User-Agent") == "" {
		r2 := new(http.Request)
		*r2 = *req
		r2.Header = r2.Header.Clone()
		r2.Header.Set("User-Agent", t.ua)
		req = r2
	}
	return t.base.RoundTrip(req)
}

func redactURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "<invalid URL>"
	}
	return parsed.Redacted()
}

func validateHTTPURL(u *url.URL, label string) error {
	if u == nil || u.Host == "" {
		return fmt.Errorf("invalid %s URL", label)
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		if isLoopbackHost(u.Hostname()) || os.Getenv(config.EnvAllowInsecureManifest) == "1" {
			return nil
		}
		return fmt.Errorf("refusing to use insecure HTTP %s %q: use HTTPS, or set %s=1 to trust an internal mirror", label, u.Host, config.EnvAllowInsecureManifest)
	default:
		return fmt.Errorf("invalid %s URL scheme %q: only https and http are supported", label, u.Scheme)
	}
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}
