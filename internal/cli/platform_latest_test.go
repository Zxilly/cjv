package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCheckUsesLatestVersionAvailableForInstalledTarget(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.0-linux-x64-ohos"), 0o755))

	server := testutil.ManifestOnlyServer(t, testutil.ManifestWithPlatformGap())
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	app.output.SetJSONMode(true)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(&buf)
	require.NoError(t, app.runCheck(cmd, nil))

	var got checkResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Toolchains, 1)
	assert.Equal(t, "lts-1.5.0-linux-x64-ohos", got.Toolchains[0].Latest)
	assert.True(t, got.Toolchains[0].UpdateAvailable)
	assert.False(t, got.Toolchains[0].NotForTarget)
}
