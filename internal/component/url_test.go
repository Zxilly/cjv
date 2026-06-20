package component

import (
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// manifestWithComponents builds a manifest carrying only a single version's
// component set for the given channel — enough to drive ResolveAssetURL without
// going through ParseManifest validation.
func manifestWithComponents(channel toolchain.Channel, version string, set dist.ComponentSet) *dist.Manifest {
	var m dist.Manifest
	ci := dist.ChannelInfo{Components: map[string]dist.ComponentSet{version: set}}
	switch channel {
	case toolchain.LTS:
		m.Channels.LTS = ci
	case toolchain.STS:
		m.Channels.STS = ci
	}
	return &m
}

func TestResolveAssetURL_StdxReadsManifestByPlatform(t *testing.T) {
	spec, err := SpecFor(Stdx)
	require.NoError(t, err)

	const version = "1.1.0-beta.25"
	mf := manifestWithComponents(toolchain.STS, version, dist.ComponentSet{
		Stdx: map[string]dist.ComponentInfo{
			// tag is the bare SDK version here, the asset keeps the .1 revision —
			// the exact inconsistency the manifest exists to capture verbatim.
			"linux-aarch64": {Name: "cangjie-stdx-linux-aarch64-1.1.0-beta.25.1.zip", URL: "https://example/stdx/linux-aarch64"},
			"windows-x64":   {Name: "cangjie-stdx-windows-x64-1.1.0-beta.25.1.zip", URL: "https://example/stdx/windows-x64"},
		},
	})

	tests := []struct {
		name    string
		tuple   string
		wantURL string
	}{
		{name: "lts linux aarch64 host tuple", tuple: "linux-arm64", wantURL: "https://example/stdx/linux-aarch64"},
		{name: "windows x64 host tuple", tuple: "win32-x64", wantURL: "https://example/stdx/windows-x64"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveAssetURL(spec, toolchain.ToolchainName{Channel: toolchain.STS, Version: version}, tt.tuple, mf)
			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, got)
		})
	}
}

func TestResolveAssetURL_StdxTargetTupleMapsToStdxPlatform(t *testing.T) {
	spec, err := SpecFor(Stdx)
	require.NoError(t, err)

	mf := manifestWithComponents(toolchain.STS, "1.1.0", dist.ComponentSet{
		Stdx: map[string]dist.ComponentInfo{
			"ohos-aarch64":  {URL: "https://example/stdx/ohos-aarch64"},
			"ohos-x64":      {URL: "https://example/stdx/ohos-x64"},
			"android-arm32": {URL: "https://example/stdx/android-arm32"},
		},
	})
	tc := toolchain.ToolchainName{Channel: toolchain.STS, Version: "1.1.0"}

	tests := map[string]string{
		"linux-x64-ohos":          "https://example/stdx/ohos-aarch64",
		"linux-x64-ohos-x64":      "https://example/stdx/ohos-x64",
		"linux-x64-android-arm32": "https://example/stdx/android-arm32",
	}
	for tuple, wantURL := range tests {
		t.Run(tuple, func(t *testing.T) {
			got, err := ResolveAssetURL(spec, tc, tuple, mf)
			require.NoError(t, err)
			assert.Equal(t, wantURL, got)
		})
	}
}

func TestResolveAssetURL_StdxMissingPlatformReportsNotPublished(t *testing.T) {
	spec, _ := SpecFor(Stdx)
	mf := manifestWithComponents(toolchain.STS, "1.1.0", dist.ComponentSet{
		Stdx: map[string]dist.ComponentInfo{"linux-x64": {URL: "https://example/stdx/linux-x64"}},
	})
	// win32-x64 maps to "windows-x64", which this version does not publish.
	_, err := ResolveAssetURL(spec, toolchain.ToolchainName{Channel: toolchain.STS, Version: "1.1.0"}, "win32-x64", mf)
	var notPub *cjverr.ComponentNotPublishedError
	require.ErrorAs(t, err, &notPub)
	assert.Equal(t, "windows-x64", notPub.Target)
}

func TestResolveAssetURL_StdxRequiresTuple(t *testing.T) {
	spec, _ := SpecFor(Stdx)
	mf := manifestWithComponents(toolchain.LTS, "1.0.5", dist.ComponentSet{
		Stdx: map[string]dist.ComponentInfo{"linux-x64": {URL: "https://example/stdx"}},
	})
	_, err := ResolveAssetURL(spec, toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}, "", mf)
	assert.Error(t, err)
}

func TestResolveAssetURL_DocsAndStdxDocsReadManifest(t *testing.T) {
	docsSpec, _ := SpecFor(Docs)
	stdxDocsSpec, _ := SpecFor(StdxDocs)
	mf := manifestWithComponents(toolchain.LTS, "1.0.5", dist.ComponentSet{
		Docs:     &dist.ComponentInfo{URL: "https://example/docs"},
		StdxDocs: &dist.ComponentInfo{URL: "https://example/stdx-docs"},
	})
	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}

	docsURL, err := ResolveAssetURL(docsSpec, tc, "", mf)
	require.NoError(t, err)
	assert.Equal(t, "https://example/docs", docsURL)

	stdxDocsURL, err := ResolveAssetURL(stdxDocsSpec, tc, "", mf)
	require.NoError(t, err)
	assert.Equal(t, "https://example/stdx-docs", stdxDocsURL)
}

func TestResolveAssetURL_LTSRequiresManifest(t *testing.T) {
	spec, _ := SpecFor(Docs)
	_, err := ResolveAssetURL(spec, toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}, "", nil)
	require.Error(t, err)
}

func TestResolveAssetURL_VersionNotInManifest(t *testing.T) {
	spec, _ := SpecFor(Docs)
	mf := manifestWithComponents(toolchain.STS, "1.1.0", dist.ComponentSet{
		Docs: &dist.ComponentInfo{URL: "https://example/docs"},
	})
	_, err := ResolveAssetURL(spec, toolchain.ToolchainName{Channel: toolchain.STS, Version: "9.9.9"}, "", mf)
	var notPub *cjverr.ComponentNotPublishedError
	require.ErrorAs(t, err, &notPub)
}

func TestNormalizeList(t *testing.T) {
	got, err := NormalizeList([]string{"stdx", "docs,stdx-docs", "stdx"})
	require.NoError(t, err)
	assert.Equal(t, []Name{Stdx, Docs, StdxDocs}, got)

	_, err = NormalizeList([]string{"unknown"})
	assert.Error(t, err)
}

func TestAvailableComponentsFiltersBySelectedPlatform(t *testing.T) {
	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}

	got := AvailableComponents(tc, "linux-x64-ohos")

	// stdx is now available for cross-compile target tuples (target stdx).
	assert.Contains(t, got, Stdx)
	assert.Contains(t, got, Docs)
	assert.Contains(t, got, StdxDocs)
}

func TestResolveAssetURLNightlyUsesReleaseMetadata(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)

	const tag = "1.1.0-alpha.20260613020028"
	const assetVersion = "1.2.0-alpha.20260613020028"
	tc := toolchain.ToolchainName{Channel: toolchain.Nightly, Version: assetVersion}
	roots, err := RootsFor(tc.String())
	require.NoError(t, err)
	require.NoError(t, toolchain.WriteNightlyReleaseMetadata(roots.TcDir, toolchain.NightlyReleaseMetadata{
		ReleaseTag: tag,
		Version:    assetVersion,
	}))

	docsSpec, err := SpecFor(Docs)
	require.NoError(t, err)
	docsURL, err := ResolveAssetURL(docsSpec, tc, "", nil)
	require.NoError(t, err)
	assert.Equal(t,
		"https://gitcode.com/Cangjie/nightly_build/releases/download/"+tag+"/cangjie-docs-html-"+assetVersion+".tar.gz",
		docsURL)

	stdxSpec, err := SpecFor(Stdx)
	require.NoError(t, err)
	stdxURL, err := ResolveAssetURL(stdxSpec, tc, "linux-x64", nil)
	require.NoError(t, err)
	assert.Equal(t,
		"https://gitcode.com/Cangjie/nightly_build/releases/download/"+tag+"/cangjie-stdx-linux-x64-"+assetVersion+".1.zip",
		stdxURL)
}
