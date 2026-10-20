// Package clitest provides the test harness shared by cjv's command
// packages: a TestMain that keeps tests from polluting the developer's PATH,
// output capture, and a root command that runs one command group the way
// cli.Execute runs the full tree.
package clitest

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Main runs a command package's tests. Tests isolate process environment and
// sometimes redirect os.Stdout, so they remain sequential even though each
// invocation owns its command state.
func Main(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	if os.Getenv("CI") == "true" {
		// CI mode: let reachable.ConfigurePath run for real so the actual
		// code path is exercised, but wrap the run in a platform-specific
		// guard that saves and restores the system PATH afterward.
		return runWithPathGuard(m)
	}

	// Local dev: disable PATH writes entirely to avoid polluting the
	// developer's system. Each test uses a unique t.TempDir() as
	// CJV_HOME, so without this override every test run would append
	// a new temp-dir entry to the system PATH.
	if err := os.Setenv(config.EnvNoPathSetup, "1"); err != nil {
		panic(err)
	}
	config.ResetDefaultSettingsFileCache()
	return m.Run()
}

// Register adds one command group to root.
type Register func(root *cobra.Command, ctx *clictx.Context)

// Root builds a minimal cjv root holding the groups from register, with the
// global flags and argument handling of the real tree.
func Root(register ...Register) *clictx.Context {
	ctx := clictx.New("dev", "")
	root := &cobra.Command{Use: "cjv", SilenceUsage: true, SilenceErrors: true}
	ctx.ConfigureRoot(root)
	for _, r := range register {
		r(root, ctx)
	}
	return ctx
}

// Execute runs args through ctx's root and returns what the command tree
// wrote to its output writer, including JSON error envelopes.
func Execute(t *testing.T, ctx *clictx.Context, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	ctx.Root.SetOut(&out)
	err := ctx.Execute(args)
	return out.String(), err
}

// CommandOutput runs fn with a command whose output writer is a buffer, and
// returns what the command wrote.
func CommandOutput(t *testing.T, fn func(cmd *cobra.Command) error) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	err := fn(cmd)
	return out.String(), err
}

// CaptureStdout returns what fn wrote to os.Stdout.
func CaptureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	runErr := fn()

	os.Stdout = oldStdout
	require.NoError(t, w.Close())
	data, readErr := io.ReadAll(r)
	require.NoError(t, readErr)
	require.NoError(t, r.Close())
	return string(data), runErr
}

// Home isolates a fresh CJV_HOME containing an empty toolchain directory
// named tcName, and returns that directory.
func Home(t *testing.T, tcName string) string {
	t.Helper()
	home := t.TempDir()
	configtest.Isolate(t, home)
	tcDir := filepath.Join(home, "toolchains", tcName)
	require.NoError(t, os.MkdirAll(tcDir, 0o755))
	return tcDir
}

// NightlyDistHome isolates a fresh CJV_HOME whose distribution server
// publishes a nightly SDK with components. It returns the home and the
// toolchain name to install.
func NightlyDistHome(t *testing.T) (string, string) {
	t.Helper()
	home := t.TempDir()
	configtest.Isolate(t, home)
	t.Setenv(config.EnvNoPathSetup, "1")
	t.Setenv(config.EnvDistServer, "")
	t.Setenv(config.EnvFallbackSettings, filepath.Join(home, "missing-fallback.toml"))
	server := testutil.SplitNightlyMockServer(t)
	settings := config.DefaultSettings()
	settings.DistServer = server.URL + "/corp/cjv"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
	return home, "nightly"
}

// UseManifest isolates a fresh CJV_HOME whose settings point at the
// sdk-versions.json served by serverURL, and returns the home.
func UseManifest(t *testing.T, serverURL string) string {
	t.Helper()
	home := t.TempDir()
	configtest.Isolate(t, home)
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	url := serverURL + "/sdk-versions.json"
	_, err = sf.Update(config.SettingsUpdate{ManifestURL: &url})
	require.NoError(t, err)
	return home
}

// InstallToolchain installs name through the production entry with ctx's
// lifecycle adapters, as `cjv install <name>` would.
func InstallToolchain(t *testing.T, ctx *clictx.Context, name string) {
	t.Helper()
	require.NoError(t, lifecycle.Install(context.Background(), lifecycle.InstallRequest{Toolchain: name}, ctx.LifecycleOptions()))
}
