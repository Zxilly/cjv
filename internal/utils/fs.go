package utils

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

var (
	createSymlink   = os.Symlink
	createHardLink  = os.Link
	copyFileForLink = CopyFile
)

// CreateLink creates a link from src to dst with three-level fallback:
// symlink -> hard link -> copy.
// When src and dst are in the same directory, symlinks use a relative target
// so the link remains valid if the parent directory is moved.
//
// The replacement is staged through a temporary path so a failed update does
// not delete an existing destination.
func CreateLink(src, dst string) error {
	tmpPath, err := createReplacementPath(dst)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup

	symlinkTarget := src
	if filepath.Dir(src) == filepath.Dir(dst) {
		symlinkTarget = filepath.Base(src)
	}
	if err := createSymlink(symlinkTarget, tmpPath); err == nil {
		return RenameRetry(tmpPath, dst)
	}
	if err := createHardLink(src, tmpPath); err == nil {
		return RenameRetry(tmpPath, dst)
	}
	if err := copyFileForLink(src, tmpPath, 0o755); err != nil {
		return err
	}
	return RenameRetry(tmpPath, dst)
}

func createReplacementPath(dst string) (string, error) {
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	f, err := os.CreateTemp(dir, "."+filepath.Base(dst)+"-*")
	if err != nil {
		return "", err
	}
	path := f.Name()
	if err := f.Close(); err != nil {
		return "", errors.Join(err, os.Remove(path))
	}
	if err := os.Remove(path); err != nil {
		return "", err
	}
	return path, nil
}

// CopyFile copies a single file from src to dst with the given permissions.
func CopyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck // read-only

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	if err != nil {
		return errors.Join(err, out.Close(), os.Remove(dst))
	}
	if err := out.Close(); err != nil {
		return errors.Join(err, os.Remove(dst))
	}
	return nil
}
