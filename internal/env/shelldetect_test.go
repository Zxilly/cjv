package env

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyShellName(t *testing.T) {
	// Cross-platform test cases
	tests := []struct {
		name     string
		expected ShellType
		ok       bool
	}{
		{"bash", ShellPosix, true},
		{"zsh", ShellPosix, true},
		{"sh", ShellPosix, true},
		{"fish", ShellFish, true},
		{"powershell", ShellPowerShell, true},
		{"pwsh", ShellPowerShell, true},
		{"cmd", ShellCmd, true},
		{"unknown-shell", ShellPosix, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shell, ok := ClassifyShellName(tt.name)
			assert.Equal(t, tt.expected, shell)
			assert.Equal(t, tt.ok, ok)
		})
	}

	// .exe suffix cases only apply on Windows
	if runtime.GOOS == "windows" {
		winTests := []struct {
			name     string
			expected ShellType
			ok       bool
		}{
			{"powershell.exe", ShellPowerShell, true},
			{"pwsh.exe", ShellPowerShell, true},
			{"cmd.exe", ShellCmd, true},
			{"explorer.exe", ShellPosix, false},
		}

		for _, tt := range winTests {
			t.Run(tt.name, func(t *testing.T) {
				shell, ok := ClassifyShellName(tt.name)
				assert.Equal(t, tt.expected, shell)
				assert.Equal(t, tt.ok, ok)
			})
		}
	}
}
