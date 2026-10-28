package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestComponentEditsKeepBestEffortRemovalAndUserSource(t *testing.T) {
	for _, identity := range []string{"lts-1.0.5", "local-sdk"} {
		t.Run(identity, func(t *testing.T) {
			installedToolchainHome(t, identity)
			source := t.TempDir()
			for _, child := range []string{"dynamic", "static"} {
				require.NoError(t, os.MkdirAll(filepath.Join(source, child), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(source, child, "payload"), []byte("user"), 0o644))
			}
			recorder := &testutil.ProgressRecorder{}
			opts := Options{Progress: recorder}
			linked, err := LinkComponent(t.Context(), identity, component.Stdx, source, false, opts)
			require.NoError(t, err)
			require.Equal(t, source, linked)
			roots, err := component.RootsFor(identity)
			require.NoError(t, err)
			require.FileExists(t, filepath.Join(roots.StdxDir, "dynamic", "payload"))
			result, err := RemoveComponents(t.Context(), identity, []component.Name{component.Docs, component.Stdx, component.StdxDocs}, opts)
			var missing *cjverr.ComponentNotInstalledError
			require.ErrorAs(t, err, &missing)
			require.NotNil(t, result)
			require.Equal(t, []component.Name{component.Stdx}, result.Removed)
			require.False(t, component.IsInstalled(roots.TcDir, component.Stdx))
			require.FileExists(t, filepath.Join(source, "dynamic", "payload"))
			require.FileExists(t, filepath.Join(source, "static", "payload"))
			require.Equal(t, []progress.Kind{progress.InstallingComponent, progress.RemovingComponent}, recorder.Kinds)
		})
	}
}
