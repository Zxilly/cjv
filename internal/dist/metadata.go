package dist

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
)

type offlineKey struct{}

// WithOffline forbids network access for this operation. Cached artifacts may
// still be used. CJV_OFFLINE=1 applies the same policy to proxy invocations.
func WithOffline(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, offlineKey{}, enabled)
}

func Offline(ctx context.Context) bool {
	enabled, _ := ctx.Value(offlineKey{}).(bool)
	return enabled || os.Getenv("CJV_OFFLINE") == "1"
}

// FetchMetadata applies the transfer retry policy to bounded metadata reads.
// Parsing is deliberately left to callers: malformed documents are not retried.
func FetchMetadata(ctx context.Context, rawURL string) ([]byte, error) {
	return fetchValidatedMetadata(ctx, rawURL, nil)
}

// Validate before publication so a malformed response cannot replace a usable
// offline copy. Callers also parse cached bytes before trusting their contents.
func fetchValidatedMetadata(ctx context.Context, rawURL string, validate func([]byte) error) ([]byte, error) {
	fetch := func() ([]byte, error) {
		data, err := fetchMetadata(ctx, rawURL)
		if err == nil && validate != nil {
			err = validate(data)
		}
		return data, err
	}
	if enabled, _ := ctx.Value(metadataCacheKey{}).(bool); enabled {
		return cachedMetadata(ctx, rawURL, fetch)
	}
	return fetch()
}

func fetchMetadata(ctx context.Context, rawURL string) ([]byte, error) {
	var data []byte
	err := retryTransfer(ctx, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return &nonRetriableError{err: err}
		}
		resp, err := HTTPClient().Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close() //nolint:errcheck
		if resp.StatusCode != http.StatusOK {
			fetchErr := &ManifestFetchError{URL: rawURL, StatusCode: resp.StatusCode}
			if isNonRetriableHTTPStatus(resp.StatusCode) {
				return &nonRetriableError{err: fetchErr}
			}
			return &retryAfterError{err: fetchErr, delay: retryAfter(resp.Header.Get("Retry-After"))}
		}
		data, err = io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize+1))
		if err == nil && len(data) > MaxResponseSize {
			return &nonRetriableError{err: fmt.Errorf("metadata exceeds %d bytes", MaxResponseSize)}
		}
		return err
	})
	return data, err
}

// FetchRedirectLocation reads a release endpoint without following its redirect,
// using the same offline and transient-failure policy as other metadata.
func FetchRedirectLocation(ctx context.Context, rawURL string) (string, error) {
	client := *HTTPClient()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	var location string
	err := retryTransfer(ctx, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
		if err != nil {
			return &nonRetriableError{err: err}
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close() //nolint:errcheck
		if resp.StatusCode >= 400 {
			return transferResponseError(resp, rawURL)
		}
		location = resp.Header.Get("Location")
		if resp.StatusCode < 300 || resp.StatusCode >= 400 || location == "" {
			return &nonRetriableError{err: fmt.Errorf("no redirect location from %s (HTTP %d)", rawURL, resp.StatusCode)}
		}
		return nil
	})
	return location, err
}
