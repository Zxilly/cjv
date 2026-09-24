//go:build !windows

package fsops

import "os"

// SymlinkOrJunction creates a directory symlink.
// On non-Windows platforms, this is just os.Symlink.
func SymlinkOrJunction(target, link string) error {
	return os.Symlink(target, link)
}
