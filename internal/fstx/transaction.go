// Package fstx owns filesystem transaction journals and their recovery. A
// transaction may change one destination tree and its adjacent staging tree.
package fstx

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/utils"
)

const (
	stateActive    = "active"
	stateRollback  = "rollback"
	statePrepared  = "prepared"
	stateCommitted = "committed"
)

type change struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	From   string `json:"from,omitempty"`
	Backup string `json:"backup,omitempty"`
}

type journal struct {
	Version int      `json:"version"`
	Target  string   `json:"target"`
	Scope   string   `json:"scope,omitempty"`
	State   string   `json:"state"`
	Changes []change `json:"changes"`
}

type Transaction struct {
	rootDir string
	tmpDir  string
	journal journal
	closed  bool
	// syncDirectories is the filesystem durability seam. Tests use it to
	// exercise errors after publishing a journal, without corrupting state.
	syncDirectories func(*os.Root, ...string) error
}

// RecoveryError means transaction data is still needed. Callers must not clean
// staging paths after this error; Recover retries once the obstruction is gone.
type RecoveryError struct {
	Directory string
	Err       error
}

func (e *RecoveryError) Error() string {
	return fmt.Sprintf("fstx: recovery incomplete; transaction retained at %s: %v", e.Directory, e.Err)
}

func (e *RecoveryError) Unwrap() error { return e.Err }

// NewTransaction records changes to prefix and its staging tree
// (config.StagingDir(prefix)). Existing destinations are never overwritten;
// remove them transactionally first.
func NewTransaction(prefix string) (*Transaction, error) {
	abs, err := filepath.Abs(prefix)
	if err != nil {
		return nil, err
	}
	if !validTarget(filepath.Base(abs)) {
		return nil, fmt.Errorf("fstx: invalid transaction target %q", prefix)
	}
	return newTransaction(filepath.Dir(abs), filepath.Base(abs), "")
}

// NewToolchainTransaction owns the SDK, stdx and documentation entries of one
// toolchain under home. All entries share one commit marker and recovery plan.
func NewToolchainTransaction(home, name string) (*Transaction, error) {
	if !validTarget(name) {
		return nil, fmt.Errorf("fstx: invalid toolchain name %q", name)
	}
	abs, err := filepath.Abs(home)
	if err != nil {
		return nil, err
	}
	return newTransaction(abs, name, "toolchain")
}

func newTransaction(rootDir, target, scope string) (*Transaction, error) {
	tempParent := rootDir
	if scope == "toolchain" {
		tempParent = filepath.Join(rootDir, config.ToolchainsSubdir)
		root, err := os.OpenRoot(rootDir)
		if err != nil {
			return nil, err
		}
		defer root.Close() //nolint:errcheck
		if err := checkParents(root, filepath.Join(config.ToolchainsSubdir, "entry")); err != nil {
			return nil, err
		}
		if err := root.MkdirAll(config.ToolchainsSubdir, 0o755); err != nil {
			return nil, err
		}
	}
	tmpDir, err := os.MkdirTemp(tempParent, config.TxTempPrefix+"*")
	if err != nil {
		return nil, fmt.Errorf("fstx: create temp dir: %w", err)
	}
	tx := &Transaction{rootDir: rootDir, tmpDir: tmpDir, journal: journal{
		Version: 1, Target: target, Scope: scope, State: stateActive,
	}}
	if err := tx.save(tx.journal); err != nil {
		// No caller has received the transaction, so no user data has moved.
		// Avoid leaving a partial initial journal that would block recovery.
		if cleanupErr := tx.cleanup(); cleanupErr != nil {
			return nil, &RecoveryError{Directory: tmpDir, Err: errors.Join(err, cleanupErr)}
		}
		return nil, err
	}
	return tx, nil
}

func (tx *Transaction) AddFile(src, dst string) error {
	rel, err := tx.relativePath(dst)
	if err != nil {
		return err
	}
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close() //nolint:errcheck
	info, err := input.Stat()
	if err != nil {
		return err
	}
	root, err := tx.openRoot()
	if err != nil {
		return err
	}
	defer root.Close() //nolint:errcheck
	if err := requireMissing(root, rel); err != nil {
		return err
	}
	if err := root.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		return err
	}
	if err := tx.record(change{Kind: "add", Path: rel}); err != nil {
		return err
	}
	out, err := root.OpenFile(rel, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, input)
	if copyErr == nil {
		copyErr = out.Sync()
	}
	return errors.Join(copyErr, out.Close())
}

func (tx *Transaction) RemoveFile(path string) error { return tx.remove(path) }
func (tx *Transaction) RemoveDir(path string) error  { return tx.remove(path) }

func (tx *Transaction) remove(path string) error {
	rel, err := tx.relativePath(path)
	if err != nil {
		return err
	}
	root, err := tx.openRoot()
	if err != nil {
		return err
	}
	defer root.Close() //nolint:errcheck
	if _, err := root.Lstat(rel); err != nil {
		return err
	}
	backup := fmt.Sprintf("%d-%s", len(tx.journal.Changes), filepath.Base(rel))
	if err := tx.record(change{Kind: "remove", Path: rel, Backup: backup}); err != nil {
		return err
	}
	return rename(root, rel, filepath.Join(tx.tempName(), backup))
}

func (tx *Transaction) RenameFile(from, to string) error {
	fromRel, err := tx.relativePath(from)
	if err != nil {
		return err
	}
	toRel, err := tx.relativePath(to)
	if err != nil {
		return err
	}
	root, err := tx.openRoot()
	if err != nil {
		return err
	}
	defer root.Close() //nolint:errcheck
	if _, err := root.Lstat(fromRel); err != nil {
		return err
	}
	if err := requireMissing(root, toRel); err != nil {
		return err
	}
	if err := root.MkdirAll(filepath.Dir(toRel), 0o755); err != nil {
		return err
	}
	if err := tx.record(change{Kind: "rename", Path: toRel, From: fromRel}); err != nil {
		return err
	}
	return rename(root, fromRel, toRel)
}

func (tx *Transaction) record(c change) error {
	if tx.closed || tx.journal.State != stateActive {
		return errors.New("fstx: transaction is no longer active")
	}
	next := tx.journal
	next.Changes = append(append([]change(nil), next.Changes...), c)
	return tx.save(next)
}

func (tx *Transaction) Commit() error {
	if tx.closed || tx.journal.State != stateActive {
		return errors.New("fstx: transaction already committed or rolled back")
	}
	next := tx.journal
	next.State = stateCommitted
	if err := tx.save(next); err != nil {
		if tx.journal.State != stateCommitted {
			return err
		}
		// Publishing the commit marker is the commit point. A directory sync
		// error after it must not send callers down their compensation path.
		slog.Warn("fstx: committed journal could not be synced", "path", tx.tmpDir, "error", err)
	}
	return tx.finishCommit()
}

// CommitWith publishes references to an already usable installed tree. Before
// publish runs, recovery is instructed to retain that tree, so a crash cannot
// leave published references dangling. A publication error still permits an
// explicit Rollback; the callback must leave no references to the new tree on
// error. After successful publication this operation cannot return a failure.
// Removals must publish their references before staging data and use Commit.
func (tx *Transaction) CommitWith(publish func() error) error {
	if tx.closed || tx.journal.State != stateActive {
		return errors.New("fstx: transaction already committed or rolled back")
	}
	next := tx.journal
	next.State = statePrepared
	if err := tx.save(next); err != nil {
		return err
	}
	if err := publish(); err != nil {
		return err
	}
	// The prepared marker already directs startup to keep this tree. Once
	// references are published, failure to write the final marker can safely
	// defer cleanup, but can no longer turn success into rollback.
	next.State = stateCommitted
	if err := tx.save(next); err != nil {
		slog.Warn("fstx: published installation cleanup deferred", "path", tx.tmpDir, "error", err)
		tx.closed = true
		return nil
	}
	return tx.finishCommit()
}

func (tx *Transaction) finishCommit() error {
	tx.closed = true
	if err := tx.cleanup(); err != nil {
		// The durable commit marker makes retrying cleanup safe at startup.
		slog.Warn("fstx: failed to clean committed transaction", "path", tx.tmpDir, "error", err)
	}
	return nil
}

func (tx *Transaction) Rollback() error {
	if tx.closed || tx.journal.State == stateCommitted {
		return nil
	}
	if err := tx.rollback(); err != nil {
		return &RecoveryError{Directory: tx.tmpDir, Err: err}
	}
	tx.closed = true
	return nil
}

func (tx *Transaction) rollback() error {
	next := tx.journal
	next.State = stateRollback
	if err := tx.save(next); err != nil {
		return err
	}
	root, err := tx.openRoot()
	if err != nil {
		return err
	}
	defer root.Close() //nolint:errcheck
	for len(tx.journal.Changes) > 0 {
		c := tx.journal.Changes[len(tx.journal.Changes)-1]
		switch c.Kind {
		case "add":
			err = root.Remove(c.Path)
			if errors.Is(err, os.ErrNotExist) {
				err = nil
			}
		case "remove":
			err = restore(root, filepath.Join(tx.tempName(), c.Backup), c.Path)
		case "rename":
			err = restore(root, c.Path, c.From)
		}
		if err != nil {
			// Later reversals depend on this one: never restore an old SDK over
			// the new SDK if moving the new SDK back to staging failed.
			return err
		}
		next := tx.journal
		next.Changes = next.Changes[:len(next.Changes)-1]
		if err := tx.save(next); err != nil {
			return err
		}
	}
	return tx.cleanup()
}

// restore is also safe after interruption between an undo and its journal
// update. Existing destinations are obstructions, never permission to delete.
func restore(root *os.Root, from, to string) error {
	if _, err := root.Lstat(from); errors.Is(err, os.ErrNotExist) {
		_, err = root.Lstat(to)
		return err
	} else if err != nil {
		return err
	}
	return rename(root, from, to)
}

func rename(root *os.Root, from, to string) error {
	if err := requireMissing(root, to); err != nil {
		return &os.LinkError{Op: "rename", Old: filepath.Join(root.Name(), from), New: filepath.Join(root.Name(), to), Err: err}
	}
	if err := utils.RetryWithBackoff(10, utils.IsRetryableError, func() error { return root.Rename(from, to) }); err != nil {
		return err
	}
	return syncDirs(root, filepath.Dir(from), filepath.Dir(to))
}

func requireMissing(root *os.Root, path string) error {
	if _, err := root.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("%s: %w", path, os.ErrExist)
}

func (tx *Transaction) cleanup() error {
	root, err := tx.openRoot()
	if err != nil {
		return err
	}
	defer root.Close() //nolint:errcheck
	dir := tx.tempName()
	f, err := root.Open(dir)
	if err != nil {
		return err
	}
	entries, readErr := f.ReadDir(-1)
	if err := errors.Join(readErr, f.Close()); err != nil {
		return err
	}
	// Keep the commit marker until every backup is gone. If cleanup itself is
	// interrupted, the remaining backups must never look like legacy recovery.
	for _, entry := range entries {
		if entry.Name() == journalName {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if err := utils.RetryWithBackoff(10, utils.IsRetryableError, func() error { return root.RemoveAll(path) }); err != nil {
			return err
		}
	}
	if err := syncDirs(root, dir); err != nil {
		return err
	}
	if err := root.Remove(filepath.Join(dir, journalName)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return root.Remove(dir)
}

func (tx *Transaction) tempName() string {
	name := filepath.Base(tx.tmpDir)
	if tx.journal.Scope == "toolchain" {
		return filepath.Join(config.ToolchainsSubdir, name)
	}
	return name
}
