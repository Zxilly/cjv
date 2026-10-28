package showcmd

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clitest"
	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/toolchain"
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
	c := newTestCommands()
	tcName := "lts-1.0.5"
	tcDir := clitest.Home(t, tcName)
	roots, err := componentlib.RootsFor(tcName)
	require.NoError(t, err)
	docFile := filepath.Join(roots.DocsDir, "main", "index.html")
	require.NoError(t, os.MkdirAll(filepath.Dir(docFile), 0o755))
	require.NoError(t, os.WriteFile(docFile, []byte("docs"), 0o644))
	require.NoError(t, componentlib.WriteManifest(tcDir, componentlib.Docs, []string{"index.html"}))

	c.docPath = true
	c.docToolchain = tcName

	stdout, err := clitest.CommandOutput(t, func(cmd *cobra.Command) error {
		return c.runDoc(cmd, nil)
	})

	require.NoError(t, err)
	assert.Equal(t, docFile, strings.TrimSpace(stdout))
}

func TestRunDocUsesTrackingInstallationIdentity(t *testing.T) {
	for _, selection := range []string{"flag", "selector", "active"} {
		t.Run(selection, func(t *testing.T) {
			tcDir := clitest.Home(t, "lts")
			require.NoError(t, toolchain.WriteInstallation(tcDir, toolchain.Installation{Release: "lts-1.0.5"}))
			roots, err := componentlib.RootsFor("lts")
			require.NoError(t, err)
			docFile := filepath.Join(roots.DocsDir, "main", "index.html")
			require.NoError(t, os.MkdirAll(filepath.Dir(docFile), 0o755))
			require.NoError(t, os.WriteFile(docFile, []byte("docs"), 0o644))
			require.NoError(t, componentlib.WriteManifest(tcDir, componentlib.Docs, []string{"index.html"}))

			args := []string{"--json", "doc"}
			switch selection {
			case "flag":
				args = append(args, "--toolchain", "lts")
			case "selector":
				args = append([]string{"+lts"}, args...)
			case "active":
				t.Setenv(config.EnvToolchain, "lts")
			}
			stdout, err := clitest.Execute(t, clitest.Root(Register), args...)
			require.NoError(t, err)
			var result docResult
			require.NoError(t, json.Unmarshal([]byte(stdout), &result))
			assert.Equal(t, "lts", result.Toolchain)
			assert.Equal(t, docFile, result.Path)
			assert.False(t, result.Opened)
		})
	}
}

func TestRunDocOpensEscapedFileURL(t *testing.T) {
	c := newTestCommands()
	tcName := "lts-1.0.5"
	tcDir := clitest.Home(t, tcName)
	roots, err := componentlib.RootsFor(tcName)
	require.NoError(t, err)
	docFile := filepath.Join(roots.DocsDir, "main", "docs #1", "index%.html")
	require.NoError(t, os.MkdirAll(filepath.Dir(docFile), 0o755))
	require.NoError(t, os.WriteFile(docFile, []byte("docs"), 0o644))
	require.NoError(t, componentlib.WriteManifest(tcDir, componentlib.Docs, []string{"docs #1/index%.html"}))

	var opened string
	c.docPath = false
	c.docToolchain = tcName
	c.openURL = func(u string) error {
		opened = u
		return nil
	}

	err = c.runDoc(&cobra.Command{}, []string{"docs #1/index%"})

	require.NoError(t, err)
	assert.Contains(t, opened, "%23")
	assert.Contains(t, opened, "%25")
}
