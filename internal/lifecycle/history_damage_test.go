package lifecycle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
)

func writeHistoryMetadata(t *testing.T, home string, entry HistoryEntry) string {
	t.Helper()
	dir := filepath.Join(home, config.HistorySubdir, entry.Toolchain, entry.ID)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	data, err := json.Marshal(entry)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, historyManifestName), data, 0o644))
	return dir
}

func historyMetadataAt(when time.Time) HistoryEntry {
	return HistoryEntry{
		FormatVersion: historyFormatVersion, ID: when.UTC().Format(historyTimeFormat) + "-aabbccddeeff",
		Toolchain: "sts", HostTuple: "win32-x64", CreatedAt: when,
		Members: []HistoryMember{{Name: "sts", Release: "sts-1.0.0", Tuple: "win32-x64", Roots: []HistoryRoot{{Kind: "sdk", SHA256: strings.Repeat("a", 64)}}}},
	}
}

func TestHistoryDamagedMetadataDoesNotHideOrBlockPruningHealthySnapshots(t *testing.T) {
	home := updateHome(t)
	start := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	var paths []string
	for i := range 3 {
		paths = append(paths, writeHistoryMetadata(t, home, historyMetadataAt(start.Add(time.Duration(i)*time.Second))))
	}
	bad := historyMetadataAt(start.Add(3 * time.Second))
	badDir := writeHistoryMetadata(t, home, bad)
	require.NoError(t, os.WriteFile(filepath.Join(badDir, historyManifestName), []byte("{"), 0o644))
	report, err := InspectHistory(t.Context(), "sts")
	require.NoError(t, err)
	require.Len(t, report.Snapshots, 3)
	require.Len(t, report.Invalid, 1)
	require.Equal(t, bad.ID, report.Invalid[0].ID)
	require.Equal(t, "sts", report.Invalid[0].Toolchain)
	require.Contains(t, report.Invalid[0].Error, "unexpected EOF")
	entries, err := ListHistory(t.Context(), "sts")
	require.NoError(t, err)
	require.Equal(t, report.Snapshots, entries)

	removed, err := PruneHistory(t.Context(), "sts", 1)
	require.NoError(t, err)
	require.Equal(t, 2, removed)
	require.NoDirExists(t, paths[0])
	require.NoDirExists(t, paths[1])
	require.DirExists(t, paths[2])
	require.DirExists(t, badDir, "nonzero retention preserves damaged data for inspection")
	removed, err = PruneHistory(t.Context(), "sts", 0)
	require.NoError(t, err)
	require.Equal(t, 2, removed, "explicit zero retention also removes malformed owned snapshots")
	report, err = InspectHistory(t.Context(), "sts")
	require.NoError(t, err)
	require.Empty(t, report.Snapshots)
	require.Empty(t, report.Invalid)
}

func TestHistoryDamagedSnapshotFileCanBePrunedWithoutTouchingUnknownEntries(t *testing.T) {
	home := updateHome(t)
	entry := historyMetadataAt(time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC))
	owner := filepath.Join(home, config.HistorySubdir, "sts")
	require.NoError(t, os.MkdirAll(owner, 0o755))
	badFile := filepath.Join(owner, entry.ID)
	require.NoError(t, os.WriteFile(badFile, []byte("not a snapshot directory"), 0o644))
	unknown := filepath.Join(owner, "my-backup")
	require.NoError(t, os.WriteFile(unknown, []byte("unowned"), 0o644))
	report, err := InspectHistory(t.Context(), "sts")
	require.NoError(t, err)
	require.Len(t, report.Invalid, 1)
	removed, err := PruneHistory(t.Context(), "sts", 0)
	require.NoError(t, err)
	require.Equal(t, 1, removed)
	require.NoFileExists(t, badFile)
	require.FileExists(t, unknown)
}

func TestHistoryRollbackDoesNotSilentlySkipNewestDamagedSnapshot(t *testing.T) {
	home := updateHome(t)
	t.Setenv(config.EnvNoPathSetup, "1")
	tuple, err := target.CurrentHostTuple("")
	require.NoError(t, err)
	sdk := filepath.Join(home, config.ToolchainsSubdir, "sts")
	require.NoError(t, os.MkdirAll(filepath.Join(sdk, "bin"), 0o755))
	for _, name := range []string{"cjc", "cjc.exe"} {
		require.NoError(t, os.WriteFile(filepath.Join(sdk, "bin", name), []byte("old compiler"), 0o755))
	}
	old := toolchain.Installation{Release: "sts-1.0.0", Tuple: tuple}
	require.NoError(t, toolchain.WriteInstallation(sdk, old))
	snapshot, err := prepareHistorySnapshot(t.Context(), home, "sts", tuple, []*groupMember{{identity: "sts", installed: installationSnapshot{record: &old}}})
	require.NoError(t, err)
	require.NoError(t, os.Rename(snapshot.staging, snapshot.path))
	current := toolchain.Installation{Release: "sts-2.0.0", Tuple: tuple}
	require.NoError(t, toolchain.WriteInstallation(sdk, current))
	bad := historyMetadataAt(snapshot.entry.CreatedAt.Add(time.Second))
	badDir := writeHistoryMetadata(t, home, bad)
	require.NoError(t, os.WriteFile(filepath.Join(badDir, historyManifestName), []byte("{"), 0o644))

	_, err = RollbackHistory(t.Context(), "sts", "", Options{})
	require.ErrorContains(t, err, "latest history snapshot "+bad.ID+" is invalid")
	require.ErrorContains(t, err, "select a valid snapshot ID explicitly")
	require.Equal(t, current, readRelease(t, home, "sts"))
	_, err = RollbackHistory(t.Context(), "sts", bad.ID, Options{})
	require.ErrorContains(t, err, "unexpected EOF")
	require.Equal(t, current, readRelease(t, home, "sts"))
	result, err := RollbackHistory(t.Context(), "sts", snapshot.entry.ID, Options{})
	require.NoError(t, err, "an unrelated bad record must not block an explicitly selected verified snapshot")
	require.Equal(t, snapshot.entry.ID, result.Snapshot)
	require.Equal(t, old, readRelease(t, home, "sts"))
}
