//go:build !windows

package fsops

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func tryLockFile(file *os.File) (bool, error) {
	for {
		err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if errors.Is(err, unix.EWOULDBLOCK) {
			return false, nil
		}
		return err == nil, err
	}
}
