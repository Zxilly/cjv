package cli

import (
	"context"
	"fmt"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateExistingLatestAcrossRenderModes(t *testing.T) {
	for _, jsonMode := range []bool{false, true} {
		t.Run(fmt.Sprint(jsonMode), func(t *testing.T) {
			app := newApplication("dev", "")
			app.output.SetJSONMode(jsonMode)
			home := t.TempDir()
			config.IsolateForTest(t, home)
			t.Setenv(config.EnvNoPathSetup, "1")
			require.NoError(t, config.EnsureDirs())
			for _, name := range []string{"lts-1.0.0", "lts-1.0.5"} {
				require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", name), 0755))
			}
			server := validMockServer(t)
			settings := config.DefaultSettings()
			settings.ManifestURL = server.URL + "/sdk-versions.json"
			settings.DefaultToolchain = "lts-1.0.0"
			settings.Overrides[filepath.Join(home, "project")] = "lts-1.0.0"
			settingsPath := filepath.Join(home, ".cjv", "settings.toml")
			require.NoError(t, config.SaveSettings(&settings, settingsPath))
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())
			err := app.runUpdate(cmd, nil)
			loaded, loadErr := config.LoadSettings(settingsPath)
			require.NoError(t, loadErr)
			_, oldErr := os.Stat(filepath.Join(home, "toolchains", "lts-1.0.0"))
			t.Logf("json=%v err=%v default=%s oldExists=%v", jsonMode, err, loaded.DefaultToolchain, oldErr == nil)
			require.NoError(t, err, "render mode must not decide whether update succeeds")
			require.Equal(t, "lts-1.0.5", loaded.DefaultToolchain)
			require.True(t, os.IsNotExist(oldErr))
		})
	}
}
