//go:build !windows

package utils

// PauseIfStandaloneConsole is a no-op outside Windows. Unix terminals don't
// vanish when the launching process exits — the user's shell owns the window.
func PauseIfStandaloneConsole() {}

// EnableConsoleUTF8 is a no-op outside Windows. Unix terminals are UTF-8 and
// have no per-console code page to switch.
func EnableConsoleUTF8() func() { return func() {} }
