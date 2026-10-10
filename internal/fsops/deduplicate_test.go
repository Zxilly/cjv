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
	canLink := hardlinksAvailable(t, base)
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
		assert.Equal(t, canLink && name == "compiler", os.SameFile(a, b), name)
	}
	// Replacements break the link; removing one installation preserves the other.
	require.NoError(t, WriteFileAtomic(filepath.Join(dest, "compiler"), []byte("new"), 0o644))
	require.NoError(t, os.RemoveAll(dest))
	data, err := os.ReadFile(filepath.Join(source, "compiler"))
	require.NoError(t, err)
	assert.Equal(t, "same", string(data))
}

// HDC's SELinux shell domain can deny hard links while permitting symlinks.
// Only permission failures are treated as a capability limitation.
func hardlinksAvailable(t *testing.T, dir string) bool {
	t.Helper()
	src, dst := filepath.Join(dir, "hardlink-probe"), filepath.Join(dir, "hardlink-probe-link")
	require.NoError(t, os.WriteFile(src, nil, 0o600))
	defer os.Remove(src)
	defer os.Remove(dst)
	err := os.Link(src, dst)
	if errors.Is(err, os.ErrPermission) {
		return false
	}
	require.NoError(t, err)
	return true
}

func TestDeduplicationSkipsModifiedSourceDirectoryLinks(t *testing.T) {
	base := t.TempDir()
	source, staged, outside := filepath.Join(base, "source"), filepath.Join(base, "staged"), filepath.Join(base, "outside")
	for _, dir := range []string{source, staged, outside} {
		require.NoError(t, os.Mkdir(dir, 0o755))
	}
	require.NoError(t, os.Mkdir(filepath.Join(staged, "lib"), 0o755))
	stagedFile, outsideFile := filepath.Join(staged, "lib", "payload"), filepath.Join(outside, "payload")
	require.NoError(t, os.WriteFile(stagedFile, []byte("original"), 0o644))
	require.NoError(t, os.WriteFile(outsideFile, []byte("edited"), 0o644))
	if err := SymlinkOrJunction(outside, filepath.Join(source, "lib")); err != nil {
		t.Skipf("directory links unavailable: %v", err)
	}
	require.NoError(t, HardlinkIdenticalTree(source, staged))
	data, err := os.ReadFile(stagedFile)
	require.NoError(t, err)
	assert.Equal(t, "original", string(data))
	a, err := os.Stat(stagedFile)
	require.NoError(t, err)
	b, err := os.Stat(outsideFile)
	require.NoError(t, err)
	assert.False(t, os.SameFile(a, b))
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
