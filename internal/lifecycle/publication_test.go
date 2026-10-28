package lifecycle

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
)

func TestPublicationChangesOnlySelectedRootsAndReportsRetainedCommit(t *testing.T) {
	for _, retained := range []bool{false, true} {
		t.Run(map[bool]string{false: "committed", true: "settings-restore-failed"}[retained], func(t *testing.T) {
			installedToolchainHome(t, "local-sdk")
			home, err := config.Home()
			require.NoError(t, err)
			roots, err := component.RootsFor("local-sdk")
			require.NoError(t, err)
			payload := filepath.Join(roots.TcDir, "sdk-payload")
			require.NoError(t, os.WriteFile(payload, []byte("unchanged SDK"), 0o644))
			before, err := os.Stat(payload)
			require.NoError(t, err)
			prepared := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(prepared, "index.html"), []byte("docs"), 0o644))
			lock, err := toolchain.LockHome(t.Context())
			require.NoError(t, err)
			defer lock.Close()
			require.NoError(t, lock.Recover())
			failure := errors.New("settings restoration failed")
			publish := func() error {
				if retained {
					return &publishedSettingsError{failure}
				}
				return nil
			}
			var installed []string
			opts := Options{Progress: componentInstallSink(func(e progress.Event) {
				if e.Kind == progress.ToolchainInstalled {
					installed = append(installed, e.Toolchain)
				}
			})}
			err = publishPrepared(t.Context(), home, []toolchainPublication{{
				identity: "local-sdk", roots: []rootChange{{prepared, roots.DocsDir}},
			}}, publish, opts)
			if retained {
				require.ErrorIs(t, err, failure)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, []string{"local-sdk"}, installed)
			require.FileExists(t, filepath.Join(roots.DocsDir, "index.html"))
			after, err := os.Stat(payload)
			require.NoError(t, err)
			require.True(t, os.SameFile(before, after), "publishing one root must preserve the SDK")
		})
	}
}

func TestFirstPlacementCannotReplaceAnExistingDestination(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "local-sdk")
	stage := config.StagingDir(dest)
	for _, dir := range []string{dest, stage} {
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dest, "existing"), []byte("keep"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(stage, "replacement"), []byte("new"), 0o644))
	finalized := false
	err := swapInstalledToolchain(stage, dest, false, func() error { finalized = true; return nil }, nil)
	require.Error(t, err)
	require.False(t, finalized)
	require.FileExists(t, filepath.Join(dest, "existing"))
	require.NoFileExists(t, filepath.Join(dest, "replacement"))
	require.FileExists(t, filepath.Join(stage, "replacement"))
}
