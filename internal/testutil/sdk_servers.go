package testutil

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/sdktools"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// MockSDKZip builds an SDK zip carrying a stub for every proxied tool at its
// platform-correct relative path, plus the envsetup scripts.
func MockSDKZip() ([]byte, string) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	writeEntry := func(name, content string) {
		f, err := w.Create(name)
		if err != nil {
			panic(fmt.Sprintf("zip create %s: %v", name, err))
		}
		if _, err := f.Write([]byte(content)); err != nil {
			panic(fmt.Sprintf("zip write %s: %v", name, err))
		}
	}

	for _, tool := range sdktools.AllProxyTools() {
		relPath := sdktools.ToolRelativePath(tool)
		name := "cangjie/" + relPath
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		writeEntry(name, "stub-"+tool)
	}
	writeEntry("cangjie/envsetup.sh", "export CANGJIE_HOME=\"$PWD\"")
	writeEntry("cangjie/envsetup.ps1", "$env:CANGJIE_HOME = $PWD.Path")

	if err := w.Close(); err != nil {
		panic(fmt.Sprintf("zip close: %v", err))
	}
	hash := sha256.Sum256(buf.Bytes())
	return buf.Bytes(), hex.EncodeToString(hash[:])
}

// MockTarGz builds a tar.gz from name->content entries.
func MockTarGz(t testing.TB, files map[string]string) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}); err != nil {
			t.Fatalf("tar header %s: %v", name, err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("tar write %s: %v", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return buf.Bytes(), hex.EncodeToString(sum[:])
}

func serveManifestAndSDK(t testing.TB, manifest func(base string) dist.Manifest, sdkData []byte) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	m := manifest(server.URL)
	mux.HandleFunc("/download/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(sdkData)
	})
	mux.HandleFunc("/sdk-versions.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(m); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	return server
}

// ValidMockServer serves a manifest at /sdk-versions.json with lts 1.0.5 and
// sts 2.0.0 for the host, both backed by MockSDKZip under /download/.
func ValidMockServer(t testing.TB) *httptest.Server {
	t.Helper()
	sdkData, sha := MockSDKZip()
	return MockServerWithSDK(t, sdkData, sha)
}

// MockServerWithSDK serves a manifest at /sdk-versions.json with lts 1.0.5
// and sts 2.0.0 for the host, both backed by the given archive.
func MockServerWithSDK(t testing.TB, sdkData []byte, sha string) *httptest.Server {
	t.Helper()
	pk, err := sdktarget.CurrentHostTuple("")
	if err != nil {
		t.Fatalf("failed to get host tuple: %v", err)
	}
	return serveManifestAndSDK(t, func(base string) dist.Manifest {
		var manifest dist.Manifest
		manifest.Channels.LTS = dist.ChannelInfo{
			Latest: "1.0.5",
			Versions: map[string]map[string]dist.DownloadInfo{
				"1.0.5": {pk: {Name: "cangjie-sdk-1.0.5.zip", SHA256: sha, URL: base + "/download/cangjie-sdk-1.0.5.zip"}},
			},
		}
		manifest.Channels.STS = dist.ChannelInfo{
			Latest: "2.0.0",
			Versions: map[string]map[string]dist.DownloadInfo{
				"2.0.0": {pk: {Name: "cangjie-sdk-2.0.0.zip", SHA256: sha, URL: base + "/download/cangjie-sdk-2.0.0.zip"}},
			},
		}
		return manifest
	}, sdkData)
}

// MockServerWithTargetSDKs publishes channel at version for the host and
// each target environment, so a host toolchain and its target variants can
// be installed together. The other channel gets one host-only release.
func MockServerWithTargetSDKs(t testing.TB, channel toolchain.Channel, version string, targets ...string) *httptest.Server {
	t.Helper()
	sdkData, sha := MockSDKZip()
	hostKey, err := sdktarget.CurrentHostTuple("")
	if err != nil {
		t.Fatalf("failed to get host tuple: %v", err)
	}
	keys := map[string]string{hostKey: "cangjie-sdk-" + version + ".zip"}
	for _, target := range targets {
		key, err := sdktarget.CurrentTargetTuple("", target)
		if err != nil {
			t.Fatalf("failed to get target tuple: %v", err)
		}
		keys[key] = "cangjie-sdk-" + key + "-" + version + ".zip"
	}
	return serveManifestAndSDK(t, func(base string) dist.Manifest {
		platforms := make(map[string]dist.DownloadInfo, len(keys))
		for key, name := range keys {
			platforms[key] = dist.DownloadInfo{Name: name, SHA256: sha, URL: base + "/download/" + name}
		}
		hostOnly := func(v string) dist.ChannelInfo {
			name := "cangjie-sdk-" + v + ".zip"
			return dist.ChannelInfo{
				Latest:   v,
				Versions: map[string]map[string]dist.DownloadInfo{v: {hostKey: {Name: name, SHA256: sha, URL: base + "/download/" + name}}},
			}
		}
		var manifest dist.Manifest
		if channel == toolchain.LTS {
			manifest.Channels.LTS = dist.ChannelInfo{Latest: version, Versions: map[string]map[string]dist.DownloadInfo{version: platforms}}
			manifest.Channels.STS = hostOnly("2.0.0")
		} else {
			manifest.Channels.LTS = hostOnly("1.0.5")
			manifest.Channels.STS = dist.ChannelInfo{Latest: version, Versions: map[string]map[string]dist.DownloadInfo{version: platforms}}
		}
		return manifest
	}, sdkData)
}

// SplitNightlyMockServer serves a distribution root at /corp/cjv with
// versions.json (lts 1.0.5, sts 1.1.0) and a separate nightly.json whose one
// release also publishes a docs component. Artifact links are relative to
// the root, as a dist_server publishes them.
func SplitNightlyMockServer(t testing.TB) *httptest.Server {
	t.Helper()
	sdkData, sha := MockSDKZip()
	docsData, docsSHA := MockTarGz(t, map[string]string{"index.html": "nightly docs"})
	tuple, err := sdktarget.CurrentHostTuple("")
	if err != nil {
		t.Fatalf("failed to get host tuple: %v", err)
	}

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	const version = "1.2.0-alpha.20260822010101"
	channel := dist.ChannelInfo{
		Latest: version,
		Versions: map[string]map[string]dist.DownloadInfo{
			version: {tuple: {Name: "nightly.zip", SHA256: sha, URL: "nightly/nightly.zip"}},
		},
		Components: map[string]dist.ComponentSet{
			version: {Docs: &dist.ComponentInfo{Name: "docs.tar.gz", URL: "nightly/docs.tar.gz", SHA256: docsSHA}},
		},
	}
	manifest := dist.Manifest{}
	manifest.Channels.LTS = dist.ChannelInfo{
		Latest:   "1.0.5",
		Versions: map[string]map[string]dist.DownloadInfo{"1.0.5": {tuple: {Name: "lts.zip", SHA256: sha, URL: "sdk/lts.zip"}}},
	}
	manifest.Channels.STS = dist.ChannelInfo{
		Latest:   "1.1.0",
		Versions: map[string]map[string]dist.DownloadInfo{"1.1.0": {tuple: {Name: "sts.zip", SHA256: sha, URL: "sdk/sts.zip"}}},
	}
	serveJSON := func(v any) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(v); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	}
	serveBytes := func(contentType string, data []byte) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", contentType)
			_, _ = w.Write(data)
		}
	}
	mux.HandleFunc("/corp/cjv/versions.json", serveJSON(manifest))
	mux.HandleFunc("/corp/cjv/nightly.json", serveJSON(channel))
	mux.HandleFunc("/corp/cjv/nightly/nightly.zip", serveBytes("application/zip", sdkData))
	mux.HandleFunc("/corp/cjv/nightly/docs.tar.gz", serveBytes("application/gzip", docsData))
	return server
}

// ManifestOnlyServer serves manifest at /sdk-versions.json and nothing else.
func ManifestOnlyServer(t testing.TB, manifest dist.Manifest) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/sdk-versions.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(manifest); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	return server
}

// ManifestWithPlatformGap describes an lts channel whose latest (2.0.0) is
// published for win32-x64 only while 1.5.0 covers linux-x64 and its ohos
// variant, so "latest available for a tuple" differs from "latest".
func ManifestWithPlatformGap() dist.Manifest {
	const sha = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	var manifest dist.Manifest
	manifest.Channels.LTS = dist.ChannelInfo{
		Latest: "2.0.0",
		Versions: map[string]map[string]dist.DownloadInfo{
			"2.0.0": {
				"win32-x64": {Name: "cangjie-sdk-win32-x64-2.0.0.zip", SHA256: sha, URL: "https://example.invalid/cangjie-sdk-win32-x64-2.0.0.zip"},
			},
			"1.5.0": {
				"linux-x64":      {Name: "cangjie-sdk-linux-x64-1.5.0.tar.gz", SHA256: sha, URL: "https://example.invalid/cangjie-sdk-linux-x64-1.5.0.tar.gz"},
				"linux-x64-ohos": {Name: "cangjie-sdk-linux-x64-ohos-1.5.0.tar.gz", SHA256: sha, URL: "https://example.invalid/cangjie-sdk-linux-x64-ohos-1.5.0.tar.gz"},
			},
		},
	}
	manifest.Channels.STS = dist.ChannelInfo{
		Latest: "3.0.0",
		Versions: map[string]map[string]dist.DownloadInfo{
			"3.0.0": {
				"linux-x64": {Name: "cangjie-sdk-linux-x64-3.0.0.tar.gz", SHA256: sha, URL: "https://example.invalid/cangjie-sdk-linux-x64-3.0.0.tar.gz"},
			},
		},
	}
	return manifest
}
