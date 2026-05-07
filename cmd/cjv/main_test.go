package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMainHelpReturnsNormally(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"cjv", "--help"}
	t.Cleanup(func() { os.Args = oldArgs })

	require.NotPanics(t, main)
}

func TestIsInitInvocation(t *testing.T) {
	assert.True(t, isInitInvocation("cjv-init"))
	assert.True(t, isInitInvocation("cjv-init(1)"))
	assert.True(t, isInitInvocation("cjv-init-2"))
	assert.True(t, isInitInvocation("cjv-setup"))
	assert.False(t, isInitInvocation("cjv"))
	assert.False(t, isInitInvocation("cjv-mirror"))
	assert.False(t, isInitInvocation("cjc"))
}
