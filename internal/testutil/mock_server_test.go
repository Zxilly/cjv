package testutil

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for MockDistServer and CreateMockSDKZip — verifies the test
// infrastructure itself works correctly.

func TestMockDistServer_ServesManifest(t *testing.T) {
	server := MockDistServer(t)

	resp, err := http.Get(server.URL + "/sdk-versions.json")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "channels")
	assert.Contains(t, string(body), "lts")
}

func TestMockDistServer_ServesDownloads(t *testing.T) {
	server := MockDistServer(t)

	resp, err := http.Get(server.URL + "/download/test-sdk.zip")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.NotEmpty(t, body, "download should return non-empty content")
}

func TestCreateMockSDKZip_ValidArchive(t *testing.T) {
	data, hash := CreateMockSDKZip("1.0.5")

	assert.NotEmpty(t, data, "zip data should not be empty")
	assert.Len(t, hash, 64, "SHA256 hash should be 64 hex characters")

	// Verify it's a valid zip archive
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)

	// Should contain cangjie directory structure
	fileNames := make([]string, len(reader.File))
	for i, f := range reader.File {
		fileNames[i] = f.Name
	}
	assert.NotEmpty(t, fileNames)
}

func TestCreateMockSDKZip_DifferentVersions(t *testing.T) {
	data1, hash1 := CreateMockSDKZip("1.0.0")
	data2, hash2 := CreateMockSDKZip("2.0.0")

	// Different versions should produce different zip content (different scripts)
	assert.NotEqual(t, hash1, hash2,
		"different versions should produce different hashes")
	assert.NotEqual(t, data1, data2)
}
