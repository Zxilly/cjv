package selfupdate

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/proxy"
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
	managed := filepath.Join(binDir, proxy.CjvBinaryName())
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
