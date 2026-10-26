package runcmd

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInspectionJSONRetainsOfflineAutoInstallCause(t *testing.T) {
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
			require.ErrorIs(t, err, os.ErrNotExist)
			var missing *cjverr.ToolchainNotInstalledError
			require.ErrorAs(t, err, &missing)
			payload := decodeAutoInstallError(t, out)
			assert.Equal(t, cjverr.ErrorCodeOfflineCacheMiss, payload.Code)
			assert.Equal(t, "metadata", payload.Details["kind"])
			assert.Equal(t, manifest, payload.Details["url"])
			assert.Contains(t, missing.Error(), "not cached")
		})
	}
}

func TestWhichJSONRetainsAutoInstallFailures(t *testing.T) {
	for _, test := range []struct {
		name  string
		cause error
		code  cjverr.ErrorCode
	}{
		{"checksum", &cjverr.ChecksumMismatchError{Expected: "expected", Actual: "actual"}, cjverr.ErrorCodeChecksumMismatch},
		{"network", &net.DNSError{Err: "no such host", Name: "example.invalid"}, cjverr.ErrorCodeNetwork},
		{"canceled", context.Canceled, cjverr.ErrorCodeCanceled},
		{"deadline", context.DeadlineExceeded, cjverr.ErrorCodeDeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			home := t.TempDir()
			configtest.Isolate(t, home)
			t.Chdir(home)
			t.Setenv(config.EnvToolchain, "lts")
			t.Setenv(config.EnvAutoInstall, "1")
			previous := resolve.AutoInstallFunc
			resolve.AutoInstallFunc = func(context.Context, string, []string) error { return test.cause }
			t.Cleanup(func() { resolve.AutoInstallFunc = previous })
			out, err := execute(t, "--json", "which", "cjc")
			require.ErrorIs(t, err, test.cause)
			payload := decodeAutoInstallError(t, out)
			assert.Equal(t, test.code, payload.Code)
			assert.Equal(t, test.cause.Error(), payload.Message)
			if test.code == cjverr.ErrorCodeChecksumMismatch {
				assert.Equal(t, "expected", payload.Details["expected"])
				assert.Equal(t, "actual", payload.Details["actual"])
			}
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
