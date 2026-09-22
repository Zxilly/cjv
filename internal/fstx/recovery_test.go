package fstx

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeMarker(t *testing.T, dir, name string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(name), 0o644))
	return path
}

func TestRecoverInterruptedSwap(t *testing.T) {
	for _, interruptedAfter := range []string{"backup", "placement", "undo-before-journal-update", "commit-marker"} {
		t.Run(interruptedAfter, func(t *testing.T) {
			root := t.TempDir()
			dest := filepath.Join(root, "sdk")
			stage := dest + stagingSuffix
			writeMarker(t, dest, "old")
			writeMarker(t, stage, "new")
			tx, err := NewTransaction(dest)
			require.NoError(t, err)
			require.NoError(t, tx.RemoveDir(dest))
			if interruptedAfter != "backup" {
				require.NoError(t, tx.RenameFile(stage, dest))
			}
			if interruptedAfter == "undo-before-journal-update" {
				// A process can stop after a reverse rename and before its
				// journal is shortened. Recovery must recognize the undo.
				require.NoError(t, os.Rename(dest, stage))
			}
			if interruptedAfter == "commit-marker" {
				// This on-disk state is the gap between publishing commitment
				// and deleting backup files; there must be no rollback here.
				state := tx.journal
				state.State = stateCommitted
				data, err := json.Marshal(state)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(tx.tmpDir, journalName), data, 0o600))
			}

			require.NoError(t, Recover(root))
			assert.NoDirExists(t, tx.tmpDir)
			if interruptedAfter == "commit-marker" {
				assert.FileExists(t, filepath.Join(dest, "new"))
				assert.NoFileExists(t, filepath.Join(dest, "old"))
			} else {
				assert.FileExists(t, filepath.Join(dest, "old"))
				assert.FileExists(t, filepath.Join(stage, "new"))
			}
			require.NoError(t, Recover(root), "repeated startup should be harmless")
		})
	}
}

func TestRollbackCanRetryWithoutRepeatingCompletedChanges(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "sdk")
	stage := dest + stagingSuffix
	writeMarker(t, dest, "old")
	writeMarker(t, stage, "new")
	tx, err := NewTransaction(dest)
	require.NoError(t, err)
	require.NoError(t, tx.RemoveDir(dest))
	require.NoError(t, tx.RenameFile(stage, dest))
	blocker := writeMarker(t, stage, "occupied")
	require.Error(t, tx.Rollback())
	assert.FileExists(t, filepath.Join(tx.tmpDir, "0-sdk", "old"))
	require.NoError(t, os.Remove(blocker))
	require.NoError(t, os.Remove(stage))
	require.NoError(t, tx.Rollback())
	require.NoError(t, tx.Rollback())
	assert.FileExists(t, filepath.Join(dest, "old"))
	assert.FileExists(t, filepath.Join(stage, "new"))
}

func TestToolchainTransactionRecoversAllOwnedRoots(t *testing.T) {
	home := t.TempDir()
	for _, dir := range []string{"toolchains", "stdx", "docs"} {
		writeMarker(t, filepath.Join(home, dir, "sdk"), dir)
	}
	unrelated := writeMarker(t, home, "settings.toml")
	tx, err := NewToolchainTransaction(home, "sdk")
	require.NoError(t, err)
	require.Error(t, tx.RemoveFile(unrelated), "settings are outside the transaction scope")
	for _, dir := range []string{"docs", "stdx", "toolchains"} {
		require.NoError(t, tx.RemoveDir(filepath.Join(home, dir, "sdk")))
	}
	require.NoError(t, Recover(filepath.Join(home, "toolchains")))
	for _, dir := range []string{"toolchains", "stdx", "docs"} {
		assert.FileExists(t, filepath.Join(home, dir, "sdk", dir))
	}
	assert.FileExists(t, unrelated)
	assert.NoDirExists(t, tx.tmpDir)
}

func TestRecoverRejectsInvalidJournalsWithoutChangingFiles(t *testing.T) {
	for name, payload := range map[string]string{
		"truncated":                 `{`,
		"unknown-version":           `{"version":2,"target":"sdk","state":"active","changes":[]}`,
		"unknown-state":             `{"version":1,"target":"sdk","state":"invalid","changes":[]}`,
		"unknown-scope":             `{"version":1,"target":"sdk","scope":"arbitrary","state":"active","changes":[]}`,
		"outside-path":              `{"version":1,"target":"sdk","state":"active","changes":[{"kind":"add","path":"../victim"}]}`,
		"outside-backup":            `{"version":1,"target":"sdk","state":"active","changes":[{"kind":"remove","path":"sdk","backup":"../victim"}]}`,
		"unknown-operation":         `{"version":1,"target":"sdk","state":"active","changes":[{"kind":"erase","path":"sdk"}]}`,
		"invalid-earlier-operation": `{"version":1,"target":"sdk","state":"active","changes":[{"kind":"erase","path":"sdk"},{"kind":"add","path":"sdk"}]}`,
		"absolute-target":           `{"version":1,"target":"/tmp/outside","state":"active","changes":[]}`,
		"trailing-json":             `{"version":1,"target":"sdk","state":"active","changes":[]} {}`,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			victim := writeMarker(t, root, "sdk")
			txDir := filepath.Join(root, ".fstx-interrupted")
			backup := writeMarker(t, filepath.Join(txDir, "0-sdk"), "old")
			require.NoError(t, os.WriteFile(filepath.Join(txDir, journalName), []byte(payload), 0o600))
			require.Error(t, Recover(root))
			assert.FileExists(t, victim)
			assert.FileExists(t, backup)
		})
	}
}

func TestRecoveryDoesNotFollowUnownedLinks(t *testing.T) {
	for _, linkAt := range []string{"transaction", "journal", "target-parent"} {
		t.Run(linkAt, func(t *testing.T) {
			root := t.TempDir()
			outside := t.TempDir()
			victim := writeMarker(t, outside, "victim")
			state := journal{Version: 1, Target: "sdk", State: stateActive,
				Changes: []change{{Kind: "add", Path: filepath.Join("sdk", "victim")}}}
			data, err := json.Marshal(state)
			require.NoError(t, err)
			txDir := filepath.Join(root, ".fstx-interrupted")
			var from, to string
			switch linkAt {
			case "transaction":
				require.NoError(t, os.WriteFile(filepath.Join(outside, journalName), data, 0o600))
				from, to = outside, txDir
			case "journal":
				require.NoError(t, os.MkdirAll(txDir, 0o755))
				from, to = filepath.Join(outside, journalName), filepath.Join(txDir, journalName)
				require.NoError(t, os.WriteFile(from, data, 0o600))
			case "target-parent":
				require.NoError(t, os.MkdirAll(txDir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(txDir, journalName), data, 0o600))
				from, to = outside, filepath.Join(root, "sdk")
			}
			if err := os.Symlink(from, to); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			require.Error(t, Recover(root))
			assert.FileExists(t, victim)
			_, err = os.Lstat(to)
			require.NoError(t, err)
		})
	}
}

func TestTransactionOwnsLinkedSDKEntryWithoutOwningItsTarget(t *testing.T) {
	for _, commit := range []bool{false, true} {
		t.Run(map[bool]string{false: "rollback", true: "commit"}[commit], func(t *testing.T) {
			root := t.TempDir()
			outside := t.TempDir()
			marker := writeMarker(t, outside, "user-sdk")
			dest := filepath.Join(root, "sdk")
			if err := os.Symlink(outside, dest); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			tx, err := NewTransaction(dest)
			require.NoError(t, err)
			require.Error(t, tx.RemoveFile(filepath.Join(dest, "user-sdk")))
			require.NoError(t, tx.RemoveDir(dest))
			if commit {
				require.NoError(t, tx.Commit())
				_, err := os.Lstat(dest)
				require.ErrorIs(t, err, os.ErrNotExist)
			} else {
				require.NoError(t, Recover(root))
				link, err := os.Readlink(dest)
				require.NoError(t, err)
				assert.Equal(t, outside, link)
			}
			assert.FileExists(t, marker)
		})
	}
}

func TestRecoverInitialJournalDebris(t *testing.T) {
	for _, data := range []string{"", `{`, `{"version":1,"target":"sdk","state":"active","changes":null}`} {
		t.Run(fmt.Sprintf("bytes=%d", len(data)), func(t *testing.T) {
			root := t.TempDir()
			marker := writeMarker(t, filepath.Join(root, "sdk"), "old")
			stage := writeMarker(t, filepath.Join(root, "sdk.staging"), "new")
			// The constructor writes and syncs journal.json.next before the
			// initial rename. No caller can mutate a target in this interval.
			txDir := filepath.Join(root, ".fstx-interrupted-initialization")
			require.NoError(t, os.Mkdir(txDir, 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(txDir, journalName+".next"), []byte(data), 0o600))
			require.NoError(t, Recover(root))
			assert.NoDirExists(t, txDir)
			assert.FileExists(t, marker)
			assert.FileExists(t, stage)
		})
	}
}

func TestCommitAfterMarkerPublicationDoesNotReportRollback(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "sdk")
	writeMarker(t, dest, "old")
	tx, err := NewTransaction(dest)
	require.NoError(t, err)
	require.NoError(t, tx.RemoveDir(dest))
	tx.syncDirectories = func(*os.Root, ...string) error {
		return errors.New("directory sync failed after journal rename")
	}
	require.NoError(t, tx.Commit())
	require.NoError(t, tx.Rollback())
	require.NoError(t, Recover(root))
	assert.NoDirExists(t, dest)
}

func TestCommitWithPreservesPublicationWhenFinalMarkerCannotBeWritten(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "sdk")
	stage := dest + stagingSuffix
	writeMarker(t, dest, "old")
	writeMarker(t, stage, "new")
	tx, err := NewTransaction(dest)
	require.NoError(t, err)
	require.NoError(t, tx.RemoveDir(dest))
	require.NoError(t, tx.RenameFile(stage, dest))
	require.NoError(t, tx.CommitWith(func() error {
		// Publish a reference, then obstruct the final journal update.
		writeMarker(t, root, "published-reference")
		writeMarker(t, filepath.Join(tx.tmpDir, journalName+".next"), "occupied")
		return nil
	}))
	require.NoError(t, tx.Rollback(), "publication has crossed the commit point")
	require.NoError(t, Recover(root))
	assert.FileExists(t, filepath.Join(dest, "new"))
	assert.FileExists(t, filepath.Join(root, "published-reference"))
	assert.NoDirExists(t, tx.tmpDir)
}

func TestCommitWithPublicationFailureAllowsRollback(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "sdk")
	stage := dest + stagingSuffix
	writeMarker(t, dest, "old")
	writeMarker(t, stage, "new")
	tx, err := NewTransaction(dest)
	require.NoError(t, err)
	require.NoError(t, tx.RemoveDir(dest))
	require.NoError(t, tx.RenameFile(stage, dest))
	failure := errors.New("settings publication failed")
	require.ErrorIs(t, tx.CommitWith(func() error { return failure }), failure)
	require.NoError(t, tx.Rollback())
	assert.FileExists(t, filepath.Join(dest, "old"))
	assert.NoFileExists(t, filepath.Join(dest, "new"))
}
