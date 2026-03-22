package env

import (
	"errors"
	"os"
	"slices"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procSendMessageTimeoutW = user32.NewProc("SendMessageTimeoutW")
)

const (
	hwndBroadcast   = 0xFFFF
	wmSettingChange = 0x001A
	smtoAbortIfHung = 0x0002
)

// broadcastSettingChange notifies other processes that environment variables have changed.
func broadcastSettingChange() {
	env, _ := syscall.UTF16PtrFromString("Environment")
	_, _, _ = procSendMessageTimeoutW.Call(
		hwndBroadcast,
		wmSettingChange,
		0,
		uintptr(unsafe.Pointer(env)),
		smtoAbortIfHung,
		5000,
		0,
	)
}

// findPathValueName enumerates the registry key's value names and returns
// the actual casing of the "Path" value (e.g. "Path", "PATH", "path").
// Windows registry value names are case-insensitive, but the actual stored
// casing varies. If no match is found, returns "Path" as the default.
func findPathValueName(key registry.Key) string {
	names, err := key.ReadValueNames(-1)
	if err != nil {
		return "Path"
	}
	for _, name := range names {
		if strings.EqualFold(name, "Path") {
			return name
		}
	}
	return "Path"
}

// AddPathToWindowsRegistry adds binDir to HKCU\Environment\Path.
// Uses REG_EXPAND_SZ to preserve %VAR% expansions in existing entries.
func AddPathToWindowsRegistry(binDir string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer key.Close() //nolint:errcheck // best-effort cleanup

	pathName := findPathValueName(key)

	existing, _, err := key.GetStringValue(pathName)
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}

	// Check if already present
	entries := strings.Split(existing, string(os.PathListSeparator))
	if slices.ContainsFunc(entries, func(e string) bool {
		return strings.EqualFold(e, binDir)
	}) {
		return nil
	}

	newPath := binDir
	if existing != "" {
		newPath = binDir + string(os.PathListSeparator) + existing
	}

	if err := setPathValue(key, pathName, newPath); err != nil {
		return err
	}

	broadcastSettingChange()
	return nil
}

// RemovePathFromWindowsRegistry removes binDir from HKCU\Environment\Path.
func RemovePathFromWindowsRegistry(binDir string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return nil // Key doesn't exist, nothing to remove
	}
	defer key.Close() //nolint:errcheck // best-effort cleanup

	pathName := findPathValueName(key)

	existing, _, err := key.GetStringValue(pathName)
	if err != nil {
		return nil
	}

	entries := strings.Split(existing, string(os.PathListSeparator))
	original := len(entries)
	entries = slices.DeleteFunc(entries, func(e string) bool {
		return strings.EqualFold(e, binDir)
	})
	if len(entries) == original {
		return nil // nothing to remove, avoid unnecessary registry write
	}

	if err := setPathValue(key, pathName, strings.Join(entries, string(os.PathListSeparator))); err != nil {
		return err
	}

	broadcastSettingChange()
	return nil
}

// setPathValue writes a PATH registry value, always using EXPAND_SZ
// to preserve %VAR% references in existing PATH entries.
func setPathValue(key registry.Key, name, value string) error {
	return key.SetExpandStringValue(name, value)
}
