package dist

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/fsops"
	"github.com/Zxilly/cjv/internal/progress"
)

type metadataCacheKey struct{}
type cacheReadOnlyKey struct{}

// WithCacheReadOnly permits reads but suppresses metadata cache publication during
// inspection and dry-run operations.
func WithCacheReadOnly(ctx context.Context) context.Context {
	return context.WithValue(ctx, cacheReadOnlyKey{}, true)
}

// CacheEntry describes reusable content, separate from resumable downloads.
type CacheEntry struct {
	Key     string    `json:"key"`
	Kind    string    `json:"kind"`
	URL     string    `json:"url"`
	SHA256  string    `json:"sha256,omitempty"`
	Size    int64     `json:"size"`
	Updated time.Time `json:"updated"`
	Problem string    `json:"problem,omitempty"`
}

func CacheDirectory() (string, error) {
	home, err := config.Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "cache"), nil
}

func cacheID(kind, rawURL, digest string) string {
	sum := sha256.Sum256([]byte(kind + "\x00" + rawURL + "\x00" + strings.ToLower(digest)))
	return hex.EncodeToString(sum[:])
}

func lockCache(ctx context.Context, directory string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(directory), 0o755); err != nil {
		return nil, err
	}
	return fsops.LockFile(ctx, directory+".lock")
}

// Entry locks live outside the cache, so cleaning never replaces lock inodes
// that another process may already be waiting on.
func lockCacheEntry(ctx context.Context, directory, key string) (*os.File, error) {
	locks := directory + ".locks"
	if err := os.MkdirAll(locks, 0o700); err != nil {
		return nil, err
	}
	return fsops.LockFile(ctx, filepath.Join(locks, key))
}

func writeCacheRecord(ctx context.Context, directory string, entry CacheEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return fsops.WriteFileAtomicContext(ctx, filepath.Join(directory, entry.Key+".json"), data, 0o600)
}

func withMetadataCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, metadataCacheKey{}, true)
}

func cachedMetadata(ctx context.Context, rawURL string, fetch func() ([]byte, error)) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	directory, err := CacheDirectory()
	if err != nil {
		return nil, err
	}
	key := cacheID("metadata", rawURL, "")
	path := filepath.Join(directory, key+".data")
	if Offline(ctx) {
		file, err := os.Open(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil, &cjverr.OfflineCacheMissError{Kind: "metadata", URL: redactURL(rawURL)}
		}
		if err != nil {
			return nil, err
		}
		defer file.Close() //nolint:errcheck
		data, err := io.ReadAll(io.LimitReader(file, MaxResponseSize+1))
		if err == nil && len(data) > MaxResponseSize {
			return nil, fmt.Errorf("cached metadata is too large")
		}
		return data, err
	}
	data, err := fetch()
	if err != nil {
		return nil, err
	}
	if readOnly, _ := ctx.Value(cacheReadOnlyKey{}).(bool); readOnly {
		return data, nil
	}
	if err := publishMetadata(ctx, directory, key, rawURL, data); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		// A successful online query does not depend on an optional disk cache.
		slog.Warn("could not cache metadata", "url", redactURL(rawURL), "error", err)
	}
	return data, nil
}

func publishMetadata(ctx context.Context, directory, key, rawURL string, data []byte) error {
	lock, err := lockCacheEntry(ctx, directory, key)
	if err != nil {
		return err
	}
	defer lock.Close() //nolint:errcheck
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	if err := fsops.WriteFileAtomicContext(ctx, filepath.Join(directory, key+".data"), data, 0o600); err != nil {
		return err
	}
	return writeCacheRecord(ctx, directory, CacheEntry{Key: key, Kind: "metadata", URL: rawURL, Size: int64(len(data)), Updated: time.Now().UTC()})
}

// downloadPersistent copies a verified cache hit to the installation's private
// staging area. No hardlinks are used: installed contents cannot mutate cache.
func downloadPersistent(ctx context.Context, directory, staging, rawURL, digest, name string, sink progress.Sink) (string, error) {
	digest = strings.ToLower(digest)
	if digest == "" {
		return DownloadCachedWithName(ctx, rawURL, digest, staging, name, sink)
	}
	key := cacheID("artifact", rawURL, digest)
	lock, err := lockCacheEntry(ctx, directory, key)
	if err != nil {
		return "", err
	}
	defer lock.Close() //nolint:errcheck
	path := filepath.Join(directory, key+".data")
	dest := filepath.Join(staging, cacheKey(rawURL, digest))
	if err := verifyStagedFileContext(ctx, path, digest); err == nil {
		if err := os.MkdirAll(staging, 0o755); err != nil {
			return "", err
		}
		if err := fsops.CopyFileAtomicContext(ctx, path, dest, 0o644); err != nil {
			return "", err
		}
		return dest, nil
	} else if ctx.Err() != nil {
		return "", ctx.Err()
	}
	archive, err := DownloadCachedWithName(ctx, rawURL, digest, staging, name, sink)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	if err := fsops.CopyFileAtomicContext(ctx, archive, path, 0o600); err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	err = writeCacheRecord(ctx, directory, CacheEntry{Key: key, Kind: "artifact", URL: rawURL, SHA256: digest, Size: info.Size(), Updated: time.Now().UTC()})
	return archive, err
}
