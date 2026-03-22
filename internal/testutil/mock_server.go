package testutil

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/dist"
)

// MockDistServer creates a mock distribution server providing
// sdk-versions.json and SDK archive downloads.
func MockDistServer(t testing.TB) *httptest.Server {
	archive, hash := CreateMockSDKZip("1.0.5")

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	platformKey, err := dist.CurrentPlatformKey("")
	if err != nil {
		t.Fatalf("failed to get platform key: %v", err)
	}

	var manifest dist.Manifest
	manifest.Channels.LTS = dist.ChannelInfo{
		Latest: "1.0.5",
		Versions: map[string]map[string]dist.DownloadInfo{
			"1.0.5": {
				platformKey: {
					Name:   "cangjie-sdk-1.0.5.zip",
					SHA256: hash,
					URL:    server.URL + "/download/cangjie-sdk-1.0.5.zip",
				},
			},
		},
	}
	manifest.Channels.STS = dist.ChannelInfo{
		Latest: "1.1.0-beta.1",
		Versions: map[string]map[string]dist.DownloadInfo{
			"1.1.0-beta.1": {
				platformKey: {
					Name:   "cangjie-sdk-1.1.0-beta.1.zip",
					SHA256: hash,
					URL:    server.URL + "/download/cangjie-sdk-1.1.0-beta.1.zip",
				},
			},
		},
	}

	mux.HandleFunc("/sdk-versions.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(manifest); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(archive)
	})

	t.Cleanup(server.Close)
	return server
}

// CreateMockSDKZip creates a minimal SDK zip archive with stub executables.
func CreateMockSDKZip(version string) ([]byte, string) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	writeFile := func(name, content string) {
		f, err := w.Create(name)
		if err != nil {
			panic(fmt.Sprintf("zip create %s: %v", name, err))
		}
		if _, err := fmt.Fprint(f, content); err != nil {
			panic(fmt.Sprintf("zip write %s: %v", name, err))
		}
	}

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	writeFile("cangjie/bin/cjc"+ext, fmt.Sprintf("#!/bin/sh\necho 'cjc %s'\n", version))
	writeFile("cangjie/tools/bin/cjpm"+ext, fmt.Sprintf("#!/bin/sh\necho 'cjpm %s'\n", version))
	writeFile("cangjie/envsetup.sh", "export CANGJIE_HOME=\"$PWD\"\n")
	writeFile("cangjie/envsetup.ps1", "$env:CANGJIE_HOME = $PWD.Path\n")

	if err := w.Close(); err != nil {
		panic(fmt.Sprintf("zip close: %v", err))
	}

	data := buf.Bytes()
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	return data, hash
}

