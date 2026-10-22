package selfupdate

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/fsops"
)

// LockManaged serializes recovery, binary replacement and proxy refresh across
// self-update and init. The lock lives outside bin so replacing bin cannot
// create a second lock inode while a process is waiting.
func LockManaged(ctx context.Context) (*os.File, error) {
	home, err := config.Home()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return nil, err
	}
	return fsops.LockFile(ctx, filepath.Join(home, ".self-update.lock"))
}
