package fsops

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTryLockFileObservesOwnershipAndRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	first, locked, err := TryLockFile(path)
	require.NoError(t, err)
	require.True(t, locked)
	t.Cleanup(func() { _ = first.Close() })
	second, locked, err := TryLockFile(path)
	require.NoError(t, err)
	require.False(t, locked)
	require.Nil(t, second)
	require.NoError(t, first.Close())
	third, locked, err := TryLockFile(path)
	require.NoError(t, err)
	require.True(t, locked)
	require.NoError(t, third.Close())
	require.FileExists(t, path)
}
