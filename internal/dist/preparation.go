package dist

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/fsops"
	"github.com/Zxilly/cjv/internal/progress"
)

// Preparation owns shared downloads and private scratch for one installation.
// Keep it open through publication: its lock excludes other installations and
// purges, while the separate home lock can be released during network work.
// BeginPreparation must precede taking the home lock.
type Preparation struct {
	directory string
	lock      *os.File
	scratch   []string
	archives  map[string]struct{}
	complete  bool
}

// BeginPreparation acquires the installation lock beside the downloads
// directory. A canceled wait leaves existing downloads and scratch untouched.
func BeginPreparation(ctx context.Context, downloadsDir string) (*Preparation, error) {
	if downloadsDir == "" {
		return nil, fmt.Errorf("downloads directory is empty")
	}
	directory, err := filepath.Abs(downloadsDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(directory), 0o755); err != nil {
		return nil, err
	}
	lock, err := fsops.LockFile(ctx, filepath.Join(filepath.Dir(directory), ".install.lock"))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		_ = lock.Close() //nolint:errcheck // preserve the directory error
		return nil, err
	}
	return &Preparation{directory: directory, lock: lock, archives: make(map[string]struct{})}, nil
}

// TempDir creates private scratch removed on Close regardless of the outcome.
// Durable transaction journals and their backups must live outside this scratch.
func (p *Preparation) TempDir(pattern string) (string, error) {
	dir, err := os.MkdirTemp(p.directory, pattern)
	if err != nil {
		return "", err
	}
	p.scratch = append(p.scratch, dir)
	return dir, nil
}

// Download retains a verified archive for retries until the entire operation
// completes. Local user-supplied archives never enter this ownership set.
func (p *Preparation) Download(ctx context.Context, url, sha256, displayName string, sink progress.Sink) (string, error) {
	archive, err := DownloadCachedWithName(ctx, url, sha256, p.directory, displayName, sink)
	if err != nil {
		return "", err
	}
	p.archives[archive] = struct{}{}
	return archive, nil
}

// Complete marks successful publication. Close will now discard the downloads;
// without Complete, verified archives and resumable partials remain for retry.
func (p *Preparation) Complete() { p.complete = true }

// Close removes private scratch and releases the installation lock last.
// It is safe to call more than once. Cleanup never traverses local input archives.
func (p *Preparation) Close() error {
	if p == nil || p.lock == nil {
		return nil
	}
	var errs []error
	for _, dir := range p.scratch {
		errs = append(errs, fsops.RemoveAllRetry(dir))
	}
	if p.complete {
		for archive := range p.archives {
			errs = append(errs, CleanupDownload(archive))
		}
	}
	errs = append(errs, p.lock.Close())
	p.lock = nil
	return errors.Join(errs...)
}
