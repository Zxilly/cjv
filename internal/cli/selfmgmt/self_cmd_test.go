package selfmgmt

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSelfCommandWiresSubcommandsAndUpdate(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)
	clean := &cobra.Command{Use: "clean-cache"}

	cmd := NewSelfCommand("dev", "", clean)

	assert.NotNil(t, cmd)
	assert.NotNil(t, findSubcommand(cmd, "update"))
	assert.NotNil(t, findSubcommand(cmd, "uninstall"))
	assert.NotNil(t, findSubcommand(cmd, "clean-cache"))

	update := findSubcommand(cmd, "update")
	require.NoError(t, update.RunE(update, nil))
	assert.FileExists(t, filepath.Join(home, "bin", proxy.CjvBinaryName()))
	assert.FileExists(t, filepath.Join(home, "bin", proxy.PlatformBinaryName("cjc")))
}

func TestSelfUninstallDoesNotCleanPathWhenRemoveHomeFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	oldYes := uninstallYes
	oldEnsure := ensureSelfManagedExecutable
	oldRemove := removeSelfHomeDir
	oldCleanup := cleanupSelfPathEntries
	ensureSelfManagedExecutable = func() (string, error) {
		return filepath.Join(home, "bin", proxy.CjvBinaryName()), nil
	}
	removeSelfHomeDir = func(home, managedExe string) error {
		return errors.New("remove failed")
	}
	cleanupCalled := false
	cleanupSelfPathEntries = func() {
		cleanupCalled = true
	}
	t.Cleanup(func() {
		uninstallYes = oldYes
		ensureSelfManagedExecutable = oldEnsure
		removeSelfHomeDir = oldRemove
		cleanupSelfPathEntries = oldCleanup
	})

	cmd := NewSelfCommand("dev", "", nil)
	uninstallYes = true
	uninstall := findSubcommand(cmd, "uninstall")
	require.Error(t, uninstall.RunE(uninstall, nil))
	assert.False(t, cleanupCalled)
}

func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, child := range cmd.Commands() {
		if child.Name() == name {
			return child
		}
	}
	return nil
}
