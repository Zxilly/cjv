package component

import (
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootsForUsesConfiguredHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	roots, err := RootsFor("lts-1.0.5")

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "toolchains", "lts-1.0.5"), roots.TcDir)
	assert.Equal(t, filepath.Join(home, "docs", "lts-1.0.5"), roots.DocsDir)
	assert.Equal(t, filepath.Join(home, "stdx", "lts-1.0.5"), roots.StdxDir)
}

func TestSpecStringAndInstallRoot(t *testing.T) {
	roots := Roots{DocsDir: "docs", StdxDir: "stdx"}
	spec, err := SpecFor(Docs)
	require.NoError(t, err)

	assert.Equal(t, "docs", spec.String())
	assert.Equal(t, filepath.Join("docs", "main"), spec.InstallRoot(roots))

	stdx, err := SpecFor(Stdx)
	require.NoError(t, err)
	assert.Equal(t, "stdx", stdx.InstallRoot(roots))
}

func TestAvailableComponentsFiltersUnresolvedToolchain(t *testing.T) {
	got := AvailableComponents(toolchain.ToolchainName{Channel: toolchain.LTS}, "linux-x64")
	assert.Empty(t, got)
}
