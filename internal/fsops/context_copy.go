package fsops

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// CopyContext checks for cancellation between reads, including when src or dst
// has an optimized copy implementation. An in-progress filesystem call must
// return before cancellation can be observed.
func CopyContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	n, err := io.Copy(dst, contextReader{ctx: ctx, reader: src})
	if err != nil {
		return n, err
	}
	return n, ctx.Err()
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

// CopyFileAtomicContext copies to a sibling temporary file and publishes only
// after a complete, uncanceled copy. Existing dst contents survive failure.
func CopyFileAtomicContext(ctx context.Context, src, dst string, mode os.FileMode) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck // read-only
	return writeFileAtomicContext(ctx, dst, mode, in)
}

// WriteFileAtomicContext publishes bytes with the same cancellation and
// temporary-file ownership rules as CopyFileAtomicContext.
func WriteFileAtomicContext(ctx context.Context, dst string, data []byte, mode os.FileMode) error {
	return writeFileAtomicContext(ctx, dst, mode, bytes.NewReader(data))
}

func writeFileAtomicContext(ctx context.Context, dst string, mode os.FileMode, src io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), "."+filepath.Base(dst)+".partial-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) //nolint:errcheck // also runs after successful rename
	_, err = CopyContext(ctx, tmp, src)
	if err == nil {
		err = tmp.Chmod(mode)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if err = errors.Join(err, tmp.Close()); err != nil {
		return err
	}
	return Retry(func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return os.Rename(tmpPath, dst)
	})
}
