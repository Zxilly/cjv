package dist

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/stretchr/testify/require"
)

func TestNightlyChecksumCacheTracksPresence(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	t.Setenv("CJV_OFFLINE", "")
	digest := strings.Repeat("ab", 32)
	var present atomic.Bool
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if !present.Load() {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(strings.ToUpper(digest) + "\n"))
	}))
	defer server.Close()
	assetURL := server.URL + "/sdk.zip"
	for _, exists := range []bool{false, true, false} {
		present.Store(exists)
		expected := ""
		if exists {
			expected = digest
		}
		actual, err := FetchNightlySHA256(t.Context(), assetURL)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
		before := requests.Load()
		actual, err = FetchNightlySHA256(WithOffline(t.Context(), true), assetURL)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
		require.Equal(t, before, requests.Load(), "offline reads must not contact the source")
	}

	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("offline lookup contacted another source")
		_, _ = w.Write([]byte(digest))
	}))
	defer other.Close()
	_, err := FetchNightlySHA256(WithOffline(t.Context(), true), other.URL+"/sdk.zip")
	require.ErrorIs(t, err, os.ErrNotExist, "a cached 404 must not apply to another distribution source")
}

func TestNightlyChecksumFailuresPreserveCachedState(t *testing.T) {
	for _, initiallyPresent := range []bool{false, true} {
		name := "missing"
		if initiallyPresent {
			name = "present"
		}
		t.Run(name, func(t *testing.T) {
			configtest.Isolate(t, t.TempDir())
			t.Setenv("CJV_OFFLINE", "")
			t.Setenv(config.EnvMaxRetries, "0")
			type response struct {
				status int
				body   string
			}
			expected := ""
			initial := response{status: http.StatusNotFound}
			if initiallyPresent {
				expected = strings.Repeat("ab", 32)
				initial = response{status: http.StatusOK, body: expected}
			}
			var current atomic.Value
			current.Store(initial)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				result := current.Load().(response)
				w.WriteHeader(result.status)
				_, _ = w.Write([]byte(result.body))
			}))
			defer server.Close()
			assetURL := server.URL + "/sdk.zip"
			_, err := FetchNightlySHA256(t.Context(), assetURL)
			require.NoError(t, err)
			for _, invalid := range []response{
				{status: http.StatusServiceUnavailable},
				{status: http.StatusForbidden},
				{status: http.StatusOK},
				{status: http.StatusOK, body: "not-a-digest"},
				{status: http.StatusOK, body: `{"status_code":404}`},
			} {
				current.Store(invalid)
				_, err := FetchNightlySHA256(t.Context(), assetURL)
				require.Error(t, err, "only a real 404 may mean that the sidecar is absent")
				actual, err := FetchNightlySHA256(WithOffline(t.Context(), true), assetURL)
				require.NoError(t, err)
				require.Equal(t, expected, actual, "a failed fetch must leave the last validated cache entry intact")
			}
			server.Close()
			_, err = FetchNightlySHA256(t.Context(), assetURL)
			require.Error(t, err)
			actual, err := FetchNightlySHA256(WithOffline(t.Context(), true), assetURL)
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	}
}

func TestReadOnlyNightlyChecksumDoesNotPublishMissingState(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	t.Setenv("CJV_OFFLINE", "")
	var missing atomic.Bool
	missing.Store(true)
	digest := strings.Repeat("ab", 32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if missing.Load() {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(digest))
	}))
	defer server.Close()
	assetURL := server.URL + "/sdk.zip"
	actual, err := FetchNightlySHA256(WithCacheReadOnly(t.Context()), assetURL)
	require.NoError(t, err)
	require.Empty(t, actual)
	directory, err := CacheDirectory()
	require.NoError(t, err)
	require.NoDirExists(t, directory)
	_, err = FetchNightlySHA256(WithOffline(t.Context(), true), assetURL)
	require.ErrorIs(t, err, os.ErrNotExist)

	missing.Store(false)
	_, err = FetchNightlySHA256(t.Context(), assetURL)
	require.NoError(t, err)
	missing.Store(true)
	actual, err = FetchNightlySHA256(WithCacheReadOnly(t.Context()), assetURL)
	require.NoError(t, err)
	require.Empty(t, actual, "the current read-only lookup must honor the new 404")
	actual, err = FetchNightlySHA256(WithOffline(t.Context(), true), assetURL)
	require.NoError(t, err)
	require.Equal(t, digest, actual, "read-only inspection must not alter the previously cached response")
}
