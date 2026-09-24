package lifecycle_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/fstx"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// installVia runs the placement pipeline through the acquisition named by
// source: "manifest" resolves lts from the mock server, "url" fetches the
// same archive as a CI bundle, "zip" links it from a local copy. All three
// share placeToolchain; only how the archive reaches staging differs.
func installVia(t *testing.T, source, serverURL string, force bool, opts lifecycle.Options) error {
	t.Helper()
	url := serverURL + "/download/cangjie-sdk-1.0.5.zip"
	switch source {
	case "manifest":
		return lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: "lts", Force: force}, opts)
	case "url":
		return lifecycle.InstallToolchainFromURL(t.Context(), "custom-sdk", url, "", force, true, opts)
	default:
		archive := filepath.Join(t.TempDir(), "sdk.zip")
		if _, err := os.Stat(archive); err != nil {
			resp, err := http.Get(url) //nolint:noctx // test helper
			require.NoError(t, err)
			data, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			require.NoError(t, os.WriteFile(archive, data, 0o644))
		}
		return lifecycle.InstallToolchainFromZip(t.Context(), "custom-sdk", archive, "", force, true, opts)
	}
}

func installedNameFor(source string) string {
	if source == "manifest" {
		return "lts-1.0.5"
	}
	return "custom-sdk"
}

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
	for _, source := range []string{"manifest", "url", "zip"} {
		for _, reinstall := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/reinstall=%t", source, reinstall), func(t *testing.T) {
				home, sf, serverURL := prepareInstallTest(t)
				name := installedNameFor(source)
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
				finalizeCalled, publishCalled := false, false
				lifecycle.SetAfterFinalizeHook(t, func() error {
					finalizeCalled = true
					// The real archive must have been downloaded, validated and
					// placed at its final path before finalization fails.
					data, readErr := os.ReadFile(compiler)
					require.NoError(t, readErr)
					assert.Contains(t, string(data), "cjc 1.0.5")
					assert.NoFileExists(t, oldMarker)
					// Finalization itself has already established the managed
					// binary and the proxy links by the time the hook runs.
					assert.FileExists(t, filepath.Join(home, "bin", sdktools.CjvBinaryName()))
					assert.FileExists(t, filepath.Join(home, "bin", sdktools.PlatformBinaryName("cjc")))
					return finalizeErr
				})
				// PATH is configured only as part of publishing the default,
				// which a failed finalization must never reach.
				lifecycle.SetAfterPublishHook(t, func() error {
					publishCalled = true
					return nil
				})
				opts := lifecycle.Options{ConfigurePath: true}
				err = installVia(t, source, serverURL, reinstall, opts)

				require.ErrorIs(t, err, finalizeErr)
				assert.True(t, finalizeCalled)
				assert.False(t, publishCalled)
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
	staging := config.StagingDir(dest)
	finalizeErr := errors.New("proxy refresh failed")
	lifecycle.SetAfterFinalizeHook(t, func() error {
		require.FileExists(t, compilerPath(dest))
		// Simulate a conflicting filesystem entry appearing while the
		// finalization callback runs. The installed directory cannot be
		// renamed back to staging, so rollback must report its own error.
		require.NoError(t, os.MkdirAll(staging, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(staging, "occupied"), []byte("occupied"), 0o644))
		return finalizeErr
	})
	err := lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: "lts"}, lifecycle.Options{})

	require.ErrorIs(t, err, finalizeErr)
	assert.ErrorContains(t, err, "rollback after failed install also failed")
	var renameErr *os.LinkError
	require.ErrorAs(t, err, &renameErr)
	assert.Equal(t, dest, renameErr.Old)
	assert.Equal(t, staging, renameErr.New)
	assert.FileExists(t, compilerPath(dest))
	assert.DirExists(t, staging, "blocked recovery must retain all transaction paths")
}

func TestForceInstallRetainsOldSDKUntilBlockedRecoveryCanFinish(t *testing.T) {
	for _, source := range []string{"manifest", "url", "zip"} {
		t.Run(source, func(t *testing.T) {
			home, _, serverURL := prepareInstallTest(t)
			name := installedNameFor(source)
			tcRoot := filepath.Join(home, "toolchains")
			dest := filepath.Join(tcRoot, name)
			staging := config.StagingDir(dest)
			require.NoError(t, os.MkdirAll(filepath.Dir(compilerPath(dest)), 0o755))
			require.NoError(t, os.WriteFile(compilerPath(dest), []byte("old compiler"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(dest, "old-sdk-only"), []byte("old SDK"), 0o644))
			finalizeErr := errors.New("finalization failed while staging is occupied")
			install := func(opts lifecycle.Options) error {
				return installVia(t, source, serverURL, true, opts)
			}
			lifecycle.SetAfterFinalizeHook(t, func() error {
				require.NoError(t, os.MkdirAll(staging, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(staging, "occupied"), []byte("obstruction"), 0o644))
				return finalizeErr
			})
			err := install(lifecycle.Options{})
			require.ErrorIs(t, err, finalizeErr)
			lifecycle.SetAfterFinalizeHook(t, nil)
			var recoveryErr *fstx.RecoveryError
			require.ErrorAs(t, err, &recoveryErr)
			assert.ErrorContains(t, err, recoveryErr.Directory)
			backup := filepath.Join(recoveryErr.Directory, "0-"+name, "old-sdk-only")
			assert.FileExists(t, backup)

			// Startup and another attempted force-install must keep the old
			// SDK and the obstructing path until recovery is possible.
			require.ErrorAs(t, toolchain.RecoverHome(), &recoveryErr)
			assert.FileExists(t, backup)
			assert.FileExists(t, filepath.Join(staging, "occupied"))
			err = install(lifecycle.Options{})
			require.ErrorAs(t, err, &recoveryErr)
			assert.FileExists(t, backup)

			// Once the obstruction is removed, startup resumes the saved undo
			// sequence, restores the old SDK and cleans the discarded new SDK.
			require.NoError(t, os.Remove(filepath.Join(staging, "occupied")))
			require.NoError(t, os.Remove(staging))
			require.NoError(t, toolchain.RecoverHome())
			data, err := os.ReadFile(compilerPath(dest))
			require.NoError(t, err)
			assert.Equal(t, "old compiler", string(data))
			assert.FileExists(t, filepath.Join(dest, "old-sdk-only"))
			assert.NoDirExists(t, staging)
			assert.NoDirExists(t, recoveryErr.Directory)
		})
	}
}

func TestFirstInstallDefaultSurvivesInterruptionAfterPublication(t *testing.T) {
	if os.Getenv("CJV_TEST_INTERRUPT_DEFAULT_PUBLICATION") == "1" {
		config.IsolateForTest(t, os.Getenv(config.EnvHome))
		// Stop after the real settings update and before the transaction can
		// write its final marker or run deferred rollback/cleanup.
		lifecycle.SetAfterPublishHook(t, func() error {
			os.Exit(71)
			return nil
		})
		err := lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: "lts"}, lifecycle.Options{})
		t.Fatalf("expected interruption at default publication, got %v", err)
	}

	home, sf, _ := prepareInstallTest(t)
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestFirstInstallDefaultSurvivesInterruptionAfterPublication$")
	cmd.Env = append(os.Environ(), "CJV_TEST_INTERRUPT_DEFAULT_PUBLICATION=1")
	output, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, "%s", output)
	require.Equal(t, 71, exitErr.ExitCode(), "%s", output)
	sf.Invalidate()
	settings, err := sf.Load()
	require.NoError(t, err)
	require.Equal(t, "lts-1.0.5", settings.DefaultToolchain)
	dest := filepath.Join(home, "toolchains", settings.DefaultToolchain)
	assert.FileExists(t, compilerPath(dest))
	journals, err := filepath.Glob(filepath.Join(home, "toolchains", ".fstx-*", "journal.json"))
	require.NoError(t, err)
	require.Len(t, journals, 1, "interruption must precede normal transaction cleanup")

	require.NoError(t, toolchain.RecoverHome())

	assert.FileExists(t, compilerPath(dest), "startup must retain the SDK referenced by the published default")
	journals, err = filepath.Glob(filepath.Join(home, "toolchains", ".fstx-*"))
	require.NoError(t, err)
	assert.Empty(t, journals)
}

func TestFirstInstallSettingsWriteFailureRollsBackPreparedSDK(t *testing.T) {
	home, sf, _ := prepareInstallTest(t)
	before, err := os.ReadFile(sf.Path())
	require.NoError(t, err)
	backup := sf.Path() + ".test-backup"
	publishCalled := false
	lifecycle.SetAfterPublishHook(t, func() error {
		publishCalled = true
		return nil
	})
	lifecycle.SetAfterFinalizeHook(t, func() error {
		// Fail the actual atomic settings write after SDK placement and
		// before any default reference can be published.
		require.NoError(t, os.Rename(sf.Path(), backup))
		require.NoError(t, os.Mkdir(sf.Path(), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(sf.Path(), "occupied"), []byte("block publication"), 0o644))
		return nil
	})
	err = lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: "lts"}, lifecycle.Options{ConfigurePath: true})
	require.Error(t, err)
	assert.False(t, publishCalled)
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "lts-1.0.5"))
	entries, err := os.ReadDir(filepath.Join(home, "toolchains"))
	require.NoError(t, err)
	assert.Empty(t, entries, "a synchronous publication failure must undo the prepared SDK")
	require.NoError(t, os.Remove(filepath.Join(sf.Path(), "occupied")))
	require.NoError(t, os.Remove(sf.Path()))
	require.NoError(t, os.Rename(backup, sf.Path()))
	after, err := os.ReadFile(sf.Path())
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestCustomToolchainNamesRemainManageable(t *testing.T) {
	names := []string{".local-sdk"}
	if runtime.GOOS != "windows" {
		names = append(names, "sdk:debug")
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			home, _, serverURL := prepareInstallTest(t)
			url := serverURL + "/download/cangjie-sdk-1.0.5.zip"
			require.NoError(t, lifecycle.InstallToolchainFromURL(t.Context(), name, url, "", false, true, lifecycle.Options{}))
			dest := filepath.Join(home, "toolchains", name)
			assert.FileExists(t, compilerPath(dest))
			marker := filepath.Join(dest, "old-only")
			require.NoError(t, os.WriteFile(marker, []byte("old SDK"), 0o644))
			require.NoError(t, lifecycle.InstallToolchainFromURL(t.Context(), name, url, "", true, true, lifecycle.Options{}))
			assert.FileExists(t, compilerPath(dest))
			assert.NoFileExists(t, marker)
			require.NoError(t, lifecycle.RemoveToolchain(name))
			assert.NoDirExists(t, dest)
			entries, err := os.ReadDir(filepath.Join(home, "toolchains"))
			require.NoError(t, err)
			assert.Empty(t, entries)
		})
	}
}
