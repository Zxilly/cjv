package dist

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

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

// Only a confirmed HTTP 404 creates this cache marker. An empty or malformed
// 200 response must still fail validation, including one containing this marker.
const nightlyChecksumMissingCache = `{"status_code":404}`

// FetchNightlySHA256 fetches the optional sha256 sidecar for assetURL. A 404
// represents an upstream release that relies on TLS transport integrity. Cache
// that absence as well so offline metadata operations retain the same behavior.
func FetchNightlySHA256(ctx context.Context, assetURL string) (string, error) {
	sidecarURL := assetURL + ".sha256"
	body, err := cachedMetadata(ctx, sidecarURL, func() ([]byte, error) {
		data, err := fetchMetadata(ctx, sidecarURL)
		if err != nil {
			var fetchErr *ManifestFetchError
			if errors.As(err, &fetchErr) && fetchErr.StatusCode == http.StatusNotFound {
				return []byte(nightlyChecksumMissingCache), nil
			}
			return nil, err
		}
		if parseSHA256(string(data)) == "" {
			return nil, errChecksumSidecarMalformed
		}
		return data, nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to fetch nightly checksum: %w", err)
	}
	if string(body) == nightlyChecksumMissingCache {
		return "", nil
	}
	sha := parseSHA256(string(body))
	if sha == "" {
		return "", errChecksumSidecarMalformed
	}
	return sha, nil
}
