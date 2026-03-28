package env

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyShellName(t *testing.T) {
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
		{"powershell.exe", ShellPowerShell, true},
		{"pwsh.exe", ShellPowerShell, true},
		{"cmd.exe", ShellCmd, true},
		{"cmd", ShellCmd, true},
		{"unknown-shell", ShellPosix, false},
		{"explorer.exe", ShellPosix, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shell, ok := ClassifyShellName(tt.name)
			assert.Equal(t, tt.expected, shell)
			assert.Equal(t, tt.ok, ok)
		})
	}
}
