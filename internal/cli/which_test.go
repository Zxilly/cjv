package cli

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for runWhich — shows the full path to a tool binary.

func TestRunWhich_FindsTool(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	t.Chdir(cwd)

	// Install a toolchain
	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))
	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))

	cmd := &cobra.Command{}
	err := runWhich(cmd, []string{"cjc"})
	assert.NoError(t, err)
}

func TestRunWhich_NoActiveToolchain(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("CJV_HOME", home)
	t.Setenv("CJV_TOOLCHAIN", "")

	t.Chdir(cwd)

	settings := config.DefaultSettings()
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	cmd := &cobra.Command{}
	err := runWhich(cmd, []string{"cjc"})
	assert.Error(t, err, "should error when no toolchain is active")
}
