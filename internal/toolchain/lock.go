package toolchain

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/fsops"
)

// HomeLock serializes recovery and lifecycle mutations in one CJV_HOME.
// It is not reentrant: a holder must use Recover instead of RecoverHome.
// Keep it until staging cleanup and transaction commit/rollback are finished.
type HomeLock struct {
	file  *os.File
	tcDir string
}

func LockHome(ctx context.Context) (*HomeLock, error) {
	home, err := config.Home()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return nil, err
	}
	file, err := fsops.LockFile(ctx, filepath.Join(home, ".lifecycle.lock"))
	if err != nil {
		return nil, err
	}
	return &HomeLock{file: file, tcDir: filepath.Join(home, config.ToolchainsSubdir)}, nil
}

func (lock *HomeLock) Close() error { return lock.file.Close() }
