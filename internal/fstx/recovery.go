package fstx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Zxilly/cjv/internal/utils"
)

const journalName = "journal.json"

// Recover rolls back unfinished transactions and cleans committed ones under
// rootDir. On any ambiguity it preserves the transaction and reports its path.
// Call this before removing abandoned staging trees or starting a new install.
func Recover(rootDir string) error {
	abs, err := filepath.Abs(rootDir)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(abs)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), tempPrefix) {
			continue
		}
		tx := &Transaction{rootDir: abs, tmpDir: filepath.Join(abs, entry.Name())}
		if err := tx.recover(); err != nil {
			return &RecoveryError{Directory: tx.tmpDir, Err: err}
		}
	}
	return nil
}

func (tx *Transaction) recover() error {
	root, err := tx.openRoot()
	if err != nil {
		return err
	}
	defer root.Close() //nolint:errcheck
	name := filepath.Join(filepath.Base(tx.tmpDir), journalName)
	info, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return tx.recoverLegacy(root)
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("fstx: journal must be a regular file: %s", name)
	}
	if info.Size() > 1<<20 {
		return errors.New("fstx: journal exceeds maximum size")
	}
	f, err := root.Open(name)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck
	decoder := json.NewDecoder(io.LimitReader(f, 1<<20))
	decoder.DisallowUnknownFields()
	var state journal
	if err := decoder.Decode(&state); err != nil {
		return fmt.Errorf("fstx: invalid journal: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("fstx: journal contains trailing data")
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := validateJournal(state); err != nil {
		return err
	}
	tx.journal = state
	if state.Scope == "toolchain" {
		if filepath.Base(tx.rootDir) != "toolchains" {
			return errors.New("fstx: toolchain transaction must be under toolchains")
		}
		tx.rootDir = filepath.Dir(tx.rootDir)
		root, err = tx.openRoot()
		if err != nil {
			return err
		}
		defer root.Close() //nolint:errcheck
	}
	// Validate all paths before performing even the first recovery operation.
	for _, c := range state.Changes {
		if err := checkParents(root, c.Path); err != nil {
			return err
		}
		if c.From != "" {
			if err := checkParents(root, c.From); err != nil {
				return err
			}
		}
	}
	if state.State == stateCommitted || state.State == statePrepared {
		return tx.cleanup()
	}
	return tx.rollback()
}

func validateJournal(state journal) error {
	if state.Version != 1 || !validTarget(state.Target) {
		return errors.New("fstx: invalid journal version or target")
	}
	if state.Scope != "" && state.Scope != "toolchain" {
		return errors.New("fstx: invalid transaction scope")
	}
	if state.State != stateActive && state.State != stateRollback && state.State != statePrepared && state.State != stateCommitted {
		return fmt.Errorf("fstx: invalid transaction state %q", state.State)
	}
	for i, c := range state.Changes {
		if !scopedPath(state, c.Path) {
			return fmt.Errorf("fstx: path outside transaction target: %q", c.Path)
		}
		switch c.Kind {
		case "add":
			if c.From != "" || c.Backup != "" {
				return errors.New("fstx: invalid add operation")
			}
		case "remove":
			if c.From != "" || c.Backup != fmt.Sprintf("%d-%s", i, filepath.Base(c.Path)) {
				return errors.New("fstx: invalid backup operation")
			}
		case "rename":
			if !scopedPath(state, c.From) || c.From == c.Path || c.Backup != "" {
				return errors.New("fstx: invalid rename operation")
			}
		default:
			return fmt.Errorf("fstx: unknown transaction operation %q", c.Kind)
		}
	}
	return nil
}

func validTarget(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.HasPrefix(name, tempPrefix) &&
		!strings.ContainsAny(name, `/\`) && filepath.IsLocal(name) && filepath.Clean(name) == name
}

func scopedPath(state journal, path string) bool {
	if !filepath.IsLocal(path) || filepath.Clean(path) != path {
		return false
	}
	if state.Scope == "toolchain" {
		for _, base := range []string{"toolchains", "stdx", "docs"} {
			owned := filepath.Join(base, state.Target)
			if path == owned || strings.HasPrefix(path, owned+string(filepath.Separator)) {
				return true
			}
			if base == "toolchains" && (path == owned+stagingSuffix || strings.HasPrefix(path, owned+stagingSuffix+string(filepath.Separator))) {
				return true
			}
		}
		return false
	}
	first, _, _ := strings.Cut(path, string(filepath.Separator))
	return first == state.Target || first == state.Target+stagingSuffix
}

func (tx *Transaction) relativePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(tx.rootDir, abs)
	if err != nil || !scopedPath(tx.journal, rel) {
		return "", fmt.Errorf("fstx: path outside transaction target: %s", path)
	}
	root, err := tx.openRoot()
	if err != nil {
		return "", err
	}
	defer root.Close() //nolint:errcheck
	return rel, checkParents(root, rel)
}

// Reject linked parent directories even when they point inside the root: a
// transaction owns entries, not the contents of a user-linked SDK. os.Root also
// enforces containment during operations if a parent changes after this check.
func checkParents(root *os.Root, path string) error {
	for parent := filepath.Dir(path); parent != "."; parent = filepath.Dir(parent) {
		info, err := root.Lstat(parent)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("fstx: refusing linked parent directory %s", parent)
		}
	}
	return nil
}

func (tx *Transaction) openRoot() (*os.Root, error) {
	root, err := os.OpenRoot(tx.rootDir)
	if err != nil {
		return nil, err
	}
	if err := checkParents(root, tx.tempName()); err != nil {
		_ = root.Close() //nolint:errcheck
		return nil, err
	}
	info, err := root.Lstat(tx.tempName())
	if err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
		err = errors.New("fstx: transaction directory must not be a link or file")
	}
	if err != nil {
		_ = root.Close() //nolint:errcheck
		return nil, err
	}
	return root, nil
}

// save publishes the journal before a mutation or backup deletion. Each undo
// is idempotent across the rename-to-journal gap, so an interrupted save remains
// recoverable using the previous complete journal.
func (tx *Transaction) save(state journal) error {
	if err := validateJournal(state); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	root, err := tx.openRoot()
	if err != nil {
		return err
	}
	defer root.Close() //nolint:errcheck
	dir := tx.tempName()
	temp := filepath.Join(dir, journalName+".next")
	if err := root.Remove(temp); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	f, err := root.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(data)
	if writeErr == nil {
		writeErr = f.Sync()
	}
	if err := errors.Join(writeErr, f.Close()); err != nil {
		return err
	}
	if err := utils.RetryWithBackoff(10, utils.IsRetryableError, func() error {
		return root.Rename(temp, filepath.Join(dir, journalName))
	}); err != nil {
		return err
	}
	tx.journal = state
	if tx.syncDirectories != nil {
		return tx.syncDirectories(root, dir, filepath.Dir(dir))
	}
	return syncDirs(root, dir, filepath.Dir(dir))
}

func syncDirs(root *os.Root, dirs ...string) error {
	// Windows does not support syncing directories through an os.File handle.
	if runtime.GOOS == "windows" {
		return nil
	}
	for _, dir := range dirs {
		f, err := root.Open(dir)
		if err != nil {
			return err
		}
		if err := errors.Join(f.Sync(), f.Close()); err != nil {
			return err
		}
	}
	return nil
}

// Legacy transactions have no commit marker. A present destination gives no
// evidence that its backup is obsolete, so preserve both until resolved.
func (tx *Transaction) recoverLegacy(root *os.Root) error {
	dir := filepath.Base(tx.tmpDir)
	f, err := root.Open(dir)
	if err != nil {
		return err
	}
	entries, readErr := f.ReadDir(-1)
	if err := errors.Join(readErr, f.Close()); err != nil {
		return err
	}
	// The first journal is fully written before any transaction mutation is
	// possible. Interruption before its rename can leave only this temporary
	// file (possibly partial); it owns no backup and needs no user recovery.
	if len(entries) == 1 && entries[0].Name() == journalName+".next" && entries[0].Type().IsRegular() {
		return tx.cleanup()
	}
	for _, entry := range entries {
		index, original, ok := strings.Cut(entry.Name(), "-")
		n, err := strconv.Atoi(index)
		if !ok || err != nil || n < 0 || !validTarget(original) {
			return fmt.Errorf("fstx: unrecognized legacy backup %s", entry.Name())
		}
	}
	for _, entry := range entries {
		_, original, _ := strings.Cut(entry.Name(), "-")
		if err := rename(root, filepath.Join(dir, entry.Name()), original); err != nil {
			return err
		}
	}
	return tx.cleanup()
}
