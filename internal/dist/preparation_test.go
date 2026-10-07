package dist

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreparationRetainsVerifiedDownloadsUntilPublication(t *testing.T) {
	payload := []byte("verified archive payload")
	digest := sha256.Sum256(payload)
	checksum := hex.EncodeToString(digest[:])
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write(payload)
	}))
	defer server.Close()
	downloads := filepath.Join(t.TempDir(), "downloads")
	preparation, err := BeginPreparation(t.Context(), downloads)
	require.NoError(t, err)
	defer preparation.Close()
	archive, err := preparation.Download(t.Context(), server.URL, checksum, "sdk.zip", nil)
	require.NoError(t, err)
	scratch, err := preparation.TempDir(".cjv-stage-*")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(scratch, "unpublished"), payload, 0o644))

	// Failure keeps only reusable downloads, never private extraction trees.
	require.NoError(t, preparation.Close())
	assert.FileExists(t, archive)
	assert.NoDirExists(t, scratch)
	retry, err := BeginPreparation(t.Context(), downloads)
	require.NoError(t, err)
	defer retry.Close()
	_, err = retry.Download(t.Context(), server.URL, checksum, "sdk.zip", nil)
	require.NoError(t, err)
	assert.EqualValues(t, 1, requests.Load())
	retry.Complete()
	require.NoError(t, retry.Close())
	assert.NoFileExists(t, archive)
}

func TestPreparationCanceledWaitPreservesLiveScratch(t *testing.T) {
	downloads := filepath.Join(t.TempDir(), "downloads")
	preparation, err := BeginPreparation(t.Context(), downloads)
	require.NoError(t, err)
	defer preparation.Close()
	scratch, err := preparation.TempDir(".cjv-stage-*")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = BeginPreparation(ctx, downloads)
	require.ErrorIs(t, err, context.Canceled)
	assert.DirExists(t, scratch)
	// Releasing the first scope allows the next operation to acquire the same
	// persistent lock inode; no separate cleanup call is required.
	require.NoError(t, preparation.Close())
	next, err := BeginPreparation(t.Context(), downloads)
	require.NoError(t, err)
	require.NoError(t, next.Close())
}
