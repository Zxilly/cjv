package dist

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// 10 MB

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
	err := retryTransfer(ctx, func() error {
		var fetchErr error
		sha, fetchErr = fetchNightlySHA256Once(ctx, assetURL)
		return fetchErr
	})
	return sha, err
}

func fetchNightlySHA256Once(ctx context.Context, assetURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL+".sha256", nil)
	if err != nil {
		return "", &nonRetriableError{err: err}
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
		return "", fmt.Errorf("failed to fetch nightly checksum: %w", transferHTTPError(resp.StatusCode, assetURL+".sha256"))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize))
	if err != nil {
		return "", err
	}
	sha := parseSHA256(string(body))
	if sha == "" {
		return "", &nonRetriableError{err: errChecksumSidecarMalformed}
	}
	return sha, nil
}
