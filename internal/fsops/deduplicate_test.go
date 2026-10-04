package fsops

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeduplicationLinksOnlyIdenticalPayloadAndCopiesMetadata(t *testing.T) {
	base := t.TempDir()
	source, dest := filepath.Join(base, "source"), filepath.Join(base, "dest")
	for _, root := range []string{source, dest} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, ".cjv"), 0o755))
		for _, name := range []string{"compiler", "different", ".cjv/record"} {
			require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte("same"), 0o644))
		}
	}
	require.NoError(t, os.WriteFile(filepath.Join(source, "different"), []byte("edit"), 0o644))
	require.NoError(t, HardlinkIdenticalTree(source, dest))
	for _, name := range []string{"compiler", "different", ".cjv/record"} {
		a, err := os.Stat(filepath.Join(source, name))
		require.NoError(t, err)
		b, err := os.Stat(filepath.Join(dest, name))
		require.NoError(t, err)
		assert.Equal(t, name == "compiler", os.SameFile(a, b), name)
	}
	// Replacements break the link; removing one installation preserves the other.
	require.NoError(t, WriteFileAtomic(filepath.Join(dest, "compiler"), []byte("new"), 0o644))
	require.NoError(t, os.RemoveAll(dest))
	data, err := os.ReadFile(filepath.Join(source, "compiler"))
	require.NoError(t, err)
	assert.Equal(t, "same", string(data))
}

func TestUnsupportedHardlinksRetainIndependentCopies(t *testing.T) {
	root := t.TempDir()
	source, dest := filepath.Join(root, "source"), filepath.Join(root, "dest")
	for _, dir := range []string{source, dest} {
		require.NoError(t, os.Mkdir(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "sdk"), []byte("payload"), 0o644))
	}
	require.NoError(t, hardlinkIdenticalTree(source, dest, func(string, string) error { return errors.New("cross-device link") }))
	a, err := os.Stat(filepath.Join(source, "sdk"))
	require.NoError(t, err)
	b, err := os.Stat(filepath.Join(dest, "sdk"))
	require.NoError(t, err)
	assert.False(t, os.SameFile(a, b))
	data, err := os.ReadFile(filepath.Join(dest, "sdk"))
	require.NoError(t, err)
	assert.Equal(t, "payload", string(data))
}
