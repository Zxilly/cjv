//go:build smoke

package smoke

import (
	"archive/zip"
	"bytes"
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
	"time"

	"github.com/Zxilly/cjv/internal/dist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	flowInstallCommand = "install-command"
	flowRunInstall     = "run-install"
)

var (
	smokeBuildDir   string
	smokeCJVBinary  string
	smokeStubBinary string
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
	smokeBuildDir, err = os.MkdirTemp("", "cjv-smoke-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create smoke build dir: %v\n", err)
		os.Exit(1)
	}

	if err := buildSmokeBinaries(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.RemoveAll(smokeBuildDir) //nolint:errcheck
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(smokeBuildDir) //nolint:errcheck
	os.Exit(code)
}

func buildSmokeBinaries() error {
	smokeCJVBinary = filepath.Join(smokeBuildDir, executableName("cjv"))
	cmd := exec.Command("go", "build", "-o", smokeCJVBinary, "./cmd/cjv")
	cmd.Dir = filepath.Join("..", "..")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to build cjv: %s", string(out))
	}

	stubSource := filepath.Join(smokeBuildDir, "stub.go")
	if err := os.WriteFile(stubSource, []byte(stubSourceCode), 0o644); err != nil {
		return fmt.Errorf("failed to write stub source: %w", err)
	}
	smokeStubBinary = filepath.Join(smokeBuildDir, executableName("stub"))
	cmd = exec.Command("go", "build", "-o", smokeStubBinary, stubSource)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to build tool stub: %s", string(out))
	}

	return nil
}

func TestSmokeUserInstallFlows(t *testing.T) {
	selectedFlow := os.Getenv("CJV_SMOKE_FLOW")
	flows := []string{flowInstallCommand, flowRunInstall}
	if selectedFlow != "" {
		require.Contains(t, flows, selectedFlow, "unknown CJV_SMOKE_FLOW")
		flows = []string{selectedFlow}
	}

	for _, flow := range flows {
		t.Run(flow, func(t *testing.T) {
			server := newMockSDKServer(t)
			cjvHome := setupSmokeHome(t, server.URL)

			switch flow {
			case flowInstallCommand:
				runCJV(t, smokeCJVBinary, cjvHome, "install", "lts")
			case flowRunInstall:
				stdout := runCJV(t, smokeCJVBinary, cjvHome, "run", "--install", "lts", "cjc", "smoke")
				assert.Contains(t, stdout, "cjc stub")
			default:
				t.Fatalf("unhandled smoke flow %q", flow)
			}

			managedBinary := filepath.Join(cjvHome, "bin", executableName("cjv"))
			assert.FileExists(t, managedBinary)
			assert.FileExists(t, filepath.Join(cjvHome, "bin", executableName("cjc")))
			assert.FileExists(t, filepath.Join(cjvHome, "bin", executableName("cjpm")))

			assert.Contains(t, runCJV(t, managedBinary, cjvHome, "toolchain", "list"), "lts-1.0.5")
			assert.Contains(t, runCJV(t, managedBinary, cjvHome, "show", "active"), "lts-1.0.5")
			assert.Contains(t, runCJV(t, managedBinary, cjvHome, "which", "cjc"), "cjc")
			assert.Contains(t, runCJV(t, managedBinary, cjvHome, "run", "lts", "cjc", "smoke"), "cjc stub")
			assert.Contains(t, runCJV(t, managedBinary, cjvHome, "run", "lts", "cjpm", "smoke"), "cjpm stub")

			proxyStdout := runCommand(t, filepath.Join(cjvHome, "bin", executableName("cjc")), smokeEnv(cjvHome), "smoke")
			assert.Contains(t, proxyStdout, "cjc stub")

			runCJV(t, managedBinary, cjvHome, "uninstall", "lts-1.0.5")
			assert.Contains(t, runCJV(t, managedBinary, cjvHome, "toolchain", "list"), "No toolchains installed")
			assert.NoDirExists(t, filepath.Join(cjvHome, "toolchains", "lts-1.0.5"))
		})
	}
}

func setupSmokeHome(t *testing.T, serverURL string) string {
	t.Helper()

	cjvHome := t.TempDir()
	settingsContent := fmt.Sprintf("manifest_url = %q\nauto_install = true\n", serverURL+"/sdk-versions.json")
	require.NoError(t, os.WriteFile(filepath.Join(cjvHome, "settings.toml"), []byte(settingsContent), 0o644))
	return cjvHome
}

func newMockSDKServer(t *testing.T) *httptest.Server {
	t.Helper()

	sdkArchive, sdkHash := createExecutableMockSDKZip(t)
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	platformKey, err := dist.CurrentPlatformKey("")
	require.NoError(t, err)

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
		if err := json.NewEncoder(w).Encode(manifest); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(sdkArchive)
	})

	t.Cleanup(server.Close)
	return server
}

func createExecutableMockSDKZip(t *testing.T) ([]byte, string) {
	t.Helper()

	stubData, err := os.ReadFile(smokeStubBinary)
	require.NoError(t, err)

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	writeEntry := func(name string, data []byte, mode os.FileMode) {
		hdr := &zip.FileHeader{Name: name, Method: zip.Deflate}
		hdr.SetMode(mode)
		f, err := w.CreateHeader(hdr)
		require.NoError(t, err, "create zip entry %s", name)
		_, err = f.Write(data)
		require.NoError(t, err, "write zip entry %s", name)
	}

	writeEntry("cangjie/bin/"+executableName("cjc"), stubData, 0o755)
	writeEntry("cangjie/tools/bin/"+executableName("cjpm"), stubData, 0o755)
	writeEntry("cangjie/envsetup.sh", []byte("export CANGJIE_HOME=\"$PWD\"\n"), 0o644)
	writeEntry("cangjie/envsetup.ps1", []byte("$env:CANGJIE_HOME = $PWD.Path\n"), 0o644)
	require.NoError(t, w.Close())

	data := buf.Bytes()
	sum := sha256.Sum256(data)
	return data, hex.EncodeToString(sum[:])
}

func runCJV(t *testing.T, binary, cjvHome string, args ...string) string {
	t.Helper()
	return runCommand(t, binary, smokeEnv(cjvHome), args...)
}

func runCommand(t *testing.T, binary string, env []string, args ...string) string {
	t.Helper()

	cmd := exec.Command(binary, args...)
	cmd.Env = env
	cmd.Dir = t.TempDir()

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		require.NoError(t, err, "command failed: %s %s\nstdout:\n%s\nstderr:\n%s",
			binary, strings.Join(args, " "), stdout.String(), stderr.String())
	case <-time.After(60 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatalf("command timed out: %s %s\nstdout:\n%s\nstderr:\n%s",
			binary, strings.Join(args, " "), stdout.String(), stderr.String())
	}

	return stdout.String() + stderr.String()
}

func smokeEnv(cjvHome string) []string {
	fakeHome := filepath.Join(cjvHome, "user-home")
	return append(os.Environ(),
		"CJV_HOME="+cjvHome,
		"CJV_LANG=en",
		"CJV_TOOLCHAIN=",
		"CJV_NO_PATH_SETUP=1",
		"HOME="+fakeHome,
		"USERPROFILE="+fakeHome,
	)
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}
