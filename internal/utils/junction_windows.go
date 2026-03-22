//go:build windows

package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// SymlinkOrJunction creates a directory symlink, falling back to a
// directory junction if symlinks require elevated privileges.
func SymlinkOrJunction(target, link string) error {
	if err := os.Symlink(target, link); err == nil {
		return nil
	}
	return createJunction(target, link)
}

func createJunction(target, link string) error {
	// Resolve relative targets against the link's parent directory (not CWD)
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(link), target)
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	target = filepath.Clean(absTarget)

	if err := os.MkdirAll(link, 0o755); err != nil {
		return err
	}

	linkW, err := windows.UTF16PtrFromString(link)
	if err != nil {
		return errors.Join(err, os.RemoveAll(link))
	}

	handle, err := windows.CreateFile(
		linkW,
		windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return errors.Join(err, os.RemoveAll(link))
	}
	defer windows.CloseHandle(handle) //nolint:errcheck // best-effort cleanup

	// Target must be a NT path for junctions
	ntTarget := `\??\` + target

	targetW, err := windows.UTF16FromString(ntTarget)
	if err != nil {
		return errors.Join(err, os.RemoveAll(link))
	}

	targetByteLen := len(targetW) * 2
	if targetByteLen > 65535 {
		return errors.Join(fmt.Errorf("junction target path too long (%d bytes)", targetByteLen), os.RemoveAll(link))
	}
	targetBytes := unsafe.Slice((*byte)(unsafe.Pointer(&targetW[0])), targetByteLen)

	// REPARSE_DATA_BUFFER layout for IO_REPARSE_TAG_MOUNT_POINT:
	//   [0..3]   ReparseTag           (4 bytes)
	//   [4..5]   ReparseDataLength    (2 bytes)
	//   [6..7]   Reserved             (2 bytes)
	//   [8..9]   SubstituteNameOffset (2 bytes)
	//   [10..11] SubstituteNameLength (2 bytes)
	//   [12..13] PrintNameOffset      (2 bytes)
	//   [14..15] PrintNameLength      (2 bytes)
	//   [16..]   PathBuffer: SubstituteName (with NUL) + PrintName (with NUL)
	//
	// Even when PrintName is empty (length 0), Windows requires its
	// NUL terminator (2 bytes for UTF-16) to be present in PathBuffer.
	const headerSize = 16
	bufSize := headerSize + len(targetBytes) + 2 // +2 for PrintName's NUL terminator
	buf := make([]byte, bufSize)

	// ReparseTag = IO_REPARSE_TAG_MOUNT_POINT (0xA0000003)
	buf[0] = 0x03
	buf[1] = 0x00
	buf[2] = 0x00
	buf[3] = 0xA0

	dataLen := uint16(bufSize - 8)
	buf[4] = byte(dataLen)
	buf[5] = byte(dataLen >> 8)
	// Reserved = 0 (bytes 6-7)

	// SubstituteNameOffset = 0
	// SubstituteNameLength
	substLen := uint16(len(targetBytes) - 2) // exclude null terminator
	buf[10] = byte(substLen)
	buf[11] = byte(substLen >> 8)

	// PrintNameOffset = substLen + 2
	printOff := substLen + 2
	buf[12] = byte(printOff)
	buf[13] = byte(printOff >> 8)
	// PrintNameLength = 0

	copy(buf[headerSize:], targetBytes)

	var bytesReturned uint32
	err = windows.DeviceIoControl(
		handle,
		windows.FSCTL_SET_REPARSE_POINT,
		&buf[0],
		uint32(bufSize),
		nil,
		0,
		&bytesReturned,
		nil,
	)
	if err != nil {
		return errors.Join(err, os.RemoveAll(link))
	}

	return nil
}
