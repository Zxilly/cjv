package lifecycle_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func prepareInstallTest(t *testing.T) (string, *config.SettingsFile, string) {
	t.Helper()
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvDistServer, "")
	server := testutil.MockDistServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	require.NoError(t, sf.Save(&settings))
	return home, sf, server.URL
}

func compilerPath(dir string) string {
	name := "cjc"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(dir, "bin", name)
}

func TestInstallRestoresToolchainsAfterFinalizeFailure(t *testing.T) {
	for _, source := range []string{"manifest", "url"} {
		for _, reinstall := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/reinstall=%t", source, reinstall), func(t *testing.T) {
				home, sf, serverURL := prepareInstallTest(t)
				name := "lts-1.0.5"
				if source == "url" {
					name = "custom-sdk"
				}
				tcRoot := filepath.Join(home, "toolchains")
				dest := filepath.Join(tcRoot, name)
				compiler := compilerPath(dest)
				oldMarker := filepath.Join(dest, "old-only.txt")
				if reinstall {
					require.NoError(t, os.MkdirAll(filepath.Dir(compiler), 0o755))
					require.NoError(t, os.WriteFile(compiler, []byte("previous compiler"), 0o755))
					require.NoError(t, os.WriteFile(oldMarker, []byte("previous SDK file"), 0o644))
				}
				beforeSettings, err := os.ReadFile(sf.Path())
				require.NoError(t, err)

				finalizeErr := errors.New("proxy refresh failed")
				managedCalled, proxyCalled, pathCalled := false, false, false
				opts := lifecycle.Options{
					EnsureManagedBinary: func() (string, error) {
						managedCalled = true
						return filepath.Join(home, "bin", "cjv"), nil
					},
					CreateProxyLinks: func() error {
						proxyCalled = true
						// The real archive must have been downloaded, validated and
						// placed at its final path before finalization fails.
						data, readErr := os.ReadFile(compiler)
						require.NoError(t, readErr)
						assert.Contains(t, string(data), "cjc 1.0.5")
						assert.NoFileExists(t, oldMarker)
						return finalizeErr
					},
					EnsurePathConfigured: func() { pathCalled = true },
				}
				if source == "url" {
					err = lifecycle.InstallToolchainFromURL(t.Context(), name,
						serverURL+"/download/cangjie-sdk-1.0.5.zip", "", reinstall, true, opts)
				} else {
					err = lifecycle.InstallToolchainWithExtras(t.Context(), "lts", nil, nil, reinstall, opts)
				}

				require.ErrorIs(t, err, finalizeErr)
				assert.True(t, managedCalled)
				assert.True(t, proxyCalled)
				assert.False(t, pathCalled)
				if reinstall {
					data, readErr := os.ReadFile(compiler)
					require.NoError(t, readErr)
					assert.Equal(t, "previous compiler", string(data))
					data, readErr = os.ReadFile(oldMarker)
					require.NoError(t, readErr)
					assert.Equal(t, "previous SDK file", string(data))
					assert.NoDirExists(t, filepath.Join(dest, "tools"))
				} else {
					assert.NoDirExists(t, dest)
				}
				entries, readErr := os.ReadDir(tcRoot)
				require.NoError(t, readErr)
				if reinstall {
					require.Len(t, entries, 1)
					assert.Equal(t, name, entries[0].Name())
				} else {
					assert.Empty(t, entries)
				}
				afterSettings, readErr := os.ReadFile(sf.Path())
				require.NoError(t, readErr)
				assert.Equal(t, beforeSettings, afterSettings)
			})
		}
	}
}

func TestInstallPreservesFinalizeAndRollbackErrors(t *testing.T) {
	home, _, _ := prepareInstallTest(t)
	dest := filepath.Join(home, "toolchains", "lts-1.0.5")
	staging := dest + toolchain.StagingSuffix
	finalizeErr := errors.New("proxy refresh failed")
	err := lifecycle.InstallToolchainWithExtras(t.Context(), "lts", nil, nil, false, lifecycle.Options{
		CreateProxyLinks: func() error {
			require.FileExists(t, compilerPath(dest))
			// Simulate a conflicting filesystem entry appearing while the
			// finalization callback runs. The installed directory cannot be
			// renamed back to staging, so rollback must report its own error.
			require.NoError(t, os.MkdirAll(staging, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(staging, "occupied"), []byte("occupied"), 0o644))
			return finalizeErr
		},
	})

	require.ErrorIs(t, err, finalizeErr)
	assert.ErrorContains(t, err, "rollback after failed install also failed")
	var renameErr *os.LinkError
	require.ErrorAs(t, err, &renameErr)
	assert.Equal(t, dest, renameErr.Old)
	assert.Equal(t, staging, renameErr.New)
	assert.FileExists(t, compilerPath(dest))
	assert.NoDirExists(t, staging)
}
