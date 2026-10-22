package selfupdate

import (
	"context"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/stretchr/testify/require"
)

func TestManagedUpdateLockExcludesCleanupAndReplacement(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	first, err := LockManaged(t.Context())
	require.NoError(t, err)
	defer first.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 80*time.Millisecond)
	defer cancel()
	_, err = LockManaged(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.NoError(t, first.Close())
	next, err := LockManaged(t.Context())
	require.NoError(t, err)
	require.NoError(t, next.Close())
}
