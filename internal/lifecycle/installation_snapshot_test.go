package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/stretchr/testify/require"
)

func TestInstallationSnapshotRejectsChangedLinkedSource(t *testing.T) {
	for _, operation := range []string{"component", "group"} {
		t.Run(operation, func(t *testing.T) {
			newHistoryUpdateFixture(t)
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, Options{}))
			sources := []string{t.TempDir(), t.TempDir()}
			for _, source := range sources {
				for _, child := range []string{"dynamic", "static"} {
					require.NoError(t, os.MkdirAll(filepath.Join(source, child), 0o755))
				}
			}
			_, err := LinkComponent(t.Context(), "sts", component.Stdx, sources[0], false, Options{})
			require.NoError(t, err)
			changed := false
			opts := Options{Progress: componentInstallSink(func(e progress.Event) {
				if e.Kind == progress.FetchingComponent && !changed {
					changed = true
					// Linking changes the source while keeping the same manifest paths.
					_, err := LinkComponent(t.Context(), "sts", component.Stdx, sources[1], true, Options{})
					require.NoError(t, err)
				}
			})}
			if operation == "component" {
				err = InstallComponents(t.Context(), "sts", []string{"docs"}, false, opts)
			} else {
				err = Install(t.Context(), InstallRequest{Toolchain: "sts", Components: []string{"docs"}}, opts)
			}
			require.True(t, changed)
			require.ErrorContains(t, err, "changed during")
			roots, err := component.RootsFor("sts")
			require.NoError(t, err)
			require.False(t, component.IsInstalled(roots.TcDir, component.Docs))
			intents, err := component.InstalledIntents(roots)
			require.NoError(t, err)
			require.Equal(t, []component.Intent{{Name: component.Stdx, Source: sources[1]}}, intents)
		})
	}
}
