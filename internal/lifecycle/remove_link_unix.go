//go:build !windows

package lifecycle

import (
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/fsops"
	"golang.org/x/sys/unix"
)

func linkedToolchainRemover(info os.FileInfo) func(*os.Root, string) error {
	if info.Mode()&os.ModeSymlink == 0 {
		return nil
	}
	return unlinkToolchainEntry
}

func unlinkToolchainEntry(root *os.Root, name string) error {
	if !filepath.IsLocal(name) || filepath.Base(name) != name || name == "." {
		return &os.PathError{Op: "unlinkat", Path: name, Err: os.ErrInvalid}
	}
	parent, err := root.Open(".")
	if err != nil {
		return err
	}
	defer parent.Close() //nolint:errcheck
	// Unlike os.Remove, flags=0 never falls back to removing an empty
	// directory if the entry changes after the caller's final link check.
	if err := fsops.Retry(func() error { return unix.Unlinkat(int(parent.Fd()), name, 0) }); err != nil {
		return &os.PathError{Op: "unlinkat", Path: name, Err: err}
	}
	return nil
}
