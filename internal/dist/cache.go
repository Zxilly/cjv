package dist

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

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

func writeCacheRecord(directory string, entry CacheEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return fsops.WriteFileAtomic(filepath.Join(directory, entry.Key+".json"), data, 0o600)
}

func withMetadataCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, metadataCacheKey{}, true)
}

func cachedMetadata(ctx context.Context, rawURL string, fetch func() ([]byte, error)) ([]byte, error) {
	directory, err := CacheDirectory()
	if err != nil {
		return nil, err
	}
	key := cacheID("metadata", rawURL, "")
	path := filepath.Join(directory, key+".data")
	if Offline(ctx) {
		file, err := os.Open(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("offline mode: metadata for %s is not cached: %w", rawURL, os.ErrNotExist)
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
	lock, err := lockCache(ctx, directory)
	if err != nil {
		return nil, err
	}
	defer lock.Close() //nolint:errcheck
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, err
	}
	if err := fsops.WriteFileAtomic(path, data, 0o600); err != nil {
		return nil, err
	}
	err = writeCacheRecord(directory, CacheEntry{Key: key, Kind: "metadata", URL: rawURL, Size: int64(len(data)), Updated: time.Now().UTC()})
	return data, err
}

// downloadPersistent copies a verified cache hit to the installation's private
// staging area. No hardlinks are used: installed contents cannot mutate cache.
func downloadPersistent(ctx context.Context, directory, staging, rawURL, digest, name string, sink progress.Sink) (string, error) {
	digest = strings.ToLower(digest)
	if digest == "" {
		return DownloadCachedWithName(ctx, rawURL, digest, staging, name, sink)
	}
	key := cacheID("artifact", rawURL, digest)
	lock, err := lockCache(ctx, directory)
	if err != nil {
		return "", err
	}
	defer lock.Close() //nolint:errcheck
	path := filepath.Join(directory, key+".data")
	dest := filepath.Join(staging, cacheKey(rawURL, digest))
	if err := verifyStagedFile(path, digest); err == nil {
		if err := os.MkdirAll(staging, 0o755); err != nil {
			return "", err
		}
		if err := fsops.CopyFile(path, dest, 0o644); err != nil {
			return "", err
		}
		return dest, nil
	}
	archive, err := DownloadCachedWithName(ctx, rawURL, digest, staging, name, sink)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	// Publish through a sibling temporary file so readers never see partial data.
	tmp, err := os.CreateTemp(directory, ".artifact-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		return "", err
	}
	defer os.Remove(tmpPath) //nolint:errcheck
	if err := fsops.CopyFile(archive, tmpPath, 0o600); err != nil {
		return "", err
	}
	if err := fsops.RenameRetry(tmpPath, path); err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	err = writeCacheRecord(directory, CacheEntry{Key: key, Kind: "artifact", URL: rawURL, SHA256: digest, Size: info.Size(), Updated: time.Now().UTC()})
	return archive, err
}

func ListCache(directory string) ([]CacheEntry, error) {
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return []CacheEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := []CacheEntry{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		key := strings.TrimSuffix(entry.Name(), ".json")
		decoded, err := hex.DecodeString(key)
		if err != nil || len(decoded) != sha256.Size {
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			return nil, err
		}
		var record CacheEntry
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, err
		}
		if record.Key != key || cacheID(record.Kind, record.URL, record.SHA256) != key {
			return nil, fmt.Errorf("invalid cache record %s", entry.Name())
		}
		result = append(result, record)
	}
	return result, nil
}

// PruneCache removes known entries older than cutoff; a zero cutoff clears all.
// It never traverses paths taken from metadata and leaves lock files in place.
func PruneCache(ctx context.Context, directory string, cutoff time.Time) (int, error) {
	lock, err := lockCache(ctx, directory)
	if err != nil {
		return 0, err
	}
	defer lock.Close() //nolint:errcheck
	entries, err := ListCache(directory)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, entry := range entries {
		if !cutoff.IsZero() && !entry.Updated.Before(cutoff) {
			continue
		}
		for _, suffix := range []string{".data", ".json"} {
			if err := os.Remove(filepath.Join(directory, entry.Key+suffix)); err != nil && !errors.Is(err, os.ErrNotExist) {
				return removed, err
			}
		}
		removed++
	}
	return removed, nil
}
