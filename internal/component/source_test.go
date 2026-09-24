package component

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sourceWithComponents serves a manifest carrying set as the component set of
// one version on channel, and returns the production distribution source
// that reads it. Every channel gets a placeholder release so the manifest
// passes validation; a nightly set is served from the sibling nightly.json,
// exactly as a split distribution publishes it.
func sourceWithComponents(t *testing.T, channel toolchain.Channel, version string, set dist.ComponentSet) *dist.Source {
	t.Helper()
	const sha = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	placeholder := func(v string) dist.ChannelInfo {
		return dist.ChannelInfo{Latest: v, Versions: map[string]map[string]dist.DownloadInfo{
			v: {"linux-x64": {Name: "sdk.zip", URL: "https://example.invalid/sdk.zip", SHA256: sha}},
		}}
	}
	var m dist.Manifest
	m.Channels.LTS = placeholder("0.0.1")
	m.Channels.STS = placeholder("0.0.2")
	ci := placeholder(version)
	ci.Components = map[string]dist.ComponentSet{version: set}
	var nightly *dist.ChannelInfo
	switch channel {
	case toolchain.LTS:
		m.Channels.LTS = ci
	case toolchain.STS:
		m.Channels.STS = ci
	case toolchain.Nightly:
		nightly = &ci
	}
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/versions.json", func(w http.ResponseWriter, _ *http.Request) {
		assert.NoError(t, json.NewEncoder(w).Encode(m))
	})
	if nightly != nil {
		mux.HandleFunc("/nightly.json", func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(nightly))
		})
	}
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/versions.json"
	source, err := dist.NewSource(&settings)
	require.NoError(t, err)
	return source
}

// recordingArchiveServer serves a stdx-shaped zip for every request and
// records the paths asked for, so a test can assert which manifest link an
// install resolved.
func recordingArchiveServer(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	zipPath := buildZip(t, t.TempDir(), "stdx.zip", map[string]string{
		"top/dynamic/libfoo.so": "dynamic",
		"top/static/libfoo.a":   "static",
		"index.html":            "docs",
	})
	var mu sync.Mutex
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		http.ServeFile(w, r, zipPath)
	}))
	t.Cleanup(server.Close)
	return server, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), paths...)
	}
}

func TestInstallFromSource_StdxResolvesManifestLinkByPlatform(t *testing.T) {
	server, requested := recordingArchiveServer(t)
	const version = "1.1.0-beta.25"
	// The tag is the bare SDK version, the asset keeps the .1 revision: the
	// exact inconsistency the manifest exists to capture verbatim. The link
	// is chosen by the stdx platform token the SDK tuple maps to.
	source := sourceWithComponents(t, toolchain.STS, version, dist.ComponentSet{
		Stdx: map[string]dist.ComponentInfo{
			"linux-aarch64": {Name: "cangjie-stdx-linux-aarch64-1.1.0-beta.25.1.zip", URL: server.URL + "/stdx/linux-aarch64.zip"},
			"windows-x64":   {Name: "cangjie-stdx-windows-x64-1.1.0-beta.25.1.zip", URL: server.URL + "/stdx/windows-x64.zip"},
			"ohos-aarch64":  {URL: server.URL + "/stdx/ohos-aarch64.zip"},
			"ohos-x64":      {URL: server.URL + "/stdx/ohos-x64.zip"},
			"android-arm32": {URL: server.URL + "/stdx/android-arm32.zip"},
		},
	})
	tc := toolchain.ToolchainName{Channel: toolchain.STS, Version: version}

	tests := map[string]string{
		"linux-arm64":             "/stdx/linux-aarch64.zip",
		"win32-x64":               "/stdx/windows-x64.zip",
		"linux-x64-ohos":          "/stdx/ohos-aarch64.zip",
		"linux-x64-ohos-x64":      "/stdx/ohos-x64.zip",
		"linux-x64-android-arm32": "/stdx/android-arm32.zip",
	}
	for tuple, wantPath := range tests {
		t.Run(tuple, func(t *testing.T) {
			roots := Roots{TcDir: t.TempDir(), DocsDir: t.TempDir(), StdxDir: t.TempDir()}
			require.NoError(t, InstallFromSource(context.Background(), roots, tc, Stdx, tuple, t.TempDir(), false, source, nil))
			assert.Contains(t, requested(), wantPath)
		})
	}
}

func TestInstallFromSource_StdxMissingPlatformReportsNotPublished(t *testing.T) {
	source := sourceWithComponents(t, toolchain.STS, "1.1.0", dist.ComponentSet{
		Stdx: map[string]dist.ComponentInfo{"linux-x64": {URL: "https://example.invalid/stdx/linux-x64"}},
	})
	roots := Roots{TcDir: t.TempDir(), DocsDir: t.TempDir(), StdxDir: t.TempDir()}
	tc := toolchain.ToolchainName{Channel: toolchain.STS, Version: "1.1.0"}
	// win32-x64 maps to "windows-x64", which this version does not publish.
	err := InstallFromSource(context.Background(), roots, tc, Stdx, "win32-x64", t.TempDir(), false, source, nil)
	var notPub *cjverr.ComponentNotPublishedError
	require.ErrorAs(t, err, &notPub)
	assert.Equal(t, "windows-x64", notPub.Target)
}

func TestInstallFromSource_StdxRequiresTuple(t *testing.T) {
	source := sourceWithComponents(t, toolchain.LTS, "1.0.5", dist.ComponentSet{
		Stdx: map[string]dist.ComponentInfo{"linux-x64": {URL: "https://example.invalid/stdx"}},
	})
	roots := Roots{TcDir: t.TempDir(), DocsDir: t.TempDir(), StdxDir: t.TempDir()}
	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}
	err := InstallFromSource(context.Background(), roots, tc, Stdx, "", t.TempDir(), false, source, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stdx requires a host tuple")
}

func TestInstallFromSource_VersionNotInManifest(t *testing.T) {
	source := sourceWithComponents(t, toolchain.STS, "1.1.0", dist.ComponentSet{
		Docs: &dist.ComponentInfo{URL: "https://example.invalid/docs"},
	})
	roots := Roots{TcDir: t.TempDir(), DocsDir: t.TempDir(), StdxDir: t.TempDir()}
	tc := toolchain.ToolchainName{Channel: toolchain.STS, Version: "9.9.9"}
	err := InstallFromSource(context.Background(), roots, tc, Docs, "", t.TempDir(), false, source, nil)
	var notPub *cjverr.ComponentNotPublishedError
	require.ErrorAs(t, err, &notPub)
}

func TestInstallFromSource_DocsAndStdxDocsResolveManifestLinks(t *testing.T) {
	server, requested := recordingArchiveServer(t)
	source := sourceWithComponents(t, toolchain.LTS, "1.0.5", dist.ComponentSet{
		Docs:     &dist.ComponentInfo{URL: server.URL + "/docs.zip"},
		StdxDocs: &dist.ComponentInfo{URL: server.URL + "/stdx-docs.zip"},
	})
	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}
	for _, name := range []Name{Docs, StdxDocs} {
		roots := Roots{TcDir: t.TempDir(), DocsDir: t.TempDir(), StdxDir: t.TempDir()}
		require.NoError(t, InstallFromSource(context.Background(), roots, tc, name, "", t.TempDir(), false, source, nil))
	}
	assert.ElementsMatch(t, []string{"/docs.zip", "/stdx-docs.zip"}, requested())
}

func TestInstallFromSource_NightlyReadsNightlyManifest(t *testing.T) {
	server, requested := recordingArchiveServer(t)
	const version = "1.2.0-alpha.20260613020028"
	source := sourceWithComponents(t, toolchain.Nightly, version, dist.ComponentSet{
		Docs: &dist.ComponentInfo{URL: server.URL + "/nightly/docs.zip"},
		Stdx: map[string]dist.ComponentInfo{"linux-x64": {URL: server.URL + "/nightly/stdx.zip"}},
	})
	tc := toolchain.ToolchainName{Channel: toolchain.Nightly, Version: version}

	roots := Roots{TcDir: t.TempDir(), DocsDir: t.TempDir(), StdxDir: t.TempDir()}
	require.NoError(t, InstallFromSource(context.Background(), roots, tc, Docs, "", t.TempDir(), false, source, nil))
	roots = Roots{TcDir: t.TempDir(), DocsDir: t.TempDir(), StdxDir: t.TempDir()}
	require.NoError(t, InstallFromSource(context.Background(), roots, tc, Stdx, "linux-x64", t.TempDir(), false, source, nil))
	assert.ElementsMatch(t, []string{"/nightly/docs.zip", "/nightly/stdx.zip"}, requested())
}

func TestInstallFromSource_ReportsStages(t *testing.T) {
	server, _ := recordingArchiveServer(t)
	source := sourceWithComponents(t, toolchain.LTS, "1.0.5", dist.ComponentSet{
		Docs: &dist.ComponentInfo{URL: server.URL + "/docs.zip"},
	})
	roots := Roots{TcDir: t.TempDir(), DocsDir: t.TempDir(), StdxDir: t.TempDir()}
	tc := toolchain.ToolchainName{Channel: toolchain.LTS, Version: "1.0.5"}
	var stages []string
	require.NoError(t, InstallFromSource(context.Background(), roots, tc, Docs, "", t.TempDir(), false, source, func(stage string) {
		stages = append(stages, stage)
	}))
	assert.Equal(t, "FetchingComponent,InstallingComponent", strings.Join(stages, ","))
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
