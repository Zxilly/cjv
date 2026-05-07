//go:build !windows

package utils

// PauseIfStandaloneConsole is a no-op outside Windows. Unix terminals don't
// vanish when the launching process exits — the user's shell owns the window.
func PauseIfStandaloneConsole() {}
