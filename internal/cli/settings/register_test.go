package settings

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterCommandsAddsSettingsCommands(t *testing.T) {
	root := &cobra.Command{Use: "cjv"}

	RegisterCommands(root)

	var names []string
	for _, cmd := range root.Commands() {
		names = append(names, cmd.Name())
	}
	assert.Contains(t, names, "default")
	assert.Contains(t, names, "override")
	assert.Contains(t, names, "set")
}

func TestRegisteredCommandsKeepFlagStatePerRoot(t *testing.T) {
	config.IsolateForTest(t, t.TempDir())
	explicitDir := t.TempDir()
	currentDir := t.TempDir()
	t.Chdir(currentDir)
	first := &cobra.Command{Use: "cjv", SilenceErrors: true, SilenceUsage: true}
	second := &cobra.Command{Use: "cjv", SilenceErrors: true, SilenceUsage: true}
	first.SetOut(io.Discard)
	second.SetOut(io.Discard)
	RegisterCommands(first)
	RegisterCommands(second)
	first.SetArgs([]string{"override", "set", "lts", "--path", explicitDir})
	second.SetArgs([]string{"override", "set", "sts"})
	require.NoError(t, first.Execute())
	require.NoError(t, second.Execute())
	_, settings, err := config.LoadDefaultSettings()
	require.NoError(t, err)
	assert.Equal(t, "lts", settings.Overrides[config.NormalizePath(explicitDir)])
	assert.Equal(t, "sts", settings.Overrides[config.NormalizePath(currentDir)])
}

func TestRegisteredUnsetCommandsKeepModesPerRoot(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	existingDir := t.TempDir()
	sf, settings, err := config.LoadDefaultSettings()
	require.NoError(t, err)
	settings.Overrides[config.NormalizePath(existingDir)] = "lts"
	settings.Overrides[config.NormalizePath(filepath.Join(home, "missing"))] = "sts"
	require.NoError(t, sf.Save(settings))
	first := &cobra.Command{Use: "cjv", SilenceErrors: true, SilenceUsage: true}
	second := &cobra.Command{Use: "cjv", SilenceErrors: true, SilenceUsage: true}
	first.SetOut(io.Discard)
	second.SetOut(io.Discard)
	RegisterCommands(first)
	RegisterCommands(second)
	first.SetArgs([]string{"override", "unset", "--nonexistent"})
	second.SetArgs([]string{"override", "unset", "--path", existingDir})
	require.NoError(t, first.Execute())
	require.NoError(t, second.Execute())
	_, settings, err = config.LoadDefaultSettings()
	require.NoError(t, err)
	assert.Empty(t, settings.Overrides)
}

func executeSettings(t *testing.T, args ...string) error {
	t.Helper()
	root := &cobra.Command{Use: "cjv", SilenceErrors: true, SilenceUsage: true}
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	RegisterCommands(root)
	root.SetArgs(args)
	return root.Execute()
}
