//go:build !windows

package main

// pauseIfStandaloneConsole is a no-op outside Windows. Unix terminals don't
// vanish when the launching process exits — the user's shell owns the window.
func pauseIfStandaloneConsole() {}

// enableConsoleUTF8 is a no-op outside Windows. Unix terminals are UTF-8 and
// have no per-console code page to switch.
func enableConsoleUTF8() func() { return func() {} }
