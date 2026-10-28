package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/progress"
	"github.com/stretchr/testify/require"
)

func TestUpdateAllDoesNotRecreateRemovedInstallation(t *testing.T) {
	for _, record := range []bool{true, false} {
		name := "with-record"
		if !record {
			name = "without-record"
		}
		t.Run(name, func(t *testing.T) {
			f := newHistoryUpdateFixture(t)
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, Options{}))
			dir := filepath.Join(f.home, "toolchains", "sts")
			if !record {
				require.NoError(t, os.Remove(filepath.Join(dir, ".cjv", "toolchain.toml")))
			}
			removed := false
			report, err := UpdateAll(t.Context(), Options{Progress: componentInstallSink(func(e progress.Event) {
				if e.Kind == progress.FetchingManifest && !removed {
					// Simulate removal after enumeration while resolution is unlocked.
					removed = true
					require.NoError(t, os.RemoveAll(dir))
				}
			})})
			require.True(t, removed)
			require.Error(t, err)
			require.Len(t, report.Outcomes, 1)
			require.Equal(t, "sts", report.Outcomes[0].Name)
			require.Equal(t, UpdateFailed, report.Outcomes[0].Status)
			require.ErrorIs(t, err, report.Outcomes[0].Err)
			require.NoDirExists(t, dir)
		})
	}
}

func TestUpdateAllReportsTargetHostSelectionFailure(t *testing.T) {
	f := newHistoryUpdateFixture(t)
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, Options{}))
	require.NoError(t, os.WriteFile(filepath.Join(f.home, "toolchains", "sts", ".cjv", "toolchain.toml"), []byte("invalid = ["), 0o644))
	report, err := UpdateAll(t.Context(), Options{})
	require.Error(t, err)
	require.Len(t, report.Outcomes, 2)
	for _, outcome := range report.Outcomes {
		require.Contains(t, []string{"sts", "sts-" + f.host + "-ohos"}, outcome.Name)
		require.Equal(t, UpdateFailed, outcome.Status)
		require.ErrorIs(t, err, outcome.Err)
	}
}

func TestUpdateAllReportsFinalInstallationReadFailure(t *testing.T) {
	f := newHistoryUpdateFixture(t)
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, Options{}))
	removed := false
	report, err := UpdateAll(t.Context(), Options{Progress: componentInstallSink(func(e progress.Event) {
		if e.Kind == progress.AlreadyUpToDate {
			removed = true
			require.NoError(t, os.Remove(filepath.Join(f.home, "toolchains", "sts", ".cjv", "toolchain.toml")))
		}
	})})
	require.True(t, removed)
	require.Error(t, err)
	require.Len(t, report.Outcomes, 1)
	require.Equal(t, "sts", report.Outcomes[0].Name)
	require.Equal(t, "sts-1.0.0", report.Outcomes[0].Replacement)
	require.Equal(t, UpdateFailed, report.Outcomes[0].Status)
	require.ErrorIs(t, err, report.Outcomes[0].Err)
}
