package componentcmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clitest"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/testutil"

	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunComponentRemoveBestEffort(t *testing.T) {
	c := newTestCommands()
	tcName := "lts-1.0.5"
	tcDir := clitest.Home(t, tcName)
	roots, err := componentlib.RootsFor(tcName)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(roots.StdxDir, "dynamic"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(roots.StdxDir, "dynamic", "libfoo.so"), []byte("x"), 0o644))
	require.NoError(t, componentlib.WriteManifest(tcDir, componentlib.Stdx, []string{"dynamic/libfoo.so"}))

	c.toolchain = tcName

	err = c.runRemove(&cobra.Command{}, []string{"bogus", "stdx", "docs"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
	assert.Contains(t, err.Error(), "docs")
	assert.False(t, componentlib.IsInstalled(tcDir, componentlib.Stdx))
	assert.NoFileExists(t, filepath.Join(roots.StdxDir, "dynamic", "libfoo.so"))
}

func TestRunComponentListQuietShowsInstalledThenAvailable(t *testing.T) {
	c := newTestCommands()
	tcName := "lts-1.0.5"
	tcDir := clitest.Home(t, tcName)
	require.NoError(t, componentlib.WriteManifest(tcDir, componentlib.Docs, []string{"index.html"}))

	c.toolchain = tcName
	c.listQuiet = true
	c.listInstalledOnly = false

	stdout, err := clitest.CommandOutput(t, func(cmd *cobra.Command) error {
		return c.runList(cmd, nil)
	})

	require.NoError(t, err)
	lines := strings.Fields(stdout)
	assert.Equal(t, []string{"docs", "stdx", "stdx-docs"}, lines)

	c.listInstalledOnly = true
	stdout, err = clitest.CommandOutput(t, func(cmd *cobra.Command) error {
		return c.runList(cmd, nil)
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"docs"}, strings.Fields(stdout))
}

func TestRunComponentAddInstallsForResolvedToolchain(t *testing.T) {
	home, tcName := clitest.NightlyDistHome(t)
	require.NoError(t, lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: "nightly"}, lifecycle.Options{}))
	tcDir := filepath.Join(home, "toolchains", tcName)
	// A manifest without files marks docs installed; only --force replaces it
	// with the real component from the distribution source.
	require.NoError(t, componentlib.WriteManifest(tcDir, componentlib.Docs, nil))
	docsIndex := filepath.Join(home, "docs", tcName, "main", "index.html")

	_, err := execute(t, "component", "add", "docs", "--toolchain", tcName)
	require.NoError(t, err)
	assert.NoFileExists(t, docsIndex, "an installed component is kept without --force")

	_, err = execute(t, "component", "add", "docs", "--toolchain", tcName, "--force")
	require.NoError(t, err)
	assert.FileExists(t, docsIndex)
	assert.True(t, componentlib.IsInstalled(tcDir, componentlib.Docs))
}

func TestRunComponentAddRejectsCustomToolchain(t *testing.T) {
	c := newTestCommands()
	tcName := "local-sdk"
	clitest.Home(t, tcName)

	c.toolchain = tcName

	err := c.runAdd(&cobra.Command{}, []string{"docs"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs")
}

func TestRunComponentRemoveReturnsOnlyParseErrorsWhenNoValidComponents(t *testing.T) {
	c := newTestCommands()
	err := c.runRemove(&cobra.Command{}, []string{"bogus", "unknown"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
	assert.Contains(t, err.Error(), "unknown")
}

func TestRunComponentListInstalledOnlyNoComponents(t *testing.T) {
	c := newTestCommands()
	tcName := "lts-1.0.5"
	clitest.Home(t, tcName)

	c.toolchain = tcName
	c.listQuiet = false
	c.listInstalledOnly = true

	stdout, err := clitest.CommandOutput(t, func(cmd *cobra.Command) error {
		return c.runList(cmd, nil)
	})

	require.NoError(t, err)
	assert.NotEmpty(t, stdout)
}

func TestRunComponentListNonQuietShowsInstalledAndAvailable(t *testing.T) {
	c := newTestCommands()
	tcName := "lts-1.0.5"
	tcDir := clitest.Home(t, tcName)
	require.NoError(t, componentlib.WriteManifest(tcDir, componentlib.Docs, []string{"index.html"}))

	c.toolchain = tcName
	c.listQuiet = false
	c.listInstalledOnly = false

	stdout, err := clitest.CommandOutput(t, func(cmd *cobra.Command) error {
		return c.runList(cmd, nil)
	})

	require.NoError(t, err)
	assert.Contains(t, stdout, "docs")
	assert.Contains(t, stdout, "stdx")
}

func TestRunComponentLinkInvokesLinkFunc(t *testing.T) {
	c := newTestCommands()
	tcName := "lts-1.0.5"
	tcDir := clitest.Home(t, tcName)
	src := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "dynamic"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(src, "static"), 0o755))

	c.toolchain = tcName
	c.linkForce = true
	var gotForce bool
	var gotSource string
	var gotTcDir string
	c.link = func(roots componentlib.Roots, name componentlib.Name, source string, force bool) (string, error) {
		gotForce = force
		gotSource = source
		gotTcDir = roots.TcDir
		return source, componentlib.WriteManifest(roots.TcDir, name, []string{"dynamic", "static"})
	}

	err := c.runLink(&cobra.Command{}, []string{"stdx", src})

	require.NoError(t, err)
	assert.True(t, gotForce)
	assert.Equal(t, src, gotSource)
	assert.Equal(t, tcDir, gotTcDir)
	assert.True(t, componentlib.IsInstalled(tcDir, componentlib.Stdx))
}

func TestRunComponentLinkRejectsNonStdxComponent(t *testing.T) {
	c := newTestCommands()
	tcName := "lts-1.0.5"
	clitest.Home(t, tcName)

	c.toolchain = tcName

	err := c.runLink(&cobra.Command{}, []string{"docs", t.TempDir()})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs")
}

func TestRunComponentLinkRejectsUnknownComponentName(t *testing.T) {
	c := newTestCommands()
	tcName := "lts-1.0.5"
	clitest.Home(t, tcName)

	c.toolchain = tcName

	err := c.runLink(&cobra.Command{}, []string{"bogus", t.TempDir()})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
}

func TestRunComponentLinkAllowsCustomToolchain(t *testing.T) {
	c := newTestCommands()
	tcName := "local-sdk"
	tcDir := clitest.Home(t, tcName)
	src := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "dynamic"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(src, "static"), 0o755))

	c.toolchain = tcName
	var called bool
	c.link = func(roots componentlib.Roots, name componentlib.Name, source string, force bool) (string, error) {
		called = true
		return source, componentlib.WriteManifest(roots.TcDir, name, []string{"dynamic", "static"})
	}

	err := c.runLink(&cobra.Command{}, []string{"stdx", src})

	require.NoError(t, err)
	assert.True(t, called, "Link should not be gated by IsCustom")
	assert.True(t, componentlib.IsInstalled(tcDir, componentlib.Stdx))
}

func TestComponentAddJSONIncludesResultOnInstallAndSkip(t *testing.T) {
	home, name := clitest.NightlyDistHome(t)
	require.NoError(t, lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: "nightly"}, lifecycle.Options{}))
	for _, force := range []bool{false, false, true} {
		args := []string{"--json", "component", "add", "docs", "--toolchain", name}
		if force {
			args = append(args, "--force")
		}
		stdout, err := execute(t, args...)
		require.NoError(t, err)
		var result componentAddResult
		require.NoError(t, json.Unmarshal([]byte(stdout), &result), stdout)
		require.Equal(t, name, result.Toolchain)
		require.Equal(t, force, result.Forced)
		require.Len(t, result.Components, 1)
		require.Equal(t, "docs", string(result.Components[0]))
		require.FileExists(t, filepath.Join(home, "docs", name, "main", "index.html"))
	}
	stdout, err := execute(t, "--json", "component", "add", "not-a-component", "--toolchain", name)
	require.Error(t, err)
	require.True(t, json.Valid([]byte(stdout)), stdout)
}

func TestComponentProgressUsesCommandWriter(t *testing.T) {
	_, name := clitest.NightlyDistHome(t)
	require.NoError(t, lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: "nightly"}, lifecycle.Options{}))
	// The proxy component install reports to the sink it was given and
	// writes nothing on its own.
	recorder := &testutil.ProgressRecorder{}
	stdout, err := clitest.CaptureStdout(t, func() error {
		return lifecycle.InstallComponentsForToolchain(t.Context(), name, []string{"docs"}, lifecycle.Options{Progress: recorder})
	})
	require.NoError(t, err)
	require.Empty(t, stdout)
	require.Contains(t, recorder.Kinds, progress.ComponentInstalled)

	var commandOutput string
	stdout, err = clitest.CaptureStdout(t, func() error {
		var err error
		commandOutput, err = execute(t, "component", "add", "docs", "--toolchain", name, "--force")
		return err
	})
	require.NoError(t, err)
	require.Empty(t, stdout, "progress bypassed the command writer")
	require.Contains(t, commandOutput, "docs")
	require.Contains(t, commandOutput, name)
	// The test changes neither the process's real home nor its configured PATH.
	require.Equal(t, "1", os.Getenv(config.EnvNoPathSetup))
}

func TestAbsoluteSDKComponentsCannotMutateNamesake(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	external := filepath.Join(t.TempDir(), "outside-sdk")
	require.NoError(t, os.MkdirAll(external, 0o755))
	roots, err := componentlib.RootsFor("outside-sdk")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(roots.TcDir, 0o755))
	require.NoError(t, os.MkdirAll(roots.DocsDir, 0o755))
	marker := filepath.Join(roots.DocsDir, "keep.html")
	require.NoError(t, os.WriteFile(marker, []byte("owned by another installation"), 0o644))
	require.NoError(t, componentlib.WriteManifest(external, componentlib.Docs, []string{"keep.html"}))
	t.Setenv(config.EnvToolchain, external)
	_, err = execute(t, "component", "remove", "docs", "--json")
	require.ErrorContains(t, err, "external toolchain")
	require.FileExists(t, marker)
	require.True(t, componentlib.IsInstalled(external, componentlib.Docs))
}
