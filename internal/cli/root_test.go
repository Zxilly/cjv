package cli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test for Execute — the root command entry point.

func TestExecute_Help(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"cjv", "--help"}
	defer func() { os.Args = oldArgs }()

	err := Execute("dev", "dev")
	assert.NoError(t, err)
}
