package dist

import (
	"os"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
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

// --- Tests for ListVersions / VersionsByTuple / compareSemVerDesc ---

func loadFixtureManifest(t *testing.T) *Manifest {
	t.Helper()
	data, err := os.ReadFile("testdata/sdk-versions.json")
	require.NoError(t, err)
	m, err := ParseManifest(data)
	require.NoError(t, err)
	return m
}

func TestManifestListVersions_AllSortedDesc(t *testing.T) {
	m := loadFixtureManifest(t)
	versions, err := m.ListVersions(toolchain.LTS, "")
	require.NoError(t, err)
	// fixture LTS has 1.0.0 and 1.0.5
	assert.Equal(t, []string{"1.0.5", "1.0.0"}, versions)
}

func TestManifestListVersions_FilterByPlatformKey(t *testing.T) {
	m := loadFixtureManifest(t)
	versions, err := m.ListVersions(toolchain.LTS, "win32-x64")
	require.NoError(t, err)
	assert.Equal(t, []string{"1.0.5", "1.0.0"}, versions)

	versions, err = m.ListVersions(toolchain.LTS, "freebsd-amd64")
	require.NoError(t, err)
	assert.Empty(t, versions)
}

func TestManifestListVersions_PartialPlatformMatch(t *testing.T) {
	validHash := strings.Repeat("ab", 32)
	json := `{
  "channels": {
    "lts": {
      "latest": "1.0.5",
      "versions": {
        "1.0.0": {"linux-x64": {"name":"a","sha256":"` + validHash + `","url":"http://x"}},
        "1.0.5": {
          "linux-x64": {"name":"a","sha256":"` + validHash + `","url":"http://x"},
          "linux-x64-ohos": {"name":"b","sha256":"` + validHash + `","url":"http://y"}
        }
      }
    },
    "sts": {
      "latest": "2.0.0",
      "versions": {"2.0.0": {"linux-x64": {"name":"a","sha256":"` + validHash + `","url":"http://x"}}}
    }
  }
}`
	m, err := ParseManifest([]byte(json))
	require.NoError(t, err)

	hostOnly, err := m.ListVersions(toolchain.LTS, "linux-x64")
	require.NoError(t, err)
	assert.Equal(t, []string{"1.0.5", "1.0.0"}, hostOnly)

	withTarget, err := m.ListVersions(toolchain.LTS, "linux-x64-ohos")
	require.NoError(t, err)
	assert.Equal(t, []string{"1.0.5"}, withTarget, "ohos build only present for 1.0.5")
}

func TestManifestListVersions_UnknownChannel(t *testing.T) {
	m := loadFixtureManifest(t)
	_, err := m.ListVersions(toolchain.Nightly, "")
	require.Error(t, err)
}

func TestManifestListVersions_DoubleDigitMinorOrdersBySemver(t *testing.T) {
	validHash := strings.Repeat("ab", 32)
	json := `{
  "channels": {
    "lts": {
      "latest": "1.10.0",
      "versions": {
        "1.10.0": {"linux-x64": {"name":"a","sha256":"` + validHash + `","url":"http://x"}},
        "1.2.0":  {"linux-x64": {"name":"a","sha256":"` + validHash + `","url":"http://x"}},
        "1.1.0-beta.1": {"linux-x64": {"name":"a","sha256":"` + validHash + `","url":"http://x"}}
      }
    },
    "sts": {"latest":"2.0.0","versions":{"2.0.0":{"linux-x64":{"name":"a","sha256":"` + validHash + `","url":"http://x"}}}}
  }
}`
	m, err := ParseManifest([]byte(json))
	require.NoError(t, err)

	versions, err := m.ListVersions(toolchain.LTS, "")
	require.NoError(t, err)
	// Critical: 1.10.0 must precede 1.2.0 (semver, not lexical).
	assert.Equal(t, []string{"1.10.0", "1.2.0", "1.1.0-beta.1"}, versions)
}

func TestManifestListVersions_PrereleaseOrdering(t *testing.T) {
	validHash := strings.Repeat("ab", 32)
	json := `{
  "channels": {
    "lts": {
      "latest": "1.0.0",
      "versions": {
        "1.0.0": {"linux-x64": {"name":"a","sha256":"` + validHash + `","url":"http://x"}},
        "1.0.0-beta.1": {"linux-x64": {"name":"a","sha256":"` + validHash + `","url":"http://x"}}
      }
    },
    "sts": {"latest":"2.0.0","versions":{"2.0.0":{"linux-x64":{"name":"a","sha256":"` + validHash + `","url":"http://x"}}}}
  }
}`
	m, err := ParseManifest([]byte(json))
	require.NoError(t, err)

	versions, err := m.ListVersions(toolchain.LTS, "")
	require.NoError(t, err)
	// SemVer rule: 1.0.0-beta.1 < 1.0.0
	assert.Equal(t, []string{"1.0.0", "1.0.0-beta.1"}, versions)
}

func TestManifestVersionsByTuple_All(t *testing.T) {
	m := loadFixtureManifest(t)
	got, err := m.VersionsByTuple(toolchain.LTS)
	require.NoError(t, err)

	expected := map[string][]string{
		"win32-x64":    {"1.0.5", "1.0.0"},
		"darwin-arm64": {"1.0.5", "1.0.0"},
		"linux-x64":    {"1.0.5", "1.0.0"},
	}
	assert.Equal(t, expected, got)
}

func TestManifestVersionsByTuple_UnknownChannel(t *testing.T) {
	m := loadFixtureManifest(t)
	_, err := m.VersionsByTuple(toolchain.Nightly)
	require.Error(t, err)
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

func TestComponentDownload(t *testing.T) {
	var m Manifest
	m.Channels.STS = ChannelInfo{
		Latest:   "1.1.0-beta.25",
		Versions: map[string]map[string]DownloadInfo{"1.1.0-beta.25": {"linux-x64": {Name: "sdk", SHA256: "x", URL: "u"}}},
		Components: map[string]ComponentSet{
			"1.1.0-beta.25": {
				Docs:     &ComponentInfo{Name: "docs", URL: "https://example/docs"},
				StdxDocs: &ComponentInfo{Name: "stdx-docs", URL: "https://example/stdx-docs"},
				Stdx: map[string]ComponentInfo{
					"linux-x64": {Name: "stdx-linux", URL: "https://example/stdx/linux-x64"},
				},
			},
		},
	}

	stdx, err := m.ComponentDownload(toolchain.STS, "1.1.0-beta.25", "stdx", "linux-x64")
	require.NoError(t, err)
	assert.Equal(t, "https://example/stdx/linux-x64", stdx.URL)

	docs, err := m.ComponentDownload(toolchain.STS, "1.1.0-beta.25", "docs", "")
	require.NoError(t, err)
	assert.Equal(t, "https://example/docs", docs.URL)

	stdxDocs, err := m.ComponentDownload(toolchain.STS, "1.1.0-beta.25", "stdx-docs", "")
	require.NoError(t, err)
	assert.Equal(t, "https://example/stdx-docs", stdxDocs.URL)

	// Missing stdx platform → not published, carrying the platform token.
	_, err = m.ComponentDownload(toolchain.STS, "1.1.0-beta.25", "stdx", "windows-x64")
	var notPub *cjverr.ComponentNotPublishedError
	require.ErrorAs(t, err, &notPub)
	assert.Equal(t, "windows-x64", notPub.Target)

	// Version with no component set at all.
	_, err = m.ComponentDownload(toolchain.STS, "9.9.9", "docs", "")
	require.ErrorAs(t, err, &notPub)
}

func TestParseManifestWithComponents(t *testing.T) {
	validHash := strings.Repeat("a", 64)
	json := `{
  "channels": {
    "lts": {"latest": "1.0.0", "versions": {"1.0.0": {"win32-x64": {"name": "s", "sha256": "` + validHash + `", "url": "u"}}}},
    "sts": {
      "latest": "1.1.0",
      "versions": {"1.1.0": {"win32-x64": {"name": "s", "sha256": "` + validHash + `", "url": "u"}}},
      "components": {"1.1.0": {"docs": {"name": "d", "url": "https://example/d"}, "stdx": {"linux-x64": {"name": "x", "url": "https://example/x"}}}}
    }
  }
}`
	m, err := ParseManifest([]byte(json))
	require.NoError(t, err)
	info, err := m.ComponentDownload(toolchain.STS, "1.1.0", "stdx", "linux-x64")
	require.NoError(t, err)
	assert.Equal(t, "https://example/x", info.URL)
	assert.True(t, m.HasComponents(toolchain.STS, "1.1.0"))
	assert.False(t, m.HasComponents(toolchain.LTS, "1.0.0"))
}
