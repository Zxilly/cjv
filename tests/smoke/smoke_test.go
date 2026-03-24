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

	"github.com/Zxilly/cjv/internal/dist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var latestNightlyVersion string

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	v, err := dist.FetchLatestNightly(ctx, dist.DefaultNightlyAPIURL)
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dist.DefaultNightlyAPIURL+"?limit=1", nil)
	require.NoError(t, err)

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

func TestSmokeNightlyProbeAllPlatforms(t *testing.T) {
	platforms := []struct {
		goos   string
		goarch string
	}{
		{"windows", "amd64"},
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"linux", "amd64"},
		{"linux", "arm64"},
	}

	for _, p := range platforms {
		t.Run(p.goos+"-"+p.goarch, func(t *testing.T) {
			t.Parallel()
			ctx := testContext(t)
			exists, err := dist.ProbeNightlyVersion(ctx, dist.DefaultNightlyBaseURL, latestNightlyVersion, p.goos, p.goarch)
			require.NoError(t, err, "probe should not error")
			assert.True(t, exists, "nightly %s should be available for %s-%s", latestNightlyVersion, p.goos, p.goarch)
		})
	}
}
