package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupComponentCLITest(t *testing.T, tcName string) string {
	t.Helper()
	home := t.TempDir()
	config.IsolateForTest(t, home)
	config.ResetDefaultSettingsFileCache()
	tcDir := filepath.Join(home, "toolchains", tcName)
	require.NoError(t, os.MkdirAll(tcDir, 0o755))
	return tcDir
}

func TestRunComponentRemoveBestEffort(t *testing.T) {
	app := newApplication("dev", "")
	tcName := "lts-1.0.5"
	tcDir := setupComponentCLITest(t, tcName)
	roots, err := componentlib.RootsFor(tcName)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(roots.StdxDir, "dynamic"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(roots.StdxDir, "dynamic", "libfoo.so"), []byte("x"), 0o644))
	require.NoError(t, componentlib.WriteManifest(tcDir, componentlib.Stdx, []string{"dynamic/libfoo.so"}))

	app.componentToolchain = tcName

	err = app.runComponentRemove(&cobra.Command{}, []string{"bogus", "stdx", "docs"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
	assert.Contains(t, err.Error(), "docs")
	assert.False(t, componentlib.IsInstalled(tcDir, componentlib.Stdx))
	assert.NoFileExists(t, filepath.Join(roots.StdxDir, "dynamic", "libfoo.so"))
}

func TestResolveToolchainArgValidationAndActiveFallback(t *testing.T) {
	_, _, err := resolveToolchainArg("+bad")
	require.Error(t, err)

	home := t.TempDir()
	config.IsolateForTest(t, home)
	config.ResetDefaultSettingsFileCache()
	t.Cleanup(config.ResetDefaultSettingsFileCache)
	require.NoError(t, config.EnsureDirs())

	_, _, err = resolveToolchainArg("lts-1.0.5")
	require.Error(t, err)

	tcName := "lts-1.0.5"
	tcDir := filepath.Join(home, "toolchains", tcName)
	require.NoError(t, os.MkdirAll(tcDir, 0o755))
	settings := config.DefaultSettings()
	settings.DefaultToolchain = tcName
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	gotDir, gotName, err := resolveToolchainArg("")

	require.NoError(t, err)
	assert.Equal(t, tcDir, gotDir)
	assert.Equal(t, tcName, gotName.String())
}

func TestInstallComponentsListRollsBackPreviousComponentOnLaterFailure(t *testing.T) {
	app := newApplication("dev", "")
	tcName := "lts-1.0.5"
	tcDir := setupComponentCLITest(t, tcName)

	app.componentInstallFunc = func(ctx context.Context, roots componentlib.Roots, tc toolchain.ToolchainName, name componentlib.Name, tuple, downloadsDir string, force bool) error {
		if name == componentlib.Docs {
			return errors.New("docs failed")
		}
		require.NoError(t, os.MkdirAll(filepath.Join(roots.StdxDir, "dynamic"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(roots.StdxDir, "dynamic", "libfoo.so"), []byte("x"), 0o644))
		return componentlib.WriteManifest(roots.TcDir, name, []string{"dynamic/libfoo.so"})
	}

	err := app.installComponentsList(context.Background(), tcName, []string{"stdx", "docs"}, false, true)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs failed")
	assert.False(t, componentlib.IsInstalled(tcDir, componentlib.Stdx))
	stdxDir, dirErr := config.StdxDirFor(tcName)
	require.NoError(t, dirErr)
	assert.NoFileExists(t, filepath.Join(stdxDir, "dynamic", "libfoo.so"))
}

func TestInstallComponentsListUsesTargetTupleForTargetVariant(t *testing.T) {
	app := newApplication("dev", "")
	const targetName = "lts-1.0.5-linux-x64-ohos"
	setupComponentCLITest(t, targetName)

	var gotTuple string
	var gotTcDir string

	app.componentInstallFunc = func(_ context.Context, roots componentlib.Roots, _ toolchain.ToolchainName, _ componentlib.Name, tuple, _ string, _ bool) error {
		gotTuple = tuple
		gotTcDir = roots.TcDir
		return nil
	}

	err := app.installComponentsList(context.Background(), targetName, []string{"stdx"}, false, true)
	require.NoError(t, err)

	// The target tuple encoded in the resolved name drives the stdx download,
	// not the host tuple.
	assert.Equal(t, "linux-x64-ohos", gotTuple)
	// Roots (and thus the manifest + StdxDir) are keyed by the full target name.
	assert.Equal(t, targetName, filepath.Base(gotTcDir))
}

func TestRunComponentListQuietShowsInstalledThenAvailable(t *testing.T) {
	app := newApplication("dev", "")
	tcName := "lts-1.0.5"
	tcDir := setupComponentCLITest(t, tcName)
	require.NoError(t, componentlib.WriteManifest(tcDir, componentlib.Docs, []string{"index.html"}))

	app.componentToolchain = tcName
	app.componentListQuiet = true
	app.componentListInstalledOnly = false

	stdout, err := captureStdout(t, func() error {
		return app.runComponentList(&cobra.Command{}, nil)
	})

	require.NoError(t, err)
	lines := strings.Fields(stdout)
	assert.Equal(t, []string{"docs", "stdx", "stdx-docs"}, lines)

	app.componentListInstalledOnly = true
	stdout, err = captureStdout(t, func() error {
		return app.runComponentList(&cobra.Command{}, nil)
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"docs"}, strings.Fields(stdout))
}

func TestRunComponentAddInstallsForResolvedToolchain(t *testing.T) {
	app := newApplication("dev", "")
	tcName := "lts-1.0.5"
	tcDir := setupComponentCLITest(t, tcName)

	app.componentToolchain = tcName
	app.componentAddForce = true
	var gotForce bool
	app.componentInstallFunc = func(ctx context.Context, roots componentlib.Roots, tc toolchain.ToolchainName, name componentlib.Name, tuple, downloadsDir string, force bool) error {
		gotForce = force
		return componentlib.WriteManifest(roots.TcDir, name, []string{"index.html"})
	}

	err := app.runComponentAdd(&cobra.Command{}, []string{"docs"})

	require.NoError(t, err)
	assert.True(t, gotForce)
	assert.True(t, componentlib.IsInstalled(tcDir, componentlib.Docs))
}

func TestRunComponentAddRejectsCustomToolchain(t *testing.T) {
	app := newApplication("dev", "")
	tcName := "local-sdk"
	setupComponentCLITest(t, tcName)

	app.componentToolchain = tcName

	err := app.runComponentAdd(&cobra.Command{}, []string{"docs"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs")
}

func TestRunComponentRemoveReturnsOnlyParseErrorsWhenNoValidComponents(t *testing.T) {
	app := newApplication("dev", "")
	err := app.runComponentRemove(&cobra.Command{}, []string{"bogus", "unknown"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
	assert.Contains(t, err.Error(), "unknown")
}

func TestRunComponentListInstalledOnlyNoComponents(t *testing.T) {
	app := newApplication("dev", "")
	tcName := "lts-1.0.5"
	setupComponentCLITest(t, tcName)

	app.componentToolchain = tcName
	app.componentListQuiet = false
	app.componentListInstalledOnly = true

	stdout, err := captureStdout(t, func() error {
		return app.runComponentList(&cobra.Command{}, nil)
	})

	require.NoError(t, err)
	assert.NotEmpty(t, stdout)
}

func TestRunComponentListNonQuietShowsInstalledAndAvailable(t *testing.T) {
	app := newApplication("dev", "")
	tcName := "lts-1.0.5"
	tcDir := setupComponentCLITest(t, tcName)
	require.NoError(t, componentlib.WriteManifest(tcDir, componentlib.Docs, []string{"index.html"}))

	app.componentToolchain = tcName
	app.componentListQuiet = false
	app.componentListInstalledOnly = false

	stdout, err := captureStdout(t, func() error {
		return app.runComponentList(&cobra.Command{}, nil)
	})

	require.NoError(t, err)
	assert.Contains(t, stdout, "docs")
	assert.Contains(t, stdout, "stdx")
}

func TestRunComponentLinkInvokesLinkFunc(t *testing.T) {
	app := newApplication("dev", "")
	tcName := "lts-1.0.5"
	tcDir := setupComponentCLITest(t, tcName)
	src := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "dynamic"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(src, "static"), 0o755))

	app.componentToolchain = tcName
	app.componentLinkForce = true
	var gotForce bool
	var gotSource string
	var gotTcDir string
	app.componentLinkFunc = func(roots componentlib.Roots, name componentlib.Name, source string, force bool) (string, error) {
		gotForce = force
		gotSource = source
		gotTcDir = roots.TcDir
		return source, componentlib.WriteManifest(roots.TcDir, name, []string{"dynamic", "static"})
	}

	err := app.runComponentLink(&cobra.Command{}, []string{"stdx", src})

	require.NoError(t, err)
	assert.True(t, gotForce)
	assert.Equal(t, src, gotSource)
	assert.Equal(t, tcDir, gotTcDir)
	assert.True(t, componentlib.IsInstalled(tcDir, componentlib.Stdx))
}

func TestRunComponentLinkRejectsNonStdxComponent(t *testing.T) {
	app := newApplication("dev", "")
	tcName := "lts-1.0.5"
	setupComponentCLITest(t, tcName)

	app.componentToolchain = tcName

	err := app.runComponentLink(&cobra.Command{}, []string{"docs", t.TempDir()})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs")
}

func TestRunComponentLinkRejectsUnknownComponentName(t *testing.T) {
	app := newApplication("dev", "")
	tcName := "lts-1.0.5"
	setupComponentCLITest(t, tcName)

	app.componentToolchain = tcName

	err := app.runComponentLink(&cobra.Command{}, []string{"bogus", t.TempDir()})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
}

func TestRunComponentLinkAllowsCustomToolchain(t *testing.T) {
	app := newApplication("dev", "")
	tcName := "local-sdk"
	tcDir := setupComponentCLITest(t, tcName)
	src := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "dynamic"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(src, "static"), 0o755))

	app.componentToolchain = tcName
	var called bool
	app.componentLinkFunc = func(roots componentlib.Roots, name componentlib.Name, source string, force bool) (string, error) {
		called = true
		return source, componentlib.WriteManifest(roots.TcDir, name, []string{"dynamic", "static"})
	}

	err := app.runComponentLink(&cobra.Command{}, []string{"stdx", src})

	require.NoError(t, err)
	assert.True(t, called, "Link should not be gated by IsCustom")
	assert.True(t, componentlib.IsInstalled(tcDir, componentlib.Stdx))
}

func TestInstallComponentsForToolchainUsesInstalledToolchain(t *testing.T) {
	app := newApplication("dev", "")
	tcName := "lts-1.0.5"
	tcDir := setupComponentCLITest(t, tcName)

	app.componentInstallFunc = func(ctx context.Context, roots componentlib.Roots, tc toolchain.ToolchainName, name componentlib.Name, tuple, downloadsDir string, force bool) error {
		return componentlib.WriteManifest(roots.TcDir, name, []string{"index.html"})
	}

	err := app.InstallComponentsForToolchain(context.Background(), "lts", []string{"docs"})

	require.NoError(t, err)
	assert.True(t, componentlib.IsInstalled(tcDir, componentlib.Docs))
}
