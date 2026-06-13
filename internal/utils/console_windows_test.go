//go:build windows

package utils

import "testing"

// TestEnableConsoleUTF8RestoresCodePage verifies that the restore function hands
// the console's output code page back to its original value, so cjv never
// leaves the hosting shell switched to UTF-8 behind it. When no console is
// attached (GetConsoleOutputCP returns 0), both reads are 0 and the round-trip
// is still expected to be a no-op.
func TestEnableConsoleUTF8RestoresCodePage(t *testing.T) {
	before, _, _ := procGetConsoleOutputCP.Call()

	restore := EnableConsoleUTF8()
	if restore == nil {
		t.Fatal("EnableConsoleUTF8 returned a nil restore func")
	}
	restore()

	after, _, _ := procGetConsoleOutputCP.Call()
	if before != after {
		t.Errorf("console output code page not restored: before=%d after=%d", before, after)
	}
}
