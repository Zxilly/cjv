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

func sourceTestManifestWithNightly() string {
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
            "url":"nightly/nightly.tar.gz",
            "release_tag":"nightly-20260822"
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
}`, sourceTestSHA256, sourceTestSHA256, sourceTestSHA256, sourceTestSHA256)
}

func TestSourceUnifiedRootResolvesManifestAndArtifactURLs(t *testing.T) {
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
	assert.True(t, source.Unified())

	manifest, err := source.Manifest(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "/corp/cjv/versions.json", requestedPath)

	info, err := manifest.GetDownloadInfo(toolchain.LTS, "1.0.0", "linux-x64")
	require.NoError(t, err)
	assert.Equal(t, server.URL+"/corp/cjv/sdk/cangjie-sdk-linux-x64-1.0.0.tar.gz", info.URL)
}

func TestSourceUnifiedPreservesAbsoluteArtifactURL(t *testing.T) {
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

func TestSourceLegacyUsesConfiguredManifestURL(t *testing.T) {
	settings := config.DefaultSettings()
	settings.ManifestURL = "https://manifest.example.com/custom.json"
	source, err := NewSource(&settings)
	require.NoError(t, err)

	assert.False(t, source.Unified())
	assert.Equal(t, settings.ManifestURL, source.ManifestURL())
}

func TestSourceUnifiedResolvesNightlyFromManifestWithoutGitCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(sourceTestManifestWithNightly()))
	}))
	defer server.Close()

	settings := config.DefaultSettings()
	settings.DistServer = server.URL + "/corp/cjv"
	source, err := NewSource(&settings)
	require.NoError(t, err)

	release, err := source.ResolveToolchain(context.Background(), toolchain.Nightly, "", "linux-x64")
	require.NoError(t, err)
	assert.Equal(t, "1.2.0-alpha.20260822010101", release.Version)
	assert.Equal(t, "nightly-20260822", release.ReleaseTag)
	assert.Equal(t, server.URL+"/corp/cjv/nightly/nightly.tar.gz", release.Download.URL)
}

func TestSourceUnifiedMissingNightlyReturnsManifestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(sourceTestManifest("sdk/lts.tar.gz")))
	}))
	defer server.Close()

	settings := config.DefaultSettings()
	settings.DistServer = server.URL + "/corp/cjv"
	legacyCalled := false
	source, err := NewSourceWithOptions(&settings, SourceOptions{
		FetchLatestNightlyRelease: func(context.Context, string, string) (NightlyRelease, error) {
			legacyCalled = true
			return NightlyRelease{}, errors.New("legacy nightly adapter selected")
		},
	})
	require.NoError(t, err)

	_, err = source.ResolveToolchain(context.Background(), toolchain.Nightly, "", "linux-x64")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrManifestChannelMissing), err)
	assert.False(t, legacyCalled)
}

func TestSourceUnifiedResolvesNightlyComponentFromManifest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(sourceTestManifestWithNightly()))
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
		"nightly-20260822",
	)
	require.NoError(t, err)
	assert.Equal(t, "docs.tar.gz", component.Name)
	assert.Equal(t, sourceTestSHA256, component.SHA256)
	assert.Equal(t, server.URL+"/corp/cjv/nightly/docs.tar.gz", component.URL)
}
