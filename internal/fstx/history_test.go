package fstx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/stretchr/testify/require"
)

func TestHistoryScopeAndRecoveryFollowGroupCommit(t *testing.T) {
	const id = "20261022T010203.123456789Z-abcdef123456"
	for _, commit := range []bool{false, true} {
		t.Run(map[bool]string{false: "rollback", true: "commit"}[commit], func(t *testing.T) {
			home := t.TempDir()
			path := filepath.Join(home, config.HistorySubdir, "sts", id)
			stage := config.StagingDir(path)
			require.NoError(t, os.MkdirAll(stage, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(stage, "payload"), []byte("saved"), 0o644))
			tx, err := NewToolchainGroupTransaction(home, []string{"sts", "sts-linux-x64-ohos"})
			require.NoError(t, err)
			require.Error(t, tx.IncludeHistory("../../toolchains"))
			require.NoError(t, tx.IncludeHistory(id))
			for _, outside := range []string{filepath.Join(home, "history", "lts", id), filepath.Join(path, "payload"), filepath.Join(home, "history", "sts", id+"-other")} {
				require.Error(t, tx.RemoveDir(outside))
			}
			require.NoError(t, tx.RenameFile(stage, path))
			if commit {
				require.NoError(t, tx.Commit())
			}
			require.NoError(t, Recover(filepath.Join(home, config.ToolchainsSubdir)))
			if commit {
				require.FileExists(t, filepath.Join(path, "payload"))
				require.NoDirExists(t, stage)
			} else {
				require.FileExists(t, filepath.Join(stage, "payload"))
				require.NoDirExists(t, path)
			}
		})
	}
}
