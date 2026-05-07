package selfupdate

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateSkipsNetworkForUnsupportedOrDevBuilds(t *testing.T) {
	require.NoError(t, Update(context.Background(), "", "1.0.0"))
	require.NoError(t, Update(context.Background(), PlaceholderURL, "1.0.0"))
	require.NoError(t, Update(context.Background(), "https://github.com/Zxilly/cjv/releases", "dev"))
}

func TestUpdateReturnsDetectLatestError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Update(ctx, "https://github.com/Zxilly/cjv/releases", "1.0.0")

	require.Error(t, err)
}

func TestExtractSlugFallbacks(t *testing.T) {
	assert.Equal(t, "owner/repo", extractSlug("https://github.com/owner/repo/releases"))
	assert.Equal(t, "https://github.com/owner-only", extractSlug("https://github.com/owner-only"))
	assert.Equal(t, "not a url", extractSlug("not a url"))
}
