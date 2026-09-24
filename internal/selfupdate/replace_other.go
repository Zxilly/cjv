//go:build !windows

package selfupdate

import "github.com/Zxilly/cjv/internal/fsops"

func replaceManagedExecutableFile(src, dst string) error {
	return fsops.RenameRetry(src, dst)
}
