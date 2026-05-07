package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMainHelpReturnsNormally(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"cjv", "--help"}
	t.Cleanup(func() { os.Args = oldArgs })

	require.NotPanics(t, main)
}
