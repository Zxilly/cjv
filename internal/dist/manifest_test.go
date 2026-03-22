package dist

import (
	"os"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseManifest(t *testing.T) {
	data, err := os.ReadFile("testdata/sdk-versions.json")
	require.NoError(t, err)

	m, err := ParseManifest(data)
	require.NoError(t, err)

	// LTS channel
	assert.NotEmpty(t, m.Channels.LTS.Latest)
	assert.NotEmpty(t, m.Channels.LTS.Versions)

	// STS channel
	assert.NotEmpty(t, m.Channels.STS.Latest)
	assert.NotEmpty(t, m.Channels.STS.Versions)

	// Check platform entries for a specific version
	v, ok := m.Channels.LTS.Versions[m.Channels.LTS.Latest]
	require.True(t, ok)
	assert.NotEmpty(t, v)
}

func TestManifestGetDownloadInfo(t *testing.T) {
	data, err := os.ReadFile("testdata/sdk-versions.json")
	require.NoError(t, err)

	m, err := ParseManifest(data)
	require.NoError(t, err)

	info, err := m.GetDownloadInfo(toolchain.LTS, m.Channels.LTS.Latest, "win32-x64")
	require.NoError(t, err)
	assert.NotEmpty(t, info.URL)
	assert.NotEmpty(t, info.SHA256)
	assert.NotEmpty(t, info.Name)
}

func TestManifestVersionNotFound(t *testing.T) {
	data, err := os.ReadFile("testdata/sdk-versions.json")
	require.NoError(t, err)

	m, err := ParseManifest(data)
	require.NoError(t, err)

	_, err = m.GetDownloadInfo(toolchain.LTS, "99.99.99", "win32-x64")
	assert.Error(t, err)
}

func TestManifestPlatformNotAvailable(t *testing.T) {
	data, err := os.ReadFile("testdata/sdk-versions.json")
	require.NoError(t, err)

	m, err := ParseManifest(data)
	require.NoError(t, err)

	_, err = m.GetDownloadInfo(toolchain.LTS, m.Channels.LTS.Latest, "unsupported-platform")
	assert.Error(t, err)
}

func TestManifestFindVersionChannel(t *testing.T) {
	data, err := os.ReadFile("testdata/sdk-versions.json")
	require.NoError(t, err)

	m, err := ParseManifest(data)
	require.NoError(t, err)

	ch, err := m.FindVersionChannel(m.Channels.LTS.Latest)
	require.NoError(t, err)
	assert.Equal(t, toolchain.LTS, ch)
}

func TestManifestFindVersionChannelSTS(t *testing.T) {
	data, err := os.ReadFile("testdata/sdk-versions.json")
	require.NoError(t, err)

	m, err := ParseManifest(data)
	require.NoError(t, err)

	ch, err := m.FindVersionChannel("1.1.0-beta.23")
	require.NoError(t, err)
	assert.Equal(t, toolchain.STS, ch)
}

func TestManifestFindVersionChannelNotFound(t *testing.T) {
	data, err := os.ReadFile("testdata/sdk-versions.json")
	require.NoError(t, err)

	m, err := ParseManifest(data)
	require.NoError(t, err)

	_, err = m.FindVersionChannel("99.99.99")
	assert.Error(t, err)
}

func TestGetLatestVersion(t *testing.T) {
	data, err := os.ReadFile("testdata/sdk-versions.json")
	require.NoError(t, err)

	m, err := ParseManifest(data)
	require.NoError(t, err)

	ltsLatest, err := m.GetLatestVersion(toolchain.LTS)
	require.NoError(t, err)
	assert.Equal(t, "1.0.5", ltsLatest)

	stsLatest, err := m.GetLatestVersion(toolchain.STS)
	require.NoError(t, err)
	assert.Equal(t, "1.1.0-beta.23", stsLatest)
}

func TestGetLatestVersionUnknownChannel(t *testing.T) {
	data, err := os.ReadFile("testdata/sdk-versions.json")
	require.NoError(t, err)

	m, err := ParseManifest(data)
	require.NoError(t, err)

	_, err = m.GetLatestVersion(toolchain.Nightly)
	assert.Error(t, err)
}

func TestParseManifestInvalidJSON(t *testing.T) {
	_, err := ParseManifest([]byte("not json"))
	assert.Error(t, err)
}

// --- Tests merged from manifest_edge_test.go ---

func TestValidateChannel_LatestNotInVersions(t *testing.T) {
	validHash := strings.Repeat("ab", 32)
	json := `{
  "channels": {
    "lts": {
      "latest": "9.9.9",
      "versions": {
        "1.0.0": {
          "win32-x64": {
            "name": "sdk.zip",
            "sha256": "` + validHash + `",
            "url": "https://example.com/sdk.zip"
          }
        }
      }
    },
    "sts": {
      "latest": "2.0.0",
      "versions": {
        "2.0.0": {
          "win32-x64": {
            "name": "sdk.zip",
            "sha256": "` + validHash + `",
            "url": "https://example.com/sdk.zip"
          }
        }
      }
    }
  }
}`
	_, err := ParseManifest([]byte(json))
	assert.Error(t, err, "latest version not in versions map should fail")
}

func TestValidateChannel_EmptyPlatforms(t *testing.T) {
	validHash := strings.Repeat("ab", 32)
	json := `{
  "channels": {
    "lts": {
      "latest": "1.0.0",
      "versions": {
        "1.0.0": {}
      }
    },
    "sts": {
      "latest": "2.0.0",
      "versions": {
        "2.0.0": {
          "win32-x64": {
            "name": "sdk.zip",
            "sha256": "` + validHash + `",
            "url": "https://example.com/sdk.zip"
          }
        }
      }
    }
  }
}`
	_, err := ParseManifest([]byte(json))
	assert.Error(t, err, "version with zero platforms should fail")
}

func TestGetLatestVersion_UnknownChannel(t *testing.T) {
	validHash := strings.Repeat("ab", 32)
	json := `{
  "channels": {
    "lts": {"latest":"1.0.0","versions":{"1.0.0":{"win32-x64":{"name":"s.zip","sha256":"` + validHash + `","url":"http://x"}}}},
    "sts": {"latest":"2.0.0","versions":{"2.0.0":{"win32-x64":{"name":"s.zip","sha256":"` + validHash + `","url":"http://x"}}}}
  }
}`
	m, err := ParseManifest([]byte(json))
	assert.NoError(t, err)
	_, err = m.GetLatestVersion(toolchain.Nightly)
	assert.Error(t, err, "unknown channel should error")
}

// --- Tests merged from manifest_validation_test.go ---

func TestParseManifestRejectsMissingLatestVersion(t *testing.T) {
	_, err := ParseManifest([]byte(`{
		"channels": {
			"lts": {"latest": "1.0.5", "versions": {"1.0.4": {"win32-x64": {"name": "sdk.zip", "sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "url": "https://example.com/sdk.zip"}}}},
			"sts": {"latest": "1.1.0-beta.23", "versions": {"1.1.0-beta.23": {"win32-x64": {"name": "sdk.zip", "sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "url": "https://example.com/sdk.zip"}}}}
		}
	}`))
	require.Error(t, err)
	assert.ErrorContains(t, err, "latest version")
}

func TestParseManifestRejectsIncompleteDownloadInfo(t *testing.T) {
	_, err := ParseManifest([]byte(`{
		"channels": {
			"lts": {"latest": "1.0.5", "versions": {"1.0.5": {"win32-x64": {"name": "", "sha256": "bad", "url": ""}}}},
			"sts": {"latest": "1.1.0-beta.23", "versions": {"1.1.0-beta.23": {"win32-x64": {"name": "sdk.zip", "sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "url": "https://example.com/sdk.zip"}}}}
		}
	}`))
	require.Error(t, err)
}

// --- Tests merged from validate_edge_test.go ---

func TestValidateDownloadInfo_InvalidSHA256Length(t *testing.T) {
	// SHA256 must be exactly 64 hex characters
	short := strings.Repeat("a", 63)
	err := validateDownloadInfo("lts", "1.0.5", "win32-x64", DownloadInfo{
		Name:   "sdk.zip",
		URL:    "https://example.com/sdk.zip",
		SHA256: short,
	})
	assert.Error(t, err, "SHA256 shorter than 64 chars should fail")
}

func TestValidateDownloadInfo_InvalidSHA256Chars(t *testing.T) {
	// 64 chars but contains non-hex characters
	badHash := strings.Repeat("g", 64)
	err := validateDownloadInfo("lts", "1.0.5", "win32-x64", DownloadInfo{
		Name:   "sdk.zip",
		URL:    "https://example.com/sdk.zip",
		SHA256: badHash,
	})
	assert.Error(t, err, "non-hex SHA256 should fail")
}

func TestValidateDownloadInfo_Valid(t *testing.T) {
	validHash := strings.Repeat("ab", 32) // 64 hex chars
	err := validateDownloadInfo("lts", "1.0.5", "win32-x64", DownloadInfo{
		Name:   "sdk.zip",
		URL:    "https://example.com/sdk.zip",
		SHA256: validHash,
	})
	assert.NoError(t, err)
}

func TestParseManifest_ValidManifest(t *testing.T) {
	validHash := strings.Repeat("ab", 32)
	json := `{
  "channels": {
    "lts": {
      "latest": "1.0.0",
      "versions": {
        "1.0.0": {
          "win32-x64": {
            "name": "sdk.zip",
            "sha256": "` + validHash + `",
            "url": "https://example.com/sdk.zip"
          }
        }
      }
    },
    "sts": {
      "latest": "2.0.0",
      "versions": {
        "2.0.0": {
          "win32-x64": {
            "name": "sdk.zip",
            "sha256": "` + validHash + `",
            "url": "https://example.com/sdk2.zip"
          }
        }
      }
    }
  }
}`
	m, err := ParseManifest([]byte(json))
	require.NoError(t, err)
	assert.NotNil(t, m)
}
