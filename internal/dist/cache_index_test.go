package dist

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCacheListsAndCleansDamagedAndOrphanedEntries(t *testing.T) {
	directory := t.TempDir()
	payload := []byte("valid artifact")
	digest := fmt.Sprintf("%x", sha256.Sum256(payload))
	key := cacheID("artifact", "https://example.invalid/sdk.zip", digest)
	require.NoError(t, os.WriteFile(filepath.Join(directory, key+".data"), payload, 0o600))
	require.NoError(t, writeCacheRecord(t.Context(), directory, CacheEntry{Key: key, Kind: "artifact", URL: "https://example.invalid/sdk.zip", SHA256: digest, Size: int64(len(payload)), Updated: time.Now()}))
	damaged := cacheID("metadata", "https://example.invalid/broken", "")
	orphan := cacheID("metadata", "https://example.invalid/orphan", "")
	for name, content := range map[string]string{
		damaged + ".json":                  "{",
		damaged + ".data":                  "broken",
		orphan + ".data":                   "orphan",
		"." + key + ".data.partial-123456": "unfinished",
		".artifact-123456":                 "legacy temporary",
		"notes.txt":                        "user data",
		".artifact-user":                   "unrelated file",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(directory, name), []byte(content), 0o600))
	}
	ownedLookingDirectory := filepath.Join(directory, strings.Repeat("f", 64)+".data")
	require.NoError(t, os.Mkdir(ownedLookingDirectory, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(ownedLookingDirectory, "keep"), []byte("keep"), 0o600))
	entries, err := ListCache(directory)
	require.NoError(t, err)
	require.Len(t, entries, 5)
	byKey := make(map[string]CacheEntry)
	for _, entry := range entries {
		byKey[entry.Key] = entry
	}
	require.Empty(t, byKey[key].Problem)
	require.Equal(t, "invalid", byKey[damaged].Kind)
	require.Contains(t, byKey[damaged].Problem, "invalid cache metadata")
	require.Equal(t, "orphan", byKey[orphan].Kind)
	removed, err := PruneCache(t.Context(), directory, time.Time{})
	require.NoError(t, err)
	require.Equal(t, 5, removed)
	entries, err = ListCache(directory)
	require.NoError(t, err)
	require.Empty(t, entries)
	require.FileExists(t, filepath.Join(directory, "notes.txt"))
	require.FileExists(t, filepath.Join(directory, ".artifact-user"))
	require.FileExists(t, filepath.Join(ownedLookingDirectory, "keep"))
	require.FileExists(t, directory+".lock")
	require.DirExists(t, directory+".locks")
}

func TestCachePruneAgesOrphansAndTemporaryFilesByModificationTime(t *testing.T) {
	directory := t.TempDir()
	old := time.Now().Add(-48 * time.Hour)
	cutoff := time.Now().Add(-24 * time.Hour)
	oldKey := cacheID("metadata", "https://example.invalid/old", "")
	recentKey := cacheID("metadata", "https://example.invalid/recent", "")
	for _, name := range []string{oldKey + ".data", recentKey + ".data", "." + oldKey + ".json.partial-123456"} {
		path := filepath.Join(directory, name)
		require.NoError(t, os.WriteFile(path, []byte("incomplete"), 0o600))
		if !strings.HasPrefix(name, recentKey) {
			require.NoError(t, os.Chtimes(path, old, old))
		}
	}
	removed, err := PruneCache(t.Context(), directory, cutoff)
	require.NoError(t, err)
	require.Equal(t, 2, removed)
	require.FileExists(t, filepath.Join(directory, recentKey+".data"))
	require.NoFileExists(t, filepath.Join(directory, oldKey+".data"))
}

func TestCacheCleanDoesNotFollowOwnedSymlinks(t *testing.T) {
	directory := t.TempDir()
	external := filepath.Join(t.TempDir(), "keep")
	require.NoError(t, os.WriteFile(external, []byte("external contents"), 0o600))
	key := cacheID("metadata", "https://example.invalid/link", "")
	if err := os.Symlink(external, filepath.Join(directory, key+".json")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	require.NoError(t, os.Symlink(external, filepath.Join(directory, key+".data")))
	entries, err := ListCache(directory)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Contains(t, entries[0].Problem, "not a regular file")
	removed, err := PruneCache(t.Context(), directory, time.Time{})
	require.NoError(t, err)
	require.Equal(t, 1, removed)
	data, err := os.ReadFile(external)
	require.NoError(t, err)
	require.Equal(t, "external contents", string(data))
}
