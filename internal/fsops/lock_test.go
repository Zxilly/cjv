package fsops

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLockFileSerializesAndSupportsCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	first, err := LockFile(t.Context(), path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = first.Close() })

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_, err = LockFile(ctx, path)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	// A different home does not have to wait for this lock.
	other, err := LockFile(t.Context(), filepath.Join(t.TempDir(), "lock"))
	require.NoError(t, err)
	require.NoError(t, other.Close())
	require.NoError(t, first.Close())

	next, err := LockFile(t.Context(), path)
	require.NoError(t, err)
	require.NoError(t, next.Close())
	require.FileExists(t, path, "the lock inode must remain stable between holders")
}
