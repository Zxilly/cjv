package lifecycle

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
)

func TestHistoryRetainsTwoAndRollsBackWholeGroup(t *testing.T) {
	home := updateHome(t)
	selectRelease(t, "1.0.0", "ohos")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, Options{}))
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts-1.0.0"}, Options{}))
	initial := readChoices(t)
	for _, version := range []string{"2.0.0", "3.0.0", "4.0.0"} {
		selectRelease(t, version, "ohos")
		_, err := UpdateInstalled(t.Context(), parse(t, "sts"), Options{})
		require.NoError(t, err)
	}
	entries, err := ListHistory(t.Context(), "sts")
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, "sts-3.0.0", entries[0].Members[0].Release)
	require.Equal(t, "sts-2.0.0", entries[1].Members[0].Release)
	require.Len(t, entries[0].Members, 2)
	// Break the source URL: retained history is enough for rollback.
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	unreachable := "http://127.0.0.1:1/versions.json"
	_, err = sf.Update(config.SettingsUpdate{ManifestURL: &unreachable})
	require.NoError(t, err)
	result, err := RollbackHistory(t.Context(), "sts", "", Options{})
	require.NoError(t, err)
	require.Equal(t, entries[0].ID, result.Snapshot)
	require.Len(t, result.Members, 2)
	for _, member := range entries[0].Members {
		require.Equal(t, member.Release, readRelease(t, home, member.Name).Release)
	}
	require.Equal(t, "sts-1.0.0", readRelease(t, home, "sts-1.0.0").Release)
	settings := readChoices(t)
	require.Equal(t, initial.DefaultToolchain, settings.DefaultToolchain)
	require.Equal(t, initial.Overrides, settings.Overrides)
	after, err := ListHistory(t.Context(), "sts")
	require.NoError(t, err)
	require.Len(t, after, 2)
	require.Equal(t, "sts-4.0.0", after[0].Members[0].Release)
	removed, err := PruneHistory(t.Context(), "sts", 0)
	require.NoError(t, err)
	require.Equal(t, 2, removed)
	require.Equal(t, "sts-3.0.0", readRelease(t, home, "sts").Release)
}

func TestHistoryRejectsTamperedPayloadAndRollsBackFailure(t *testing.T) {
	home := updateHome(t)
	selectRelease(t, "1.0.0", "ohos")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, Options{}))
	selectRelease(t, "2.0.0", "ohos")
	_, err := UpdateInstalled(t.Context(), parse(t, "sts"), Options{})
	require.NoError(t, err)
	entries, err := ListHistory(t.Context(), "sts")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	oldHook := afterFinalizeHook
	t.Cleanup(func() { afterFinalizeHook = oldHook })
	failure := errors.New("rollback publication failed")
	afterFinalizeHook = func() error { return failure }
	_, err = RollbackHistory(t.Context(), "sts", entries[0].ID, Options{})
	require.ErrorIs(t, err, failure)
	afterFinalizeHook = oldHook
	require.Equal(t, "sts-2.0.0", readRelease(t, home, "sts").Release)
	after, err := ListHistory(t.Context(), "sts")
	require.NoError(t, err)
	require.Equal(t, entries, after)
	root := filepath.Join(home, config.HistorySubdir, "sts", entries[0].ID)
	require.NoError(t, os.WriteFile(filepath.Join(root, "sdk", "sts", "tampered"), []byte("unexpected"), 0o644))
	_, err = RollbackHistory(t.Context(), "sts", entries[0].ID, Options{})
	require.ErrorContains(t, err, "checksum mismatch")
	require.Equal(t, "sts-2.0.0", readRelease(t, home, "sts").Release)
	_, err = RollbackHistory(t.Context(), "sts", "../../toolchains/sts", Options{})
	require.ErrorContains(t, err, "invalid history snapshot")
}

func TestHistoryRecoveryCleansPreJournalCrash(t *testing.T) {
	if os.Getenv("CJV_TEST_HISTORY_STAGE_CRASH") == "1" {
		configtest.Isolate(t, os.Getenv(config.EnvHome))
		afterStagingHook = func() error { os.Exit(73); return nil }
		_ = Install(t.Context(), InstallRequest{Toolchain: "sts"}, Options{})
		os.Exit(74)
	}
	home := updateHome(t)
	selectRelease(t, "1.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, Options{}))
	selectRelease(t, "2.0.0")
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestHistoryRecoveryCleansPreJournalCrash$")
	cmd.Env = append(os.Environ(), "CJV_TEST_HISTORY_STAGE_CRASH=1")
	output, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	require.ErrorAs(t, err, &exit, "%s", output)
	require.Equal(t, 73, exit.ExitCode(), "%s", output)
	owner := filepath.Join(home, config.HistorySubdir, "sts")
	staged, err := filepath.Glob(filepath.Join(owner, "*.staging"))
	require.NoError(t, err)
	require.Len(t, staged, 1)
	require.Equal(t, "sts-1.0.0", readRelease(t, home, "sts").Release)
	require.NoError(t, toolchain.RecoverHome())
	staged, err = filepath.Glob(filepath.Join(owner, "*.staging"))
	require.NoError(t, err)
	require.Empty(t, staged)
	entries, err := ListHistory(t.Context(), "sts")
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestHistoryReadOnlyEmptyHomeAndValidation(t *testing.T) {
	home := filepath.Join(t.TempDir(), "absent")
	configtest.Isolate(t, home)
	entries, err := ListHistory(t.Context(), "")
	require.NoError(t, err)
	require.NotNil(t, entries)
	require.Empty(t, entries)
	require.NoDirExists(t, home)
	_, err = ListHistory(t.Context(), "sts-1.0.0")
	require.Error(t, err)
	_, err = PruneHistory(t.Context(), "", -1)
	require.Error(t, err)
	require.NoDirExists(t, home)
}

func TestHistoryCancellationBeforePublicationRetainsInstalledGroup(t *testing.T) {
	home := updateHome(t)
	selectRelease(t, "1.0.0", "ohos")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, Options{}))
	selectRelease(t, "2.0.0", "ohos")
	_, err := UpdateInstalled(t.Context(), parse(t, "sts"), Options{})
	require.NoError(t, err)
	entries, err := ListHistory(t.Context(), "sts")
	require.NoError(t, err)
	for _, operation := range []string{"update", "rollback"} {
		t.Run(operation, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			previous := afterStagingHook
			t.Cleanup(func() { afterStagingHook = previous })
			afterStagingHook = func() error { cancel(); return nil }
			if operation == "update" {
				selectRelease(t, "3.0.0", "ohos")
				_, err = UpdateInstalled(ctx, parse(t, "sts"), Options{})
			} else {
				_, err = RollbackHistory(ctx, "sts", entries[0].ID, Options{})
			}
			require.ErrorIs(t, err, context.Canceled)
			require.Equal(t, "sts-2.0.0", readRelease(t, home, "sts").Release)
			after, err := ListHistory(t.Context(), "sts")
			require.NoError(t, err)
			require.Equal(t, entries, after)
			staged, err := filepath.Glob(filepath.Join(home, config.HistorySubdir, "sts", "*.staging"))
			require.NoError(t, err)
			require.Empty(t, staged)
		})
	}
}

func TestHistoryRollbackPreservesAndRestoresLaterAddedTarget(t *testing.T) {
	home := updateHome(t)
	selectRelease(t, "1.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, Options{}))
	selectRelease(t, "2.0.0", "ohos")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, Options{}))
	targets, err := trackedGroupTargets(parse(t, "sts").Channel, readRelease(t, home, "sts").Tuple, true)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	entries, err := ListHistory(t.Context(), "sts")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Len(t, entries[0].Members, 1)
	result, err := RollbackHistory(t.Context(), "sts", entries[0].ID, Options{})
	require.NoError(t, err)
	require.NotEmpty(t, result.PreviousSnapshot)
	require.Equal(t, "sts-1.0.0", readRelease(t, home, "sts").Release)
	require.NoDirExists(t, filepath.Join(home, "toolchains", targets[0]))
	_, err = RollbackHistory(t.Context(), "sts", "", Options{})
	require.NoError(t, err)
	require.Equal(t, "sts-2.0.0", readRelease(t, home, "sts").Release)
	require.Equal(t, "sts-2.0.0-"+readRelease(t, home, "sts").Tuple+"-ohos", readRelease(t, home, targets[0]).Release)
}
