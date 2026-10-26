package fsops

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type cancelOnSecondRead struct {
	reader io.Reader
	cancel context.CancelFunc
	reads  int
}

func (r *cancelOnSecondRead) Read(p []byte) (int, error) {
	r.reads++
	if r.reads == 2 {
		r.cancel()
	}
	return r.reader.Read(p)
}

func TestAtomicCopyCancellationPreservesDestination(t *testing.T) {
	directory := t.TempDir()
	dest := filepath.Join(directory, "archive")
	require.NoError(t, os.WriteFile(dest, []byte("previous contents"), 0o600))
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	src := &cancelOnSecondRead{reader: bytes.NewReader(make([]byte, 128<<10)), cancel: cancel}
	err := writeFileAtomicContext(ctx, dest, 0o600, src)
	require.ErrorIs(t, err, context.Canceled)
	require.GreaterOrEqual(t, src.reads, 2, "cancellation must happen after copying starts")
	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	require.Equal(t, "previous contents", string(data))
	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	require.Len(t, entries, 1, "the incomplete temporary file must be removed")
}

func TestCopyContextCancellationStopsHashing(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	payload := make([]byte, 128<<10)
	src := &cancelOnSecondRead{reader: bytes.NewReader(payload), cancel: cancel}
	n, err := CopyContext(ctx, io.Discard, src)
	require.ErrorIs(t, err, context.Canceled)
	require.Positive(t, n)
	require.Less(t, n, int64(len(payload)))
}

func TestCopyFileAtomicContextPublishesCompleteFile(t *testing.T) {
	directory := t.TempDir()
	src, dest := filepath.Join(directory, "source"), filepath.Join(directory, "dest")
	require.NoError(t, os.WriteFile(src, []byte("complete contents"), 0o600))
	require.NoError(t, os.WriteFile(dest, []byte("previous contents"), 0o600))
	require.NoError(t, CopyFileAtomicContext(t.Context(), src, dest, 0o600))
	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	require.Equal(t, "complete contents", string(data))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, CopyFileAtomicContext(ctx, src, dest, 0o600), context.Canceled)
	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	require.Len(t, entries, 2)
}
