package component

import (
	"testing"

	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAssetURL_Stdx(t *testing.T) {
	spec, err := SpecFor(Stdx)
	require.NoError(t, err)

	tests := []struct {
		name     string
		tc       toolchain.ToolchainName
		platform string
		wantURL  string
	}{
		{
			name:     "lts linux aarch64",
			tc:       toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.1.0-beta.25"},
			platform: "linux-arm64",
			wantURL:  "https://gitcode.com/Cangjie/cangjie_stdx/releases/download/v1.1.0-beta.25/cangjie-stdx-linux-aarch64-1.1.0-beta.25.1.tar.gz",
		},
		{
			name:     "sts windows x64",
			tc:       toolchain.ToolchainName{Channel: toolchain.STS, Version: "1.1.0-beta.25"},
			platform: "win32-x64",
			wantURL:  "https://gitcode.com/Cangjie/cangjie_stdx/releases/download/v1.1.0-beta.25/cangjie-stdx-windows-x64-1.1.0-beta.25.1.zip",
		},
		{
			name:     "nightly linux aarch64",
			tc:       toolchain.ToolchainName{Channel: toolchain.Nightly, Version: "1.1.0-alpha.20260506010057"},
			platform: "linux-arm64",
			wantURL:  "https://gitcode.com/Cangjie/nightly_build/releases/download/1.1.0-alpha.20260506010057/cangjie-stdx-linux-aarch64-1.1.0-alpha.20260506010057.1.tar.gz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveAssetURL(spec, tt.tc, tt.platform)
			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, got)
		})
	}
}

func TestResolveAssetURL_StdxRejectsTargetSuffix(t *testing.T) {
	spec, _ := SpecFor(Stdx)
	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}
	_, err := ResolveAssetURL(spec, tc, "linux-x64-ohos")
	assert.Error(t, err)
}

func TestResolveAssetURL_DocsAndStdxDocs_Nightly(t *testing.T) {
	tc := toolchain.ToolchainName{Channel: toolchain.Nightly, Version: "1.1.0-alpha.20260506010057"}

	docsSpec, _ := SpecFor(Docs)
	docsURL, err := ResolveAssetURL(docsSpec, tc, "")
	require.NoError(t, err)
	assert.Equal(t,
		"https://gitcode.com/Cangjie/nightly_build/releases/download/1.1.0-alpha.20260506010057/cangjie-docs-html-1.1.0-alpha.20260506010057.tar.gz",
		docsURL)

	stdxDocsSpec, _ := SpecFor(StdxDocs)
	stdxDocsURL, err := ResolveAssetURL(stdxDocsSpec, tc, "")
	require.NoError(t, err)
	assert.Equal(t,
		"https://gitcode.com/Cangjie/nightly_build/releases/download/1.1.0-alpha.20260506010057/cangjie-stdx-docs-html-1.1.0-alpha.20260506010057.1.tar.gz",
		stdxDocsURL)
}

func TestResolveAssetURL_Docs_LTSSTS(t *testing.T) {
	docsSpec, _ := SpecFor(Docs)

	ltsURL, err := ResolveAssetURL(docsSpec, toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.1.0-beta.25"}, "")
	require.NoError(t, err)
	assert.Equal(t,
		"https://github.com/Zxilly/cangjie-docs-bundle/releases/download/1.1.0-beta.25/cangjie-docs-html-1.1.0-beta.25.tar.gz",
		ltsURL)

	stsURL, err := ResolveAssetURL(docsSpec, toolchain.ToolchainName{Channel: toolchain.STS, Version: "1.0.5"}, "")
	require.NoError(t, err)
	assert.Equal(t,
		"https://github.com/Zxilly/cangjie-docs-bundle/releases/download/1.0.5/cangjie-docs-html-1.0.5.tar.gz",
		stsURL)
}

func TestResolveAssetURL_StdxDocs_LTSSTS(t *testing.T) {
	stdxDocsSpec, _ := SpecFor(StdxDocs)

	ltsURL, err := ResolveAssetURL(stdxDocsSpec, toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.1.0"}, "")
	require.NoError(t, err)
	assert.Equal(t,
		"https://gitcode.com/Cangjie/cangjie_stdx/releases/download/v1.1.0.1/cangjie-stdx-docs-html-1.1.0.1.tar.gz",
		ltsURL)

	stsURL, err := ResolveAssetURL(stdxDocsSpec, toolchain.ToolchainName{Channel: toolchain.STS, Version: "1.1.0-beta.25"}, "")
	require.NoError(t, err)
	assert.Equal(t,
		"https://gitcode.com/Cangjie/cangjie_stdx/releases/download/v1.1.0-beta.25.1/cangjie-stdx-docs-html-1.1.0-beta.25.1.tar.gz",
		stsURL)
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

	assert.NotContains(t, got, Stdx)
	assert.Contains(t, got, Docs)
	assert.Contains(t, got, StdxDocs)
}
