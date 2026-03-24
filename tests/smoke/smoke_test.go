//go:build smoke

package smoke

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	latestNightlyVersion string
	gitCodeAPIKey        = os.Getenv(config.EnvGitCodeAPIKey)
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	v, err := dist.FetchLatestNightly(ctx, dist.DefaultNightlyAPIURL, gitCodeAPIKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch latest nightly version: %v\n", err)
		os.Exit(1)
	}
	latestNightlyVersion = v

	os.Exit(m.Run())
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestSmokeNightlyAPIReturnsJSON(t *testing.T) {
	ctx := testContext(t)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dist.DefaultNightlyAPIURL+"?per_page=1", nil)
	require.NoError(t, err)
	if gitCodeAPIKey != "" {
		req.Header.Set(dist.GitCodeTokenHeader, gitCodeAPIKey)
	}

	resp, err := dist.HTTPClient().Do(req)
	require.NoError(t, err)
	defer func() {
		io.Copy(io.Discard, resp.Body) //nolint:errcheck
		resp.Body.Close()              //nolint:errcheck
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "nightly API should return 200")

	ct := resp.Header.Get("Content-Type")
	assert.Contains(t, ct, "application/json", "nightly API should return JSON, got Content-Type: %s", ct)
}

func TestSmokeNightlyFetchLatest(t *testing.T) {
	assert.NotEmpty(t, latestNightlyVersion, "latest nightly version should not be empty")

	idx := strings.LastIndex(latestNightlyVersion, ".")
	require.Greater(t, idx, 0, "version %q should contain a dot", latestNightlyVersion)
	assert.Regexp(t, `^\d{10,}$`, latestNightlyVersion[idx+1:],
		"latest nightly version %q should end with a numeric timestamp", latestNightlyVersion)
}

