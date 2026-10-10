package fsops

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// IsPathUnder reports whether candidate resolves to a path inside base. Both
// arguments are made absolute before comparison so callers may pass relative
// paths. Returns false if the relation cannot be computed (e.g. different
// Windows volumes), since that itself indicates the path is not under base.
func IsPathUnder(base, candidate string) bool {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return false
	}
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absBase, absCandidate)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// linkStrategy creates dst as one form of link to src.
type linkStrategy func(src, dst string) error

// symlinkStrategy uses a relative target when src and dst share a directory,
// so the link remains valid if the parent directory is moved.
func symlinkStrategy(src, dst string) error {
	target := src
	if filepath.Dir(src) == filepath.Dir(dst) {
		target = filepath.Base(src)
	}
	return os.Symlink(target, dst)
}

func hardLinkStrategy(src, dst string) error {
	return os.Link(src, dst)
}

// CreateLink installs a proxy by removing the old entry, trying a symlink, then
// falling back to a hard link. An entry already pointing to src is left alone.
// There is no copy fallback or atomic replacement: if both link attempts fail,
// the old entry has already been removed. Callers own the destination entry.
func CreateLink(src, dst string) error {
	return createLinkWith([]linkStrategy{symlinkStrategy, hardLinkStrategy}, src, dst)
}

// createLinkWith tries each strategy directly at dst after removing its old
// entry. The first success ends the chain; otherwise the last error is returned.
func createLinkWith(strategies []linkStrategy, src, dst string) error {
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if destinationInfo, err := os.Stat(dst); err == nil {
		if os.SameFile(sourceInfo, destinationInfo) {
			return nil
		}
	}
	if info, err := os.Lstat(dst); err == nil {
		// os.Remove also removes empty directories, unlike a file unlink.
		// A proxy replacement must never remove a directory entry.
		if info.IsDir() {
			return &os.PathError{Op: "remove", Path: dst, Err: errors.New("proxy destination is a directory")}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := Retry(func() error { return os.Remove(dst) }); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	err = errors.New("fsops: no link strategy")
	for _, create := range strategies {
		if err = create(src, dst); err == nil {
			return nil
		}
	}
	return err
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
