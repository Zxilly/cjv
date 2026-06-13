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
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/utils"
	"github.com/mattn/go-isatty"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

// nonRetriableError wraps an error that should not be retried (e.g. permanent HTTP 4xx).
type nonRetriableError struct {
	err error
}

func (e *nonRetriableError) Error() string { return e.err.Error() }
func (e *nonRetriableError) Unwrap() error { return e.err }

const (
	maxProgressNameWidth = 48
	downloadBarWidth     = 24
)

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

// DownloadCached stages url under cacheDir as a hash-keyed file and returns
// its path. cacheDir is a *transient* staging area, not a persistent cache:
// callers are expected to consume the returned file (typically by extracting
// an archive) and then drop it via os.Remove on the success path. Files left
// behind from a crashed earlier run are reused if their content still
// verifies, so repeated install attempts after an interruption do not
// re-download.
func DownloadCached(ctx context.Context, url, sha256Hex, cacheDir string) (string, error) {
	return DownloadCachedWithName(ctx, url, sha256Hex, cacheDir, "")
}

// DownloadCachedWithName is like DownloadCached, but displayName controls the
// interactive progress label. The staged filename remains hash-keyed.
func DownloadCachedWithName(ctx context.Context, url, sha256Hex, cacheDir, displayName string) (string, error) {
	sha256Hex = strings.ToLower(sha256Hex)
	key := cacheKey(url, sha256Hex)
	stagedPath := filepath.Join(cacheDir, key)

	if reused, err := tryReuseStaged(stagedPath, sha256Hex); err != nil {
		return "", err
	} else if reused {
		return stagedPath, nil
	}

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}
	// Old cjv versions wrote partials at "<key>.partial"; sweep them so a
	// stale fixed-name partial cannot be mistaken for a valid staged file.
	if err := removeLegacyPartial(stagedPath + ".partial"); err != nil {
		return "", err
	}
	partialPath, err := newDownloadTempPath(stagedPath)
	if err != nil {
		return "", err
	}

	var lastErr error
	for attempt := range getMaxDownloadRetries() + 1 {
		if attempt > 0 {
			slog.Info("retrying download", "attempt", attempt+1, "max", getMaxDownloadRetries()+1)
		}

		lastErr = downloadOnce(ctx, url, partialPath, displayName, sha256Hex)
		if lastErr == nil {
			if sha256Hex == "" {
				if err := verifyStagedFile(partialPath, sha256Hex); err != nil {
					cleanupDownloadTemp(partialPath)
					return "", fmt.Errorf("downloaded file is not a valid archive: %w", err)
				}
			}
			if err := utils.RenameRetry(partialPath, stagedPath); err != nil {
				return "", fmt.Errorf("promote staged file %s: %w", stagedPath, err)
			}
			return stagedPath, nil
		}

		var nre *nonRetriableError
		if errors.As(lastErr, &nre) {
			cleanupDownloadTemp(partialPath)
			break
		}
		// Retriable error: keep .partial-* on disk for the next attempt.
	}
	cleanupDownloadTemp(partialPath)
	return "", lastErr
}

func downloadDisplayName(rawURL, explicitName string) string {
	name := strings.TrimSpace(explicitName)
	if name == "" {
		if u, err := url.Parse(rawURL); err == nil {
			base := path.Base(u.Path)
			if base != "." && base != "/" {
				name = base
			}
		}
	}
	if name == "" {
		name = "download"
	}
	return fitProgressLabel(name)
}

func fitProgressLabel(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "download"
	}
	runes := []rune(name)
	if len(runes) <= maxProgressNameWidth {
		return name
	}
	return string(runes[:maxProgressNameWidth-3]) + "..."
}

// tryReuseStaged checks for a leftover staged file from a prior run. Returns
// (true, nil) if the file passes verification and can be reused as-is,
// (false, nil) if no usable file is present (callers should download fresh),
// or (false, err) on a fatal verification problem the caller cannot recover.
func tryReuseStaged(stagedPath, sha256Hex string) (bool, error) {
	if _, err := os.Stat(stagedPath); err != nil {
		return false, nil
	}
	slog.Info("staged download already present", "path", stagedPath)
	verifyErr := verifyStagedFile(stagedPath, sha256Hex)
	if verifyErr == nil {
		return true, nil
	}
	var mismatchErr *cjverr.ChecksumMismatchError
	if !errors.As(verifyErr, &mismatchErr) && sha256Hex != "" {
		return false, verifyErr
	}
	if mismatchErr != nil {
		slog.Warn("staged download checksum mismatch; redownloading", "path", stagedPath, "expected", mismatchErr.Expected, "actual", mismatchErr.Actual)
	} else {
		slog.Warn("staged download is invalid; redownloading", "path", stagedPath, "error", verifyErr)
	}
	if err := utils.RemoveAllRetry(stagedPath); err != nil {
		return false, fmt.Errorf("remove corrupt staged file %s: %w", stagedPath, err)
	}
	return false, nil
}

func removeLegacyPartial(path string) error {
	if err := utils.RemoveAllRetry(path); err != nil {
		return fmt.Errorf("remove stale partial download %s: %w", path, err)
	}
	return nil
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

// CleanupDownload removes a staged download once its archive has been
// extracted, keeping cacheDir empty in steady state. Failures
// during install must NOT call this: leaving the file on disk lets the next
// run reuse it instead of re-downloading.
func CleanupDownload(stagedPath string) error {
	return utils.RemoveAllRetry(stagedPath)
}

// VerifyArchive validates a local archive file the same way a freshly downloaded
// one is checked: against sha256Hex when non-empty, otherwise that it is a
// non-empty file carrying a recognized archive magic header. It lets the
// `toolchain link` local-archive path vet a user-supplied file before extracting
// it, without moving it through the download cache.
func VerifyArchive(path, sha256Hex string) error {
	return verifyStagedFile(path, strings.ToLower(sha256Hex))
}

func verifyStagedFile(path, sha256Hex string) error {
	if sha256Hex == "" {
		// No checksum available (e.g. nightly builds). Verify the staged
		// file is non-empty and has a valid archive magic header to catch
		// corrupt or truncated downloads.
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.Size() == 0 {
			return fmt.Errorf("staged file %s is empty", path)
		}
		if err := verifyArchiveMagic(path); err != nil {
			return fmt.Errorf("staged file %s is not a valid archive: %w", path, err)
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
		return fmt.Errorf("hash staged file %s: %w", path, err)
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
		return &nonRetriableError{err: fmt.Errorf("promote downloaded file to %s: %w", dest, err)}
	}
	return nil
}

func downloadOnce(ctx context.Context, url, tmpPath, displayName, sha256Hex string) error {
	client := HTTPClient()
	displayName = downloadDisplayName(url, displayName)

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
		if existingSize > 0 {
			if err := validateContentRangeStart(resp.Header.Get("Content-Range"), existingSize); err != nil {
				cleanupDownloadTemp(tmpPath)
				return fmt.Errorf("invalid resume response for %s: %w", url, err)
			}
		}
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
		return fmt.Errorf("cannot resume download: server rejected range for %s", url)
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

	// Suppress the animated progress bar in non-interactive contexts (CI, piped output).
	interactive := isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())

	src := io.Reader(resp.Body)
	var progressDone func(error)
	if interactive && totalSize > 0 {
		src, progressDone = newProgressReader(src, totalSize, existingSize, displayName)
	}

	reader := io.TeeReader(src, hasher)
	if _, err := io.Copy(f, reader); err != nil {
		if progressDone != nil {
			progressDone(err)
		}
		return fmt.Errorf("download write: %w", err)
	}
	if progressDone != nil {
		progressDone(nil)
	}

	if err := verifyChecksum(hasher, sha256Hex); err != nil {
		return &nonRetriableError{err: err}
	}

	return nil
}

func newProgressReader(src io.Reader, totalSize, existingSize int64, displayName string) (io.Reader, func(error)) {
	p := mpb.New(
		mpb.WithOutput(os.Stderr),
		mpb.WithRefreshRate(100*time.Millisecond),
	)
	bar := p.New(totalSize,
		mpb.BarStyle().Lbound("|").Filler("=").Tip(">").Padding(" ").Rbound("|"),
		mpb.BarWidth(downloadBarWidth),
		mpb.PrependDecorators(
			decor.Name(displayName, decor.WC{C: decor.DindentRight | decor.DextraSpace}),
			decor.Percentage(decor.WC{C: decor.DindentRight | decor.DextraSpace}),
		),
		mpb.AppendDecorators(
			decor.CountersKibiByte("% .1f / % .1f"),
		),
	)
	if existingSize > 0 {
		bar.SetCurrent(existingSize)
	}
	proxyReader := bar.ProxyReader(src)
	done := func(err error) {
		if err != nil {
			bar.Abort(false)
		} else {
			bar.SetTotal(totalSize, true)
		}
		_ = proxyReader.Close()
		p.Wait()
	}
	return proxyReader, done
}

func validateContentRangeStart(header string, expectedStart int64) error {
	if header == "" {
		return fmt.Errorf("missing Content-Range header")
	}
	unit, spec, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(unit, "bytes") {
		return fmt.Errorf("invalid Content-Range header %q", header)
	}
	rangeSpec, _, ok := strings.Cut(spec, "/")
	if !ok {
		return fmt.Errorf("invalid Content-Range header %q", header)
	}
	startText, endText, ok := strings.Cut(rangeSpec, "-")
	if !ok || startText == "" || endText == "" {
		return fmt.Errorf("invalid Content-Range header %q", header)
	}
	start, err := strconv.ParseInt(startText, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid Content-Range start %q: %w", startText, err)
	}
	if start != expectedStart {
		return fmt.Errorf("Content-Range starts at %d, expected %d", start, expectedStart)
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
