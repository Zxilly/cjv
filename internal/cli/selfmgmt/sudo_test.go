//go:build windows

package selfmgmt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Tests for CheckSudoSafety — prevents accidental execution with elevated
// privileges on Windows (running as admin when not needed).

func TestCheckSudoSafety_DoesNotPanic(t *testing.T) {
	// CheckSudoSafety should never panic regardless of privilege level
	assert.NotPanics(t, func() { CheckSudoSafety() })
}
