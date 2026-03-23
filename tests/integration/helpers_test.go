//go:build integration

package integration

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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/stretchr/testify/require"
)

// Package-level cached build artifacts, populated once by TestMain.
var (
	cachedCJVBinary  string // path to compiled cjv binary
	cachedStubBinary string // path to compiled stub executable
	cachedCJVArchive []byte // platform-appropriate archive (tar.gz or zip)
	buildTmpDir      string // temp directory holding cached binaries
)

const stubSourceCode = `package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	name := filepath.Base(os.Args[0])
	name = strings.TrimSuffix(name, ".exe")
	fmt.Printf("%s stub\n", name)
}
`

func TestMain(m *testing.M) {
	var err error
	buildTmpDir, err = os.MkdirTemp("", "cjv-integration-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create build temp dir: %v\n", err)
		os.Exit(1)
	}

	if err := buildCachedBinaries(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.RemoveAll(buildTmpDir)
		os.Exit(1)
	}

	code := runWithRegistryGuard(m)
	os.RemoveAll(buildTmpDir)
	os.Exit(code)
}

func buildCachedBinaries() error {
	// Build cjv binary once
	cachedCJVBinary = filepath.Join(buildTmpDir, "cjv")
	if runtime.GOOS == "windows" {
		cachedCJVBinary += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", cachedCJVBinary, "./cmd/cjv")
	cmd.Dir = filepath.Join("..", "..")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to build cjv: %s", string(out))
	}

	// Build stub executable once
	stubSrc := filepath.Join(buildTmpDir, "stub.go")
	if err := os.WriteFile(stubSrc, []byte(stubSourceCode), 0o644); err != nil {
		return fmt.Errorf("failed to write stub source: %v", err)
	}
	cachedStubBinary = filepath.Join(buildTmpDir, "stub")
	if runtime.GOOS == "windows" {
		cachedStubBinary += ".exe"
	}
	cmd = exec.Command("go", "build", "-o", cachedStubBinary, stubSrc)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to build stub: %s", string(out))
	}

	// Create platform-appropriate CJV archive once
	binaryData, err := os.ReadFile(cachedCJVBinary)
	if err != nil {
		return fmt.Errorf("failed to read cjv binary: %v", err)
	}
	if runtime.GOOS == "windows" {
		cachedCJVArchive = zipFromData("cjv.exe", binaryData)
	} else {
		cachedCJVArchive = tarGzFromData("cjv", binaryData)
	}

	return nil
}

func tarGzFromData(name string, data []byte) []byte {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	_ = tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(data))})
	_, _ = tw.Write(data)
	_ = tw.Close()
	_ = gw.Close()
	return buf.Bytes()
}

func zipFromData(name string, data []byte) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create(name)
	_, _ = f.Write(data)
	_ = w.Close()
	return buf.Bytes()
}

// buildCJV returns the pre-built cjv binary path.
func buildCJV(t *testing.T) string {
	t.Helper()
	if cachedCJVBinary == "" {
		t.Fatal("cachedCJVBinary not set; TestMain did not run")
	}
	return cachedCJVBinary
}

// buildStubExecutable returns the pre-built stub executable path.
func buildStubExecutable(t *testing.T) string {
	t.Helper()
	if cachedStubBinary == "" {
		t.Fatal("cachedStubBinary not set; TestMain did not run")
	}
	return cachedStubBinary
}

// requireCI skips the test when not running in CI. Used by tests that modify
// real system state (e.g., Windows registry PATH) to protect dev environments.
func requireCI(t *testing.T) {
	t.Helper()
	if os.Getenv("CI") != "true" {
		t.Skip("PATH auto-configuration test skipped outside CI to protect developer environment")
	}
}

// runCJVEnv runs the cjv binary with the given args and custom env additions.
// This is the general-purpose runner; use runCJV for the common case with
// CJV_NO_PATH_SETUP=1 automatically set.
func runCJVEnv(t *testing.T, binary, cjvHome string, extraEnv []string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Env = append(os.Environ(),
		"CJV_HOME="+cjvHome,
		"CJV_LANG=en",
		"CJV_TOOLCHAIN=",
	)
	cmd.Env = append(cmd.Env, extraEnv...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// installScriptEnv builds the env slice for install script exec.Commands.
func installScriptEnv(serverURL, cjvHome string, extra ...string) []string {
	env := append(os.Environ(),
		"CJV_UPDATE_ROOT="+serverURL,
		"CJV_HOME="+cjvHome,
		"CJV_LANG=en",
		"CJV_NO_PATH_SETUP=1",
	)
	return append(env, extra...)
}

// createExecutableMockSDKZip creates an SDK zip with real executable stubs
// for cjc and cjpm, so proxy execution tests can verify the full chain.
func createExecutableMockSDKZip(t *testing.T, stubBinaryPath string) ([]byte, string) {
	t.Helper()
	stubData, err := os.ReadFile(stubBinaryPath)
	require.NoError(t, err, "failed to read stub binary")

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	writeEntry := func(name string, data []byte, mode os.FileMode) {
		hdr := &zip.FileHeader{
			Name:   name,
			Method: zip.Deflate,
		}
		hdr.SetMode(mode)
		f, err := w.CreateHeader(hdr)
		require.NoError(t, err, "zip create %s", name)
		_, err = f.Write(data)
		require.NoError(t, err, "zip write %s", name)
	}

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	writeEntry("cangjie/bin/cjc"+ext, stubData, 0o755)
	writeEntry("cangjie/tools/bin/cjpm"+ext, stubData, 0o755)
	writeEntry("cangjie/envsetup.sh", []byte("export CANGJIE_HOME=\"$PWD\"\n"), 0o644)
	writeEntry("cangjie/envsetup.ps1", []byte("$env:CANGJIE_HOME = $PWD.Path\n"), 0o644)

	require.NoError(t, w.Close())

	data := buf.Bytes()
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	return data, hash
}

// newMockSDKServer creates a mock HTTP server serving:
//   - /sdk-versions.json — SDK manifest
//   - /download/*.zip — SDK archive
//   - /cjv_* — CJV binary archive (using cachedCJVArchive)
func newMockSDKServer(t *testing.T, sdkArchive []byte, sdkHash string) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	platformKey, err := dist.CurrentPlatformKey("")
	require.NoError(t, err, "failed to get platform key")

	var manifest dist.Manifest
	manifest.Channels.LTS = dist.ChannelInfo{
		Latest: "1.0.5",
		Versions: map[string]map[string]dist.DownloadInfo{
			"1.0.5": {
				platformKey: {
					Name:   "cangjie-sdk-1.0.5.zip",
					SHA256: sdkHash,
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
					SHA256: sdkHash,
					URL:    server.URL + "/download/cangjie-sdk-1.1.0-beta.1.zip",
				},
			},
		},
	}

	mux.HandleFunc("/sdk-versions.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(manifest)
	})

	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(sdkArchive)
	})

	serveCJVArchive := func(w http.ResponseWriter) {
		if runtime.GOOS == "windows" {
			w.Header().Set("Content-Type", "application/zip")
		} else {
			w.Header().Set("Content-Type", "application/gzip")
		}
		_, _ = w.Write(cachedCJVArchive)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/cjv_") {
			http.NotFound(w, r)
			return
		}
		serveCJVArchive(w)
	})

	t.Cleanup(server.Close)
	return server
}

// mockCJVDownloadServer creates a mock server serving only the CJV binary archive.
func mockCJVDownloadServer(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if runtime.GOOS == "windows" {
			w.Header().Set("Content-Type", "application/zip")
		} else {
			w.Header().Set("Content-Type", "application/gzip")
		}
		_, _ = w.Write(cachedCJVArchive)
	}))

	t.Cleanup(server.Close)
	return server
}

// mockCJVDownloadServerWithSDK creates a mock server for the CJV binary + SDK (text stubs).
func mockCJVDownloadServerWithSDK(t *testing.T) *httptest.Server {
	t.Helper()
	sdkArchive, sdkHash := testutil.CreateMockSDKZip("1.0.5")
	return newMockSDKServer(t, sdkArchive, sdkHash)
}

// mockCJVDownloadServerWithExecutableSDK creates a mock server for the CJV binary
// + SDK with real executable stubs, for testing proxy execution.
func mockCJVDownloadServerWithExecutableSDK(t *testing.T) *httptest.Server {
	t.Helper()
	sdkArchive, sdkHash := createExecutableMockSDKZip(t, cachedStubBinary)
	return newMockSDKServer(t, sdkArchive, sdkHash)
}
