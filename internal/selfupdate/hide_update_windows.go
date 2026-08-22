//go:build windows

package selfupdate

import "golang.org/x/sys/windows"

func hideUpdateFile(path string) error {
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return windows.SetFileAttributes(ptr, windows.FILE_ATTRIBUTE_HIDDEN)
}
