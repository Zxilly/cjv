package dist

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/utils"
	"github.com/mattn/go-isatty"
	"github.com/schollz/progressbar/v3"
)

// nonRetriableError wraps an error that should not be retried (e.g. permanent HTTP 4xx).
type nonRetriableError struct {
	err error
}

func (e *nonRetriableError) Error() string { return e.err.Error() }
func (e *nonRetriableError) Unwrap() error { return e.err }

// getMaxDownloadRetries returns the number of download retry attempts.
// Reads CJV_MAX_RETRIES at call time so tests can override via t.Setenv.
func getMaxDownloadRetries() int {
	if s := os.Getenv(config.EnvMaxRetries); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			return n
		}
	}
	return 3
}

// cacheKey returns the cache file name for a download.
// If sha256Hex is provided, it is used directly; otherwise sha256(url) is used.
func cacheKey(url, sha256Hex string) string {
	if sha256Hex != "" {
		return sha256Hex
	}
	h := sha256.Sum256([]byte(url))
	return hex.EncodeToString(h[:])
}

// DownloadFileCached downloads url to dest using cacheDir for content-addressed caching.
// If the cache already contains a file matching the expected hash, it is copied
// directly without making an HTTP request. On cache miss the file is downloaded
// into cacheDir first, then copied to dest.
func DownloadFileCached(ctx context.Context, url, dest, sha256Hex, cacheDir string) error {
	sha256Hex = strings.ToLower(sha256Hex)
	key := cacheKey(url, sha256Hex)
	cachedPath := filepath.Join(cacheDir, key)
	partialPath := cachedPath + ".partial"

	// Cache hit: file already exists in cache.
	if _, err := os.Stat(cachedPath); err == nil {
		slog.Info("cache hit", "key", key)
		if err := verifyCachedFile(cachedPath, sha256Hex); err == nil {
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return err
			}
			return utils.CopyFile(cachedPath, dest, 0o644)
		} else {
			var mismatchErr *cjverr.ChecksumMismatchError
			if errors.As(err, &mismatchErr) {
				slog.Warn("cached download checksum mismatch; redownloading", "path", cachedPath, "expected", mismatchErr.Expected, "actual", mismatchErr.Actual)
				if err := utils.RemoveAllRetry(cachedPath); err != nil {
					return fmt.Errorf("remove corrupt cache %s: %w", cachedPath, err)
				}
			} else {
				return err
			}
		}
	}

	// Cache miss: download into cacheDir.
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}

	var lastErr error
	for attempt := range getMaxDownloadRetries() + 1 {
		if attempt > 0 {
			slog.Info("retrying download", "attempt", attempt+1, "max", getMaxDownloadRetries()+1)
		}

		lastErr = downloadOnce(ctx, url, partialPath, filepath.Base(dest), sha256Hex)
		if lastErr == nil {
			// Promote partial to cached.
			if err := utils.RenameRetry(partialPath, cachedPath); err != nil {
				return fmt.Errorf("promote to cache %s: %w", cachedPath, err)
			}
			// Copy from cache to dest.
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return err
			}
			return utils.CopyFile(cachedPath, dest, 0o644)
		}

		var nre *nonRetriableError
		if errors.As(lastErr, &nre) {
			cleanupDownloadTemp(partialPath)
			break
		}
		// On retriable errors, keep .partial for resume on next attempt.
	}
	return lastErr
}

// verifyChecksum compares the hash digest against the expected hex string.
// Returns a ChecksumMismatchError on mismatch, nil on match or if expected is empty.
func verifyChecksum(hasher hash.Hash, expected string) error {
	if expected == "" {
		return nil
	}
	actual := hex.EncodeToString(hasher.Sum(nil))
	if actual != expected {
		return &cjverr.ChecksumMismatchError{Expected: expected, Actual: actual}
	}
	return nil
}

func verifyCachedFile(path, sha256Hex string) error {
	if sha256Hex == "" {
		// No checksum available (e.g. nightly builds). Verify the cached
		// file is non-empty and has a valid archive magic header to catch
		// corrupt or truncated downloads.
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.Size() == 0 {
			return fmt.Errorf("cached file %s is empty", path)
		}
		if err := verifyArchiveMagic(path); err != nil {
			return fmt.Errorf("cached file %s is not a valid archive: %w", path, err)
		}
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck // read-only

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return fmt.Errorf("hash cached file %s: %w", path, err)
	}
	return verifyChecksum(hasher, sha256Hex)
}

// DownloadFile downloads url to dest, optionally verifying the SHA256 checksum.
// An empty sha256Hex skips verification (used for nightly builds).
// Retries on transient failures.
func DownloadFile(ctx context.Context, url, dest, sha256Hex string) error {
	var lastErr error
	for attempt := range getMaxDownloadRetries() + 1 {
		tmpPath, err := newDownloadTempPath(dest)
		if err != nil {
			return err
		}

		if attempt > 0 {
			slog.Info("retrying download", "attempt", attempt+1, "max", getMaxDownloadRetries()+1)
		}

		lastErr = downloadOnce(ctx, url, tmpPath, filepath.Base(dest), sha256Hex)
		if lastErr == nil {
			lastErr = promoteDownloadedFile(tmpPath, dest)
		}
		if lastErr == nil {
			return nil
		}

		cleanupDownloadTemp(tmpPath)

		var nre *nonRetriableError
		if errors.As(lastErr, &nre) {
			break
		}
	}
	return lastErr
}

func newDownloadTempPath(dest string) (string, error) {
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	f, err := os.CreateTemp(dir, "."+filepath.Base(dest)+".partial-*")
	if err != nil {
		return "", err
	}
	path := f.Name()
	if err := f.Close(); err != nil {
		return "", errors.Join(err, os.Remove(path))
	}
	return path, nil
}

func cleanupDownloadTemp(path string) {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Warn("failed to clean up temporary file", "path", path, "error", err)
	}
}

func promoteDownloadedFile(tmpPath, dest string) error {
	if err := utils.RenameRetry(tmpPath, dest); err != nil {
		return &nonRetriableError{err: fmt.Errorf("failed to promote download cache %s: %w", dest, err)}
	}
	return nil
}

func downloadOnce(ctx context.Context, url, tmpPath, displayName, sha256Hex string) error {
	client := HTTPClient()

	// Check for existing partial file for resume.
	var existingSize int64
	if info, err := os.Stat(tmpPath); err == nil {
		existingSize = info.Size()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if existingSize > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", existingSize))
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort cleanup

	// Determine whether to resume or start fresh.
	var f *os.File
	resumed := false

	switch resp.StatusCode {
	case http.StatusPartialContent:
		resumed = true
	case http.StatusOK:
		existingSize = 0
	case http.StatusRequestedRangeNotSatisfiable:
		// Range not satisfiable — discard partial and start over.
		cleanupDownloadTemp(tmpPath)
		// Verify the partial file was actually removed. If it still exists
		// (e.g. file locked on Windows), we must not continue — the stale
		// content could produce a corrupt file that nightly builds (which
		// lack SHA256 checksums) would silently accept.
		if _, statErr := os.Stat(tmpPath); statErr == nil {
			return fmt.Errorf("cannot resume download: failed to remove stale partial file %s", tmpPath)
		}
		existingSize = 0
	default:
		if isNonRetriableHTTPStatus(resp.StatusCode) {
			return &nonRetriableError{err: fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)}
		}
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	// For SHA256 verification we need to hash the *complete* file, including
	// the already-downloaded portion when resuming. This is only done when
	// sha256Hex is non-empty; nightly builds skip checksum verification so
	// the hasher starts empty and verifyChecksum returns nil for them.
	hasher := sha256.New()

	if resumed && sha256Hex != "" {
		// Hash the existing partial content BEFORE opening the file for
		// append, to avoid a sharing violation on Windows where opening
		// the same file with incompatible modes (O_WRONLY + O_RDONLY) fails.
		existing, err := os.Open(tmpPath)
		if err != nil {
			return fmt.Errorf("open partial for hashing: %w", err)
		}
		if _, err := io.Copy(hasher, existing); err != nil {
			existing.Close() //nolint:errcheck
			return fmt.Errorf("hash existing partial: %w", err)
		}
		existing.Close() //nolint:errcheck
	}

	// Now open the file for writing.
	if resumed {
		f, err = os.OpenFile(tmpPath, os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("open partial file for append: %w", err)
		}
	} else {
		f, err = os.Create(tmpPath)
		if err != nil {
			return fmt.Errorf("create temp file: %w", err)
		}
	}
	defer f.Close() //nolint:errcheck // best-effort

	// Compute total size for progress bar.
	var totalSize int64 = -1
	if resp.ContentLength > 0 {
		totalSize = resp.ContentLength + existingSize
	}

	reader := io.TeeReader(resp.Body, hasher)

	// Suppress the animated progress bar in non-interactive contexts (CI, piped output).
	interactive := isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())

	var dst io.Writer = f
	if interactive {
		bar := progressbar.NewOptions64(totalSize,
			progressbar.OptionSetDescription(displayName),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionOnCompletion(func() { fmt.Fprint(os.Stderr, "\n") }),
		)
		if resumed && existingSize > 0 {
			bar.Set64(existingSize) //nolint:errcheck // progress bar cosmetic
		}
		dst = io.MultiWriter(f, bar)
	}

	if _, err := io.Copy(dst, reader); err != nil {
		return fmt.Errorf("download write: %w", err)
	}

	if err := verifyChecksum(hasher, sha256Hex); err != nil {
		return &nonRetriableError{err: err}
	}

	return nil
}

// verifyArchiveMagic checks that path starts with a recognized archive header
// (zip PK\x03\x04 or gzip \x1f\x8b). This catches truncated or corrupt
// downloads for nightly builds that lack SHA256 checksums.
func verifyArchiveMagic(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck // read-only

	var magic [4]byte
	n, err := f.Read(magic[:])
	if err != nil {
		return fmt.Errorf("read header: %w", err)
	}
	if n < 2 {
		return fmt.Errorf("file too small to be a valid archive")
	}

	// zip: PK\x03\x04
	if magic[0] == 0x50 && magic[1] == 0x4B && n >= 4 && magic[2] == 0x03 && magic[3] == 0x04 {
		return nil
	}
	// gzip: \x1f\x8b
	if magic[0] == 0x1f && magic[1] == 0x8b {
		return nil
	}

	return fmt.Errorf("unrecognized archive header: %#x %#x", magic[0], magic[1])
}

func isNonRetriableHTTPStatus(statusCode int) bool {
	if statusCode < http.StatusBadRequest || statusCode >= http.StatusInternalServerError {
		return false
	}
	switch statusCode {
	case http.StatusRequestTimeout, http.StatusTooManyRequests:
		return false
	default:
		return true
	}
}
