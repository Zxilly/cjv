package cli

import (
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// Tests for runInstall -- the cobra handler that turns flags into an
// InstallRequest. Resolution, download, extraction, validation and the
// transactional swap are lifecycle behaviour, tested in internal/lifecycle.

func TestRunInstall_WithoutForce(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	assert.NoError(t, config.EnsureDirs())

	server := testutil.ValidMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	assert.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	cmd := &cobra.Command{}
	cmd.Flags().BoolP("force", "f", false, "")
	err := app.runInstall(cmd, []string{"lts"})
	assert.NoError(t, err)
}

func TestRunInstall_InvalidName(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)

	cmd := &cobra.Command{}
	cmd.Flags().BoolP("force", "f", false, "")
	err := app.runInstall(cmd, []string{""})
	assert.Error(t, err, "empty name should fail")
}
