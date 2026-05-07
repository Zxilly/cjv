package cli

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileURLEscapesSpecialCharacters(t *testing.T) {
	docPath := filepath.Join(t.TempDir(), "docs #1", "index?x%.html")

	got := fileURL(docPath)
	parsed, err := url.Parse(got)

	require.NoError(t, err)
	assert.Equal(t, "file", parsed.Scheme)
	assert.Empty(t, parsed.Fragment)
	assert.Empty(t, parsed.RawQuery)
	assert.Contains(t, got, "%23")
	assert.Contains(t, got, "%3F")
	assert.Contains(t, got, "%25")
}

func TestRunDocPathPrintsResolvedDoc(t *testing.T) {
	tcName := "lts-1.0.5"
	tcDir := setupComponentCLITest(t, tcName)
	roots, err := componentlib.RootsFor(tcName)
	require.NoError(t, err)
	docFile := filepath.Join(roots.DocsDir, "main", "index.html")
	require.NoError(t, os.MkdirAll(filepath.Dir(docFile), 0o755))
	require.NoError(t, os.WriteFile(docFile, []byte("docs"), 0o644))
	require.NoError(t, componentlib.WriteManifest(tcDir, componentlib.Docs, []string{"index.html"}))

	oldDocPath := docPath
	oldDocToolchain := docToolchain
	docPath = true
	docToolchain = tcName
	t.Cleanup(func() {
		docPath = oldDocPath
		docToolchain = oldDocToolchain
	})

	stdout, err := captureStdout(t, func() error {
		return runDoc(&cobra.Command{}, nil)
	})

	require.NoError(t, err)
	assert.Equal(t, docFile, strings.TrimSpace(stdout))
}

func TestRunDocOpensEscapedFileURL(t *testing.T) {
	tcName := "lts-1.0.5"
	tcDir := setupComponentCLITest(t, tcName)
	roots, err := componentlib.RootsFor(tcName)
	require.NoError(t, err)
	docFile := filepath.Join(roots.DocsDir, "main", "docs #1", "index%.html")
	require.NoError(t, os.MkdirAll(filepath.Dir(docFile), 0o755))
	require.NoError(t, os.WriteFile(docFile, []byte("docs"), 0o644))
	require.NoError(t, componentlib.WriteManifest(tcDir, componentlib.Docs, []string{"docs #1/index%.html"}))

	oldDocPath := docPath
	oldDocToolchain := docToolchain
	oldOpenURL := openURLFunc
	var opened string
	docPath = false
	docToolchain = tcName
	openURLFunc = func(u string) error {
		opened = u
		return nil
	}
	t.Cleanup(func() {
		docPath = oldDocPath
		docToolchain = oldDocToolchain
		openURLFunc = oldOpenURL
	})

	err = runDoc(&cobra.Command{}, []string{"docs #1/index%"})

	require.NoError(t, err)
	assert.Contains(t, opened, "%23")
	assert.Contains(t, opened, "%25")
}
