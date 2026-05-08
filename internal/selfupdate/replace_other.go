//go:build !windows

package selfupdate

import "github.com/Zxilly/cjv/internal/utils"

func replaceManagedExecutableFile(src, dst string) error {
	return utils.RenameRetry(src, dst)
}
