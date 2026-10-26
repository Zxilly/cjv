package toolchaincmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/stretchr/testify/require"
)

func TestHistoryReportsDamagedSnapshotsInTextAndJSON(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	const id = "20261026T000000.000000000Z-aabbccddeeff"
	dir := filepath.Join(home, "history", "sts", id)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "snapshot.json"), []byte("{"), 0o644))
	stdout, err := execute(t, "--json", "toolchain", "history", "sts")
	require.NoError(t, err)
	var result struct {
		SchemaVersion int `json:"schema_version"`
		lifecycle.HistoryReport
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Equal(t, 1, result.SchemaVersion)
	require.Empty(t, result.Snapshots)
	require.Len(t, result.Invalid, 1)
	require.Equal(t, id, result.Invalid[0].ID)
	require.Contains(t, result.Invalid[0].Error, "unexpected EOF")
	stdout, err = execute(t, "toolchain", "history", "sts")
	require.NoError(t, err)
	require.Contains(t, stdout, id+"  [invalid]")
	require.Contains(t, stdout, "history prune --keep 0")
	stdout, err = execute(t, "--json", "toolchain", "history", "prune", "sts", "--keep", "0")
	require.NoError(t, err)
	var pruned historyPruneResult
	require.NoError(t, json.Unmarshal([]byte(stdout), &pruned))
	require.Equal(t, 1, pruned.Removed)
	require.NoDirExists(t, dir)
}

func TestHistoryJSONEmptyAndValidation(t *testing.T) {
	home := filepath.Join(t.TempDir(), "absent")
	configtest.Isolate(t, home)
	for _, args := range [][]string{{"--json", "toolchain", "history"}, {"--json", "toolchain", "history", "list", "sts"}} {
		stdout, err := execute(t, args...)
		require.NoError(t, err)
		var value struct {
			SchemaVersion int               `json:"schema_version"`
			Snapshots     []json.RawMessage `json:"snapshots"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &value), stdout)
		require.Equal(t, 1, value.SchemaVersion)
		require.NotNil(t, value.Snapshots)
		require.Empty(t, value.Snapshots)
		require.NoDirExists(t, home)
	}
	_, err := execute(t, "toolchain", "history", "prune", "--keep", "-1")
	require.ErrorContains(t, err, "nonnegative")
	_, err = execute(t, "toolchain", "rollback", "sts-1.0.0")
	require.ErrorContains(t, err, "tracking host")
	require.NoDirExists(t, home)
}

func TestUpdateDryRunJSONLeavesEmptyHomeAbsent(t *testing.T) {
	home := filepath.Join(t.TempDir(), "absent")
	configtest.Isolate(t, home)
	stdout, err := execute(t, "--json", "update", "--dry-run")
	require.NoError(t, err)
	var value struct {
		SchemaVersion int               `json:"schema_version"`
		DryRun        bool              `json:"dry_run"`
		Toolchains    []json.RawMessage `json:"toolchains"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &value), stdout)
	require.Equal(t, 1, value.SchemaVersion)
	require.True(t, value.DryRun)
	require.NotNil(t, value.Toolchains)
	require.Empty(t, value.Toolchains)
	require.NoDirExists(t, home)
}
