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

func copyStrategy(src, dst string) error {
	return CopyFile(src, dst, 0o755)
}

// CreateLink creates a link from src to dst with three-level fallback:
// symlink -> hard link -> copy.
//
// The replacement is staged through a temporary path so a failed update does
// not delete an existing destination.
func CreateLink(src, dst string) error {
	return createLinkWith([]linkStrategy{symlinkStrategy, hardLinkStrategy, copyStrategy}, src, dst)
}

// createLinkWith tries each strategy in order against a temporary path next
// to dst and moves the first success into place. When every strategy fails,
// the last error is returned and dst is left untouched.
func createLinkWith(strategies []linkStrategy, src, dst string) error {
	tmpPath, err := createReplacementPath(dst)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup

	err = errors.New("fsops: no link strategy")
	for _, create := range strategies {
		if err = create(src, tmpPath); err == nil {
			return RenameRetry(tmpPath, dst)
		}
	}
	return err
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
