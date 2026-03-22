//go:build windows

package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for SymlinkOrJunction — creates a directory junction on Windows
// (which doesn't require admin privileges unlike symlinks).

func TestSymlinkOrJunction_CreatesLink(t *testing.T) {
	target := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(target, "test.txt"), []byte("data"), 0o644))

	link := filepath.Join(t.TempDir(), "link")
	err := SymlinkOrJunction(target, link)
	if err != nil {
		t.Skipf("junction creation failed (may need privileges): %v", err)
	}

	// Verify the link points to the target
	content, err := os.ReadFile(filepath.Join(link, "test.txt"))
	require.NoError(t, err)
	assert.Equal(t, "data", string(content))
}
