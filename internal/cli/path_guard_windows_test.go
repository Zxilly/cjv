//go:build windows

package cli

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"golang.org/x/sys/windows/registry"
)

// registryPathGuard saves the current HKCU\Environment\Path value and
// restores it when restore() is called. This is the Go equivalent of
// a registry guard — tests exercise real registry writes while
// the guard guarantees the system is left untouched afterward.
type registryPathGuard struct {
	name    string // actual casing of the "Path" value name
	value   string // original PATH content
	valType uint32 // original value type (REG_SZ or REG_EXPAND_SZ)
	existed bool   // whether the value existed before
}

func saveRegistryPath() (*registryPathGuard, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE)
	if err != nil {
		// Key does not exist — nothing to save.
		return &registryPathGuard{name: "Path"}, nil
	}
	defer key.Close()

	name := findRegistryPathName(key)
	val, valType, err := key.GetStringValue(name)
	if err != nil {
		// Value does not exist under this key.
		return &registryPathGuard{name: name}, nil
	}
	return &registryPathGuard{name: name, value: val, valType: valType, existed: true}, nil
}

func (g *registryPathGuard) restore() {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		fmt.Fprintf(os.Stderr, "registryPathGuard: failed to open registry for restore: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "registryPathGuard: failed to restore PATH: %v\n", setErr)
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

func runWithPathGuard(m *testing.M) int {
	guard, err := saveRegistryPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save registry PATH, disabling PATH writes: %v\n", err)
		ensurePathConfiguredFn = func() {}
		return m.Run()
	}
	defer guard.restore()

	// Let ensurePathConfiguredFn keep its default value (the real
	// ensurePathConfigured) so that CI exercises the actual code path.
	return m.Run()
}
