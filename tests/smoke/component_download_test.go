//go:build smoke

package smoke

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSmokeRealComponentDownloads_LTSSTS(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()

	// The production distribution source against the default manifest: the
	// same resolution `cjv component add` performs.
	settings := config.DefaultSettings()
	source, err := dist.NewSource(&settings)
	require.NoError(t, err)
	mf, err := source.Manifest(ctx)
	require.NoError(t, err)
	platformKey, err := sdktarget.CurrentTargetTuple("", "")
	require.NoError(t, err)

	downloadsDir := t.TempDir()
	for _, ch := range []toolchain.Channel{toolchain.LTS, toolchain.STS} {
		versions, err := mf.ListVersions(ch, platformKey)
		require.NoError(t, err)
		// SDK releases can precede their optional component archives. Exercise
		// the newest release with components, and still require every install
		// below to succeed rather than silently skipping missing artifacts.
		var version string
		for _, candidate := range versions {
			if mf.HasComponents(ch, candidate) {
				version = candidate
				break
			}
		}
		require.NotEmpty(t, version, "%s has no release with components for %s", ch, platformKey)
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

				require.NoError(t, component.InstallFromSource(ctx, roots, tc, name, componentPlatformKey, downloadsDir, false, source, nil))
				assert.True(t, component.IsInstalled(roots.TcDir, name))

				manifest, err := component.ReadManifest(roots.TcDir, name)
				require.NoError(t, err)
				assert.NotEmpty(t, manifest)
				assertSmokeComponentFileExists(t, roots, name, manifest[0])
			})
		}
	}
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
