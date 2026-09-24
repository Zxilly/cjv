package selfupdate

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChecksumForAsset(t *testing.T) {
	digest := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	data := []byte(fmt.Sprintf("%s  other.zip\n%s *cjv.zip\n", digest, digest))

	got, err := checksumForAsset(data, "cjv.zip")
	require.NoError(t, err)
	assert.Equal(t, digest, got)

	_, err = checksumForAsset(data, "missing.zip")
	require.Error(t, err)
}

func TestNewerReleaseVersion(t *testing.T) {
	latest, newer, err := newerReleaseVersion("1.2.3", "v1.3.0")
	require.NoError(t, err)
	assert.Equal(t, "1.3.0", latest)
	assert.True(t, newer)

	_, newer, err = newerReleaseVersion("1.3.0", "v1.3.0")
	require.NoError(t, err)
	assert.False(t, newer)

	_, _, err = newerReleaseVersion("1.2.3", "not-a-version")
	require.Error(t, err)
}

func TestInstallReleaseArtifact(t *testing.T) {
	const (
		assetName  = "cjv-test.zip"
		binaryName = "cjv-new"
	)
	archive := zipExecutable(t, binaryName, []byte("new executable"))
	digest := sha256.Sum256(archive)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + assetName:
			_, _ = w.Write(archive)
		case "/checksums.txt":
			_, _ = fmt.Fprintf(w, "%x  %s\n", digest, assetName)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	home := t.TempDir()
	t.Setenv(config.EnvHome, home)
	binDir := filepath.Join(home, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	managed := filepath.Join(binDir, sdktools.CjvBinaryName())
	require.NoError(t, os.WriteFile(managed, []byte("old executable"), 0o755))

	err := installReleaseArtifact(context.Background(), releaseArtifact{
		AssetName:   assetName,
		BinaryName:  binaryName,
		AssetURL:    srv.URL + "/" + assetName,
		ChecksumURL: srv.URL + "/checksums.txt",
	})
	require.NoError(t, err)

	got, err := os.ReadFile(managed)
	require.NoError(t, err)
	assert.Equal(t, []byte("new executable"), got)
}

func TestInstallReleaseArtifactReplacesRunningExecutable(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)
	binDir := filepath.Join(home, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	managed := filepath.Join(binDir, sdktools.CjvBinaryName())
	buildRunningExecutable(t, managed, "old")

	newBinaryName := platformBinaryName("cjv-new", runtime.GOOS)
	newBinary := filepath.Join(t.TempDir(), newBinaryName)
	buildRunningExecutable(t, newBinary, "new")
	newData, err := os.ReadFile(newBinary)
	require.NoError(t, err)
	assetName := releaseAssetName("cjv-test", runtime.GOOS, runtime.GOARCH)
	archive := zipExecutable(t, newBinaryName, newData)
	digest := sha256.Sum256(archive)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + assetName:
			_, _ = w.Write(archive)
		case "/checksums.txt":
			_, _ = fmt.Fprintf(w, "%x  %s\n", digest, assetName)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	running := exec.Command(managed, "hold")
	stdin, err := running.StdinPipe()
	require.NoError(t, err)
	stdout, err := running.StdoutPipe()
	require.NoError(t, err)
	running.Stderr = os.Stderr
	require.NoError(t, running.Start())
	stopped := false
	defer func() {
		if !stopped {
			_ = running.Process.Kill()
			_ = running.Wait()
		}
	}()

	scanner := bufio.NewScanner(stdout)
	require.True(t, scanner.Scan())
	assert.Equal(t, "old", scanner.Text())

	require.NoError(t, installReleaseArtifact(context.Background(), releaseArtifact{
		AssetName:   assetName,
		BinaryName:  newBinaryName,
		AssetURL:    srv.URL + "/" + assetName,
		ChecksumURL: srv.URL + "/checksums.txt",
	}))

	output, err := exec.Command(managed).CombinedOutput()
	require.NoError(t, err, string(output))
	assert.Equal(t, "new", strings.TrimSpace(string(output)))

	oldPath := filepath.Join(binDir, "."+sdktools.CjvBinaryName()+".old")
	if runtime.GOOS == "windows" {
		assert.FileExists(t, oldPath)
		assert.True(t, fileIsHidden(t, oldPath))
	} else {
		assert.NoFileExists(t, oldPath)
	}

	require.NoError(t, stdin.Close())
	require.NoError(t, running.Wait())
	stopped = true
	CleanupOldBinaries()
	assert.NoFileExists(t, oldPath)
}

func TestExecutableUpdateRestoresCurrentFileWhenPromotionFails(t *testing.T) {
	target := filepath.Join(t.TempDir(), platformBinaryName("cjv", runtime.GOOS))
	require.NoError(t, os.WriteFile(target, []byte("old executable"), 0o755))

	realRename := renameUpdateFile
	promotionErr := errors.New("injected promotion failure")
	renameCalls := 0
	renameUpdateFile = func(oldPath, newPath string) error {
		renameCalls++
		if renameCalls == 2 {
			return promotionErr
		}
		return realRename(oldPath, newPath)
	}
	t.Cleanup(func() { renameUpdateFile = realRename })

	err := applyExecutableUpdate(bytes.NewReader([]byte("new executable")), target, 0o755)
	require.ErrorIs(t, err, promotionErr)

	got, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, []byte("old executable"), got)
	assert.FileExists(t, filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".new"))
	assert.NoFileExists(t, filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".old"))
}

func TestExecutableUpdateReportsPromotionAndRollbackFailures(t *testing.T) {
	target := filepath.Join(t.TempDir(), platformBinaryName("cjv", runtime.GOOS))
	require.NoError(t, os.WriteFile(target, []byte("old executable"), 0o755))

	realRename := renameUpdateFile
	promotionErr := errors.New("injected promotion failure")
	rollbackErr := errors.New("injected rollback failure")
	renameCalls := 0
	renameUpdateFile = func(oldPath, newPath string) error {
		renameCalls++
		switch renameCalls {
		case 2:
			return promotionErr
		case 3:
			return rollbackErr
		default:
			return realRename(oldPath, newPath)
		}
	}
	t.Cleanup(func() { renameUpdateFile = realRename })

	err := applyExecutableUpdate(bytes.NewReader([]byte("new executable")), target, 0o755)
	require.ErrorIs(t, err, promotionErr)
	require.ErrorIs(t, err, rollbackErr)

	assert.NoFileExists(t, target)
	assert.FileExists(t, filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".new"))
	assert.FileExists(t, filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".old"))
}

func buildRunningExecutable(t *testing.T, output, marker string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-ldflags", "-X main.marker="+marker, "-o", output, "./testdata/running_executable")
	cmd.Dir = "."
	data, err := cmd.CombinedOutput()
	require.NoError(t, err, string(data))
}

func zipExecutable(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetMode(0o755)
	w, err := zw.CreateHeader(header)
	require.NoError(t, err)
	_, err = w.Write(data)
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return buf.Bytes()
}
