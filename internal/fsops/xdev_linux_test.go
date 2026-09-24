//go:build linux

package fsops

import (
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

// crossDeviceDir returns a scratch directory on a different filesystem from
// t.TempDir(), or skips the test when none is available.
func crossDeviceDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/dev/shm", "cjv-fsops-*")
	if err != nil {
		t.Skipf("no writable tmpfs: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	if deviceOf(t, dir) == deviceOf(t, t.TempDir()) {
		t.Skip("tmpfs shares a device with the temp directory")
	}
	return dir
}

func deviceOf(t *testing.T, path string) uint64 {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	stat, ok := info.Sys().(*syscall.Stat_t)
	require.True(t, ok)
	return uint64(stat.Dev) //nolint:unconvert // Dev is int32 on some platforms
}
