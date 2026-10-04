package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreparedComponentBatchPreservesLinkedUserDataAndRollsBackPublicationFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			installedToolchainHome(t, "lts-1.0.5")
			roots, err := component.RootsFor("lts-1.0.5")
			require.NoError(t, err)
			user := t.TempDir()
			for _, sub := range []string{"dynamic", "static"} {
				require.NoError(t, os.Mkdir(filepath.Join(user, sub), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(user, sub, "old"), []byte("user"), 0o644))
			}
			_, err = component.Link(roots, component.Stdx, user, false)
			require.NoError(t, err)
			if fail {
				require.NoError(t, os.MkdirAll(roots.DocsDir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(roots.DocsDir, "main"), []byte("obstruction"), 0o644))
			}
			componentServer(t, map[string]string{"index.html": "docs"},
				map[string]string{"top/dynamic/new": "new", "top/static/new": "new"}, []string{hostStdxPlatform(t)})
			err = InstallComponents(t.Context(), "lts-1.0.5", []string{"stdx", "docs"}, true, Options{})
			for _, sub := range []string{"dynamic", "static"} {
				data, err := os.ReadFile(filepath.Join(user, sub, "old"))
				require.NoError(t, err)
				assert.Equal(t, "user", string(data))
				assert.NoFileExists(t, filepath.Join(user, sub, "new"))
			}
			intents, readErr := component.InstalledIntents(roots)
			require.NoError(t, readErr)
			if fail {
				require.Error(t, err)
				require.Len(t, intents, 1)
				assert.Equal(t, user, intents[0].Source)
				assert.FileExists(t, filepath.Join(roots.StdxDir, "dynamic", "old"))
				assert.NoFileExists(t, filepath.Join(roots.StdxDir, "dynamic", "new"))
				assert.False(t, component.IsInstalled(roots.TcDir, component.Docs))
			} else {
				require.NoError(t, err)
				assert.FileExists(t, filepath.Join(roots.StdxDir, "dynamic", "new"))
				assert.NoFileExists(t, filepath.Join(roots.StdxDir, "dynamic", "old"))
				for _, intent := range intents {
					assert.Empty(t, intent.Source)
				}
			}
		})
	}
}
