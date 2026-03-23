//go:build windows

package testutil

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// RegistryPathGuard saves the current HKCU\Environment\Path value and
// restores it when Restore() is called. Tests exercise real registry writes
// while the guard guarantees the system is left untouched afterward.
type RegistryPathGuard struct {
	name    string // actual casing of the "Path" value name
	value   string // original PATH content
	valType uint32 // original value type (REG_SZ or REG_EXPAND_SZ)
	existed bool   // whether the value existed before
}

// SaveRegistryPath snapshots the current HKCU\Environment\Path value.
func SaveRegistryPath() (*RegistryPathGuard, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE)
	if err != nil {
		// Key does not exist — nothing to save.
		return &RegistryPathGuard{name: "Path"}, nil
	}
	defer key.Close()

	name := findRegistryPathName(key)
	val, valType, err := key.GetStringValue(name)
	if err != nil {
		// Value does not exist under this key.
		return &RegistryPathGuard{name: name}, nil
	}
	return &RegistryPathGuard{name: name, value: val, valType: valType, existed: true}, nil
}

// Restore writes back the original registry value.
func (g *RegistryPathGuard) Restore() {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		fmt.Fprintf(os.Stderr, "RegistryPathGuard: failed to open registry for restore: %v\n", err)
		return
	}
	defer key.Close()

	if !g.existed {
		_ = key.DeleteValue(g.name)
		return
	}
	// Restore using the original value type to avoid REG_SZ ↔ REG_EXPAND_SZ
	// conversion that would alter the raw bytes stored in the registry.
	var setErr error
	if g.valType == registry.EXPAND_SZ {
		setErr = key.SetExpandStringValue(g.name, g.value)
	} else {
		setErr = key.SetStringValue(g.name, g.value)
	}
	if setErr != nil {
		fmt.Fprintf(os.Stderr, "RegistryPathGuard: failed to restore PATH: %v\n", setErr)
	}
}

// findRegistryPathName returns the actual casing of the "Path" value name
// (e.g. "Path", "PATH", "path") in the given registry key.
func findRegistryPathName(key registry.Key) string {
	names, err := key.ReadValueNames(-1)
	if err != nil {
		return "Path"
	}
	for _, n := range names {
		if strings.EqualFold(n, "Path") {
			return n
		}
	}
	return "Path"
}

// ReadRegistryPath reads the current HKCU\Environment\Path value.
func ReadRegistryPath() (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	defer key.Close()

	name := findRegistryPathName(key)
	val, _, err := key.GetStringValue(name)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return val, nil
}
