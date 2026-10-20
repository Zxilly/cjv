package toolchaincmd

import (
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/cli/clitest"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) { clitest.Main(m) }

func newTestContext() *clictx.Context {
	return clictx.New("dev", "")
}

// execute runs args through a fresh root holding this package's commands.
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	return clitest.Execute(t, clitest.Root(Register), args...)
}

// validManifestHome isolates a home whose manifest is testutil's valid mock
// server (lts 1.0.5, sts 2.0.0), and returns the home.
func validManifestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	configtest.Isolate(t, home)
	require.NoError(t, config.EnsureDirs())
	settings := config.DefaultSettings()
	settings.ManifestURL = testutil.ValidMockServer(t).URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
	return home
}
