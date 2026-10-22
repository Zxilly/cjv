package toolchaincmd

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/stretchr/testify/require"
)

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
