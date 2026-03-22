package fstx

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/utils"
)

type changeKind int

const (
	changeAddedFile changeKind = iota
	changeRemovedFile
	changeRenamedFile
	changeRemovedDir
)

type change struct {
	kind       changeKind
	path       string // target path (for rename: destination)
	backupPath string // only for remove operations
	from       string // only for rename operations
}

type Transaction struct {
	tmpDir    string
	changes   []change
	committed bool
}

func NewTransaction(prefix string) (*Transaction, error) {
	tmpDir, err := os.MkdirTemp(filepath.Dir(prefix), ".fstx-*")
	if err != nil {
		return nil, fmt.Errorf("fstx: create temp dir: %w", err)
	}
	return &Transaction{
		tmpDir: tmpDir,
	}, nil
}

func (tx *Transaction) AddFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("fstx: mkdir for add: %w", err)
	}
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("fstx: stat source: %w", err)
	}
	if err := utils.CopyFile(src, dst, info.Mode()); err != nil {
		return fmt.Errorf("fstx: copy file: %w", err)
	}
	tx.changes = append(tx.changes, change{kind: changeAddedFile, path: dst})
	return nil
}

func (tx *Transaction) backupPath(originalPath string) string {
	name := fmt.Sprintf("%d-%s", len(tx.changes), filepath.Base(originalPath))
	return filepath.Join(tx.tmpDir, name)
}

func (tx *Transaction) RemoveFile(path string) error {
	backup := tx.backupPath(path)
	if err := utils.RenameRetry(path, backup); err != nil {
		return fmt.Errorf("fstx: backup file for removal: %w", err)
	}
	tx.changes = append(tx.changes, change{kind: changeRemovedFile, path: path, backupPath: backup})
	return nil
}

func (tx *Transaction) RemoveDir(path string) error {
	backup := tx.backupPath(path)
	if err := utils.RenameRetry(path, backup); err != nil {
		return fmt.Errorf("fstx: backup dir for removal: %w", err)
	}
	tx.changes = append(tx.changes, change{kind: changeRemovedDir, path: path, backupPath: backup})
	return nil
}

func (tx *Transaction) RenameFile(from, to string) error {
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return fmt.Errorf("fstx: mkdir for rename: %w", err)
	}
	if err := utils.RenameRetry(from, to); err != nil {
		return fmt.Errorf("fstx: rename: %w", err)
	}
	tx.changes = append(tx.changes, change{kind: changeRenamedFile, path: to, from: from})
	return nil
}

func (tx *Transaction) Commit() error {
	if tx.committed {
		return errors.New("fstx: transaction already committed or rolled back")
	}
	tx.committed = true
	if err := os.RemoveAll(tx.tmpDir); err != nil {
		slog.Warn("fstx: failed to clean temp dir after commit", "path", tx.tmpDir, "error", err)
	}
	return nil
}

func (tx *Transaction) Rollback() error {
	if tx.committed {
		return nil
	}
	var errs []error
	for i := len(tx.changes) - 1; i >= 0; i-- {
		c := tx.changes[i]
		switch c.kind {
		case changeAddedFile:
			if err := os.RemoveAll(c.path); err != nil {
				slog.Error("fstx: rollback remove added", "path", c.path, "error", err)
				errs = append(errs, err)
			}
		case changeRemovedFile, changeRemovedDir:
			if err := utils.RenameRetry(c.backupPath, c.path); err != nil {
				slog.Error("fstx: rollback restore removed", "path", c.path, "error", err)
				errs = append(errs, err)
			}
		case changeRenamedFile:
			if err := utils.RenameRetry(c.path, c.from); err != nil {
				slog.Error("fstx: rollback rename", "from", c.path, "to", c.from, "error", err)
				errs = append(errs, err)
			}
		}
	}
	tx.committed = true // prevent double rollback
	if err := os.RemoveAll(tx.tmpDir); err != nil {
		slog.Warn("fstx: failed to clean temp dir after rollback", "path", tx.tmpDir, "error", err)
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
