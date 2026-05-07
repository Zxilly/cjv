package component

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveDocPathRejectsEscapingTopic(t *testing.T) {
	tcDir := t.TempDir()
	docsRoot := t.TempDir()
	roots := Roots{TcDir: tcDir, DocsDir: docsRoot}
	mainDir := filepath.Join(docsRoot, "main")
	require.NoError(t, ensureFile(filepath.Join(mainDir, "index.html"), "index"))

	secret := filepath.Join(filepath.Dir(docsRoot), "secret.html")
	require.NoError(t, ensureFile(secret, "secret"))
	require.NoError(t, WriteManifest(tcDir, Docs, []string{"index.html"}))

	got, err := ResolveDocPath(roots, "../../secret")

	require.Error(t, err)
	assert.Empty(t, got)
}

func TestResolveDocPathFindsKnownAndRelativeTopics(t *testing.T) {
	tcDir := t.TempDir()
	docsRoot := t.TempDir()
	roots := Roots{TcDir: tcDir, DocsDir: docsRoot}
	mainDir := filepath.Join(docsRoot, "main")
	require.NoError(t, ensureFile(filepath.Join(mainDir, "index.html"), "index"))
	require.NoError(t, ensureFile(filepath.Join(mainDir, "libs", "std", "core", "core_package_overview.html"), "std"))
	require.NoError(t, ensureFile(filepath.Join(mainDir, "tools", "source_zh_cn", "index.html"), "tools"))
	require.NoError(t, WriteManifest(tcDir, Docs, []string{
		"index.html",
		"libs/std/core/core_package_overview.html",
		"tools/source_zh_cn/index.html",
	}))

	got, err := ResolveDocPath(roots, "")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(mainDir, "index.html"), got)

	got, err = ResolveDocPath(roots, "std")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(mainDir, "libs", "std", "core", "core_package_overview.html"), got)

	got, err = ResolveDocPath(roots, "tools/source_zh_cn")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(mainDir, "tools", "source_zh_cn", "index.html"), got)
}

func ensureFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
