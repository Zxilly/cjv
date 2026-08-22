//go:build windows

package selfupdate

import (
	"testing"

	"golang.org/x/sys/windows"
)

func fileIsHidden(t *testing.T, path string) bool {
	t.Helper()
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	attrs, err := windows.GetFileAttributes(ptr)
	if err != nil {
		t.Fatal(err)
	}
	return attrs&windows.FILE_ATTRIBUTE_HIDDEN != 0
}
