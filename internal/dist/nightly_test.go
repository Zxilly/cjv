package dist

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSHA256(t *testing.T) {
	digest := strings.Repeat("ab", 32)
	assert.Equal(t, digest, parseSHA256(digest+"\n"))
	assert.Equal(t, digest, parseSHA256(strings.ToUpper(digest)))
	assert.Empty(t, parseSHA256("abc"))
	assert.Empty(t, parseSHA256(digest+" cangjie-sdk.zip"))
	assert.Empty(t, parseSHA256(strings.Repeat("z", 64)))
}

func TestFetchNightlySHA256(t *testing.T) {
	digest := strings.Repeat("ab", 32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/sdk.zip.sha256", r.URL.Path)
		_, _ = w.Write([]byte(strings.ToUpper(digest) + "\n"))
	}))
	defer server.Close()

	got, err := FetchNightlySHA256(context.Background(), server.URL+"/sdk.zip")
	require.NoError(t, err)
	assert.Equal(t, digest, got)
}

func TestFetchNightlySHA256MissingAndMalformed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/missing.zip.sha256":
			http.NotFound(w, r)
		case "/invalid.zip.sha256":
			_, _ = w.Write([]byte("not-a-digest"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	got, err := FetchNightlySHA256(context.Background(), server.URL+"/missing.zip")
	require.NoError(t, err)
	assert.Empty(t, got)

	_, err = FetchNightlySHA256(context.Background(), server.URL+"/invalid.zip")
	assert.Error(t, err)

	server.Close()
	_, err = FetchNightlySHA256(context.Background(), server.URL+"/network-failure.zip")
	assert.Error(t, err)
}

func TestHTTPClientReportsAppVersionInUserAgent(t *testing.T) {
	old := AppVersion
	AppVersion = "1.2.3-test"
	t.Cleanup(func() { AppVersion = old })

	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.UserAgent()
	}))
	defer server.Close()

	resp, err := newHTTPClient().Get(server.URL)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, "cjv/1.2.3-test", got)
}
