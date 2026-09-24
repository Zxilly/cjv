package dist

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sourceTestSHA256 = "0000000000000000000000000000000000000000000000000000000000000000"

func sourceTestManifest(sdkURL string) string {
	return fmt.Sprintf(`{
  "channels": {
    "lts": {
      "latest": "1.0.0",
      "versions": {
        "1.0.0": {
          "linux-x64": {
            "name": "cangjie-sdk-linux-x64-1.0.0.tar.gz",
            "sha256": %q,
            "url": %q
          }
        }
      }
    },
    "sts": {
      "latest": "1.1.0",
      "versions": {
        "1.1.0": {
          "linux-x64": {
            "name": "cangjie-sdk-linux-x64-1.1.0.tar.gz",
            "sha256": %q,
            "url": "sdk/cangjie-sdk-linux-x64-1.1.0.tar.gz"
          }
        }
      }
    }
  }
}`, sourceTestSHA256, sdkURL, sourceTestSHA256)
}

func sourceTestManifestWithNightly(nightlySHA string) string {
	return fmt.Sprintf(`{
  "channels": {
    "lts": {"latest":"1.0.0","versions":{"1.0.0":{"linux-x64":{"name":"lts.tar.gz","sha256":%q,"url":"sdk/lts.tar.gz"}}}},
    "sts": {"latest":"1.1.0","versions":{"1.1.0":{"linux-x64":{"name":"sts.tar.gz","sha256":%q,"url":"sdk/sts.tar.gz"}}}},
    "nightly": {
      "latest":"1.2.0-alpha.20260822010101",
      "versions": {
        "1.2.0-alpha.20260822010101": {
          "linux-x64": {
            "name":"nightly.tar.gz",
            "sha256":%q,
            "url":"nightly/nightly.tar.gz"
          }
        }
      },
      "components": {
        "1.2.0-alpha.20260822010101": {
          "docs": {
            "name":"docs.tar.gz",
            "sha256":%q,
            "url":"nightly/docs.tar.gz"
          }
        }
      }
    }
  }
}`, sourceTestSHA256, sourceTestSHA256, nightlySHA, sourceTestSHA256)
}

func sourceTestNightlyChannel(nightlySHA string) string {
	return fmt.Sprintf(`{
  "latest":"1.2.0-alpha.20260822010101",
  "versions": {
    "1.2.0-alpha.20260822010101": {
      "linux-x64": {
        "name":"nightly.tar.gz",
        "sha256":%q,
        "url":"nightly/nightly.tar.gz"
      }
    }
  },
  "components": {
    "1.2.0-alpha.20260822010101": {
      "docs": {
        "name":"docs.tar.gz",
        "sha256":%q,
        "url":"nightly/docs.tar.gz"
      }
    }
  }
}`, nightlySHA, sourceTestSHA256)
}

func TestSourceDistServerResolvesManifestAndArtifactURLs(t *testing.T) {
	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		_, _ = w.Write([]byte(sourceTestManifest("sdk/cangjie-sdk-linux-x64-1.0.0.tar.gz")))
	}))
	defer server.Close()

	settings := config.DefaultSettings()
	settings.DistServer = server.URL + "/corp/cjv/"
	source, err := NewSource(&settings)
	require.NoError(t, err)
	manifest, err := source.Manifest(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "/corp/cjv/versions.json", requestedPath)

	info, err := manifest.GetDownloadInfo(toolchain.LTS, "1.0.0", "linux-x64")
	require.NoError(t, err)
	assert.Equal(t, server.URL+"/corp/cjv/sdk/cangjie-sdk-linux-x64-1.0.0.tar.gz", info.URL)
}

func TestSourcePreservesAbsoluteArtifactURL(t *testing.T) {
	const artifactURL = "https://downloads.example.com/cangjie-sdk.tar.gz"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(sourceTestManifest(artifactURL)))
	}))
	defer server.Close()

	settings := config.DefaultSettings()
	settings.DistServer = server.URL + "/corp/cjv"
	source, err := NewSource(&settings)
	require.NoError(t, err)
	manifest, err := source.Manifest(context.Background())
	require.NoError(t, err)
	info, err := manifest.GetDownloadInfo(toolchain.LTS, "1.0.0", "linux-x64")
	require.NoError(t, err)
	assert.Equal(t, artifactURL, info.URL)
}

func TestSourceUsesConfiguredManifestURL(t *testing.T) {
	settings := config.DefaultSettings()
	settings.ManifestURL = "https://manifest.example.com/custom.json"
	source, err := NewSource(&settings)
	require.NoError(t, err)

	assert.Equal(t, settings.ManifestURL, source.manifestURL)
}

func TestSourceResolvesNightlyFromManifest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(sourceTestManifestWithNightly(sourceTestSHA256)))
	}))
	defer server.Close()

	settings := config.DefaultSettings()
	settings.DistServer = server.URL + "/corp/cjv"
	source, err := NewSource(&settings)
	require.NoError(t, err)

	release, err := source.ResolveToolchain(context.Background(), toolchain.Nightly, "", "linux-x64")
	require.NoError(t, err)
	assert.Equal(t, "1.2.0-alpha.20260822010101", release.Version)
	assert.Equal(t, server.URL+"/corp/cjv/nightly/nightly.tar.gz", release.Download.URL)
}

func TestSourceResolvesNightlyFromSeparateManifest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/corp/cjv/versions.json":
			_, _ = w.Write([]byte(sourceTestManifest("sdk/lts.tar.gz")))
		case "/corp/cjv/nightly.json":
			_, _ = w.Write([]byte(sourceTestNightlyChannel(sourceTestSHA256)))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	settings := config.DefaultSettings()
	settings.DistServer = server.URL + "/corp/cjv"
	source, err := NewSource(&settings)
	require.NoError(t, err)

	release, err := source.ResolveToolchain(context.Background(), toolchain.Nightly, "", "linux-x64")
	require.NoError(t, err)
	assert.Equal(t, "1.2.0-alpha.20260822010101", release.Version)
	assert.Equal(t, server.URL+"/corp/cjv/nightly/nightly.tar.gz", release.Download.URL)
}

func TestSourceLTSResolutionSkipsNightlyManifest(t *testing.T) {
	var stableRequests, nightlyRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/corp/cjv/versions.json":
			stableRequests++
			_, _ = w.Write([]byte(sourceTestManifest("sdk/lts.tar.gz")))
		case "/corp/cjv/nightly.json":
			nightlyRequests++
			http.Error(w, "nightly should stay lazy", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	settings := config.DefaultSettings()
	settings.DistServer = server.URL + "/corp/cjv"
	source, err := NewSource(&settings)
	require.NoError(t, err)

	_, err = source.ResolveToolchain(context.Background(), toolchain.LTS, "", "linux-x64")
	require.NoError(t, err)
	assert.Equal(t, 1, stableRequests)
	assert.Zero(t, nightlyRequests)
}

func TestSourceNightlyFallsBackToAggregatedManifestDuringMigration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/corp/cjv/versions.json":
			_, _ = w.Write([]byte(sourceTestManifestWithNightly(sourceTestSHA256)))
		case "/corp/cjv/nightly.json":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	settings := config.DefaultSettings()
	settings.DistServer = server.URL + "/corp/cjv"
	source, err := NewSource(&settings)
	require.NoError(t, err)

	release, err := source.ResolveToolchain(context.Background(), toolchain.Nightly, "", "linux-x64")
	require.NoError(t, err)
	assert.Equal(t, "1.2.0-alpha.20260822010101", release.Version)
}

func TestSourceResolvesNightlyChecksumSidecar(t *testing.T) {
	var sidecarRequests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions.json":
			_, _ = w.Write([]byte(sourceTestManifestWithNightly("")))
		case "/nightly/nightly.tar.gz.sha256":
			sidecarRequests = append(sidecarRequests, r.URL.Path)
			_, _ = w.Write([]byte(sourceTestSHA256 + "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/versions.json"
	source, err := NewSource(&settings)
	require.NoError(t, err)

	release, err := source.ResolveToolchain(context.Background(), toolchain.Nightly, "", "linux-x64")
	require.NoError(t, err)
	// The sidecar sits next to the asset the manifest links to.
	assert.Equal(t, []string{"/nightly/nightly.tar.gz.sha256"}, sidecarRequests)
	assert.Equal(t, sourceTestSHA256, release.Download.SHA256)
}

func TestSourceMissingNightlyReturnsManifestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(sourceTestManifest("sdk/lts.tar.gz")))
	}))
	defer server.Close()

	settings := config.DefaultSettings()
	settings.DistServer = server.URL + "/corp/cjv"
	source, err := NewSource(&settings)
	require.NoError(t, err)

	_, err = source.ResolveToolchain(context.Background(), toolchain.Nightly, "", "linux-x64")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrManifestChannelMissing), err)
}

func TestSourceResolvesNightlyComponentFromManifest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(sourceTestManifestWithNightly(sourceTestSHA256)))
	}))
	defer server.Close()

	settings := config.DefaultSettings()
	settings.DistServer = server.URL + "/corp/cjv"
	source, err := NewSource(&settings)
	require.NoError(t, err)

	component, err := source.ResolveComponent(
		context.Background(),
		toolchain.Nightly,
		"1.2.0-alpha.20260822010101",
		"docs",
		"",
	)
	require.NoError(t, err)
	assert.Equal(t, "docs.tar.gz", component.Name)
	assert.Equal(t, sourceTestSHA256, component.SHA256)
	assert.Equal(t, server.URL+"/corp/cjv/nightly/docs.tar.gz", component.URL)
}
