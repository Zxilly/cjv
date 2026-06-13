package lifecycle

import (
	"context"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stubNightlyChecksum(t *testing.T) {
	t.Helper()
	orig := FetchNightlySHA256
	FetchNightlySHA256 = func(context.Context, string) (string, error) { return "", nil }
	t.Cleanup(func() { FetchNightlySHA256 = orig })
}

func stubLatestNightlyRelease(t *testing.T, fn func(context.Context, string, string) (dist.NightlyRelease, error)) {
	t.Helper()
	orig := FetchLatestNightlyRelease
	FetchLatestNightlyRelease = fn
	t.Cleanup(func() { FetchLatestNightlyRelease = orig })
}

func quietLifecycleOptions() Options {
	return Options{IsJSON: func() bool { return true }}
}

func TestResolveNightlyReleaseSeparatesTagAndAssetVersion(t *testing.T) {
	stubNightlyChecksum(t)

	const tag = "1.1.0-alpha.20260613020028"
	const assetVersion = "1.2.0-alpha.20260613020028"
	resolved, err := resolveNightlyRelease(context.Background(), dist.NightlyRelease{
		TagName: tag,
		Version: assetVersion,
	}, "linux-x64", quietLifecycleOptions())

	require.NoError(t, err)
	assert.Equal(t, "nightly-"+assetVersion, resolved.Name)
	assert.Equal(t, tag, resolved.NightlyReleaseTag)
	assert.Equal(t, assetVersion, resolved.NightlyVersion)
	assert.True(t, strings.Contains(resolved.URL, "/"+tag+"/"), resolved.URL)
	assert.True(t, strings.Contains(resolved.URL, "cangjie-sdk-linux-x64-"+assetVersion+".tar.gz"), resolved.URL)
}

func TestResolveNightlyPinnedAssetVersionUsesLatestReleaseTag(t *testing.T) {
	stubNightlyChecksum(t)

	const tag = "1.1.0-alpha.20260613020028"
	const assetVersion = "1.2.0-alpha.20260613020028"
	settings := config.DefaultSettings()
	settings.GitCodeAPIKey = "test-token"
	stubLatestNightlyRelease(t, func(_ context.Context, _ string, apiKey string) (dist.NightlyRelease, error) {
		assert.Equal(t, "test-token", apiKey)
		return dist.NightlyRelease{TagName: tag, Version: assetVersion}, nil
	})

	resolved, err := resolveNightly(context.Background(), toolchain.ToolchainName{
		Channel: toolchain.Nightly,
		Version: assetVersion,
	}, &settings, "linux-x64", quietLifecycleOptions())

	require.NoError(t, err)
	assert.Equal(t, "nightly-"+assetVersion, resolved.Name)
	assert.Equal(t, tag, resolved.NightlyReleaseTag)
	assert.Equal(t, assetVersion, resolved.NightlyVersion)
	assert.True(t, strings.Contains(resolved.URL, "/"+tag+"/"), resolved.URL)
	assert.True(t, strings.Contains(resolved.URL, "cangjie-sdk-linux-x64-"+assetVersion+".tar.gz"), resolved.URL)
}

func TestResolveNightlyPinnedReleaseTagUsesLatestAssetVersion(t *testing.T) {
	stubNightlyChecksum(t)

	const tag = "1.1.0-alpha.20260613020028"
	const assetVersion = "1.2.0-alpha.20260613020028"
	settings := config.DefaultSettings()
	settings.GitCodeAPIKey = "test-token"
	stubLatestNightlyRelease(t, func(_ context.Context, _ string, _ string) (dist.NightlyRelease, error) {
		return dist.NightlyRelease{TagName: tag, Version: assetVersion}, nil
	})

	resolved, err := resolveNightly(context.Background(), toolchain.ToolchainName{
		Channel: toolchain.Nightly,
		Version: tag,
	}, &settings, "linux-x64", quietLifecycleOptions())

	require.NoError(t, err)
	assert.Equal(t, "nightly-"+assetVersion, resolved.Name)
	assert.True(t, strings.Contains(resolved.URL, "/"+tag+"/"), resolved.URL)
	assert.True(t, strings.Contains(resolved.URL, "cangjie-sdk-linux-x64-"+assetVersion+".tar.gz"), resolved.URL)
}

func TestResolveTargetToolchainKeepsNightlyReleaseTag(t *testing.T) {
	stubNightlyChecksum(t)

	const tag = "1.1.0-alpha.20260613020028"
	const assetVersion = "1.2.0-alpha.20260613020028"
	settings := config.DefaultSettings()
	settings.DefaultHost = "linux-amd64"
	fetcher := NewManifestFetcher("", quietLifecycleOptions())

	resolved, err := resolveTargetToolchain(context.Background(),
		toolchain.ToolchainName{Channel: toolchain.Nightly, Version: assetVersion},
		&settings,
		fetcher,
		"ohos",
		ResolvedToolchain{NightlyReleaseTag: tag, NightlyVersion: assetVersion},
	)

	require.NoError(t, err)
	assert.Equal(t, "nightly-"+assetVersion+"-linux-x64-ohos", resolved.Name)
	assert.True(t, strings.Contains(resolved.URL, "/"+tag+"/"), resolved.URL)
	assert.True(t, strings.Contains(resolved.URL, "cangjie-sdk-linux-x64-ohos-"+assetVersion+".tar.gz"), resolved.URL)
}
