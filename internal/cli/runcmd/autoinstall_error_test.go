package runcmd

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInspectionJSONReportsMissingSDKWithoutOfflineInstall(t *testing.T) {
	for _, args := range [][]string{{"which", "cjc"}, {"envsetup"}} {
		t.Run(args[0], func(t *testing.T) {
			home := t.TempDir()
			configtest.Isolate(t, home)
			t.Chdir(home)
			t.Setenv(config.EnvToolchain, "lts")
			t.Setenv(config.EnvAutoInstall, "1")
			t.Setenv("CJV_OFFLINE", "1")
			sf, err := config.DefaultSettingsFile()
			require.NoError(t, err)
			manifest := "https://example.invalid/versions.json"
			_, err = sf.Update(config.SettingsUpdate{ManifestURL: &manifest})
			require.NoError(t, err)

			out, err := execute(t, append([]string{"--json"}, args...)...)
			var missing *cjverr.ToolchainNotInstalledError
			require.ErrorAs(t, err, &missing)
			payload := decodeAutoInstallError(t, out)
			assert.Empty(t, payload.Code, "a read-only query has no installer failure cause")
			assert.NotContains(t, out, string(cjverr.ErrorCodeOfflineCacheMiss))
			assert.Contains(t, err.Error(), "cjv install lts")
		})
	}
}

func TestInspectionJSONNeverInvokesInstaller(t *testing.T) {
	for _, command := range []string{"which", "envsetup"} {
		t.Run(command, func(t *testing.T) {
			home := t.TempDir()
			configtest.Isolate(t, home)
			t.Chdir(home)
			t.Setenv(config.EnvToolchain, "lts")
			t.Setenv(config.EnvAutoInstall, "1")
			previous := resolve.AutoInstallFunc
			resolve.AutoInstallFunc = func(context.Context, string, []string) error {
				t.Fatal("JSON queries must not invoke an installer")
				return nil
			}
			t.Cleanup(func() { resolve.AutoInstallFunc = previous })
			out, err := execute(t, "--json", command)
			require.Error(t, err)
			payload := decodeAutoInstallError(t, out)
			assert.Empty(t, payload.Code)
		})
	}
}

func decodeAutoInstallError(t *testing.T, out string) cjverr.ErrorInfo {
	t.Helper()
	var result struct {
		Error struct {
			Code    cjverr.ErrorCode `json:"code"`
			Details struct {
				Name  string           `json:"name"`
				Cause cjverr.ErrorInfo `json:"cause"`
			} `json:"details"`
		} `json:"error"`
	}
	decoder := json.NewDecoder(strings.NewReader(out))
	require.NoError(t, decoder.Decode(&result))
	require.True(t, errors.Is(decoder.Decode(new(any)), io.EOF), "stdout must contain one document")
	assert.Equal(t, cjverr.ErrorCodeToolchainNotInstalled, result.Error.Code)
	assert.Equal(t, "lts", result.Error.Details.Name)
	return result.Error.Details.Cause
}
