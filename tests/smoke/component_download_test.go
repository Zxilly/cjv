//go:build smoke

package smoke

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSmokeRealComponentDownloads_LTSSTS(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()

	manifest := fetchSmokeManifest(t, ctx)
	platformKey, err := dist.CurrentTargetTuple("", "")
	require.NoError(t, err)

	downloadsDir := t.TempDir()
	for _, ch := range []toolchain.Channel{toolchain.LTS, toolchain.STS} {
		version, err := manifest.GetLatestVersion(ch)
		require.NoError(t, err)
		tc := toolchain.ToolchainName{Channel: ch, Version: version}

		for _, name := range component.KnownComponents() {
			t.Run(fmt.Sprintf("%s/%s", tc.String(), name), func(t *testing.T) {
				roots := component.Roots{
					TcDir:   t.TempDir(),
					DocsDir: t.TempDir(),
					StdxDir: t.TempDir(),
				}
				componentPlatformKey := ""
				if name == component.Stdx {
					componentPlatformKey = platformKey
				}

				require.NoError(t, component.Install(ctx, roots, tc, name, componentPlatformKey, downloadsDir, false))
				assert.True(t, component.IsInstalled(roots.TcDir, name))

				manifest, err := component.ReadManifest(roots.TcDir, name)
				require.NoError(t, err)
				assert.NotEmpty(t, manifest)
				assertSmokeComponentFileExists(t, roots, name, manifest[0])
			})
		}
	}
}

func fetchSmokeManifest(t *testing.T, ctx context.Context) *dist.Manifest {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.DefaultManifestURL, nil)
	require.NoError(t, err)
	resp, err := dist.HTTPClient().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck // smoke test cleanup

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(io.LimitReader(resp.Body, dist.MaxResponseSize))
	require.NoError(t, err)

	manifest, err := dist.ParseManifest(body)
	require.NoError(t, err)
	return manifest
}

func assertSmokeComponentFileExists(t *testing.T, roots component.Roots, name component.Name, relPath string) {
	t.Helper()

	base := roots.DocsDir
	switch name {
	case component.Stdx:
		base = roots.StdxDir
	case component.Docs:
		base = filepath.Join(roots.DocsDir, "main")
	case component.StdxDocs:
		base = filepath.Join(roots.DocsDir, "stdx")
	}
	assert.FileExists(t, filepath.Join(base, filepath.FromSlash(relPath)))
}
