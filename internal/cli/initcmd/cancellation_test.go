package initcmd

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/Zxilly/cjv/internal/selfupdate"
	"github.com/stretchr/testify/require"
)

func TestCanceledInitDoesNotWaitForSelfUpdateOrPublishBinary(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	t.Setenv(config.EnvNoPathSetup, "1")
	lock, err := selfupdate.LockManaged(t.Context())
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	c := newTestCommands()
	var output bytes.Buffer
	done := make(chan error, 1)
	finished := false
	defer func() {
		_ = lock.Close()
		if !finished {
			<-done
		}
	}()
	go func() {
		done <- c.install(ctx, initConsole{out: &output, err: &output}, home, initCustomizeOptions{home: home, toolchain: "none"})
	}()
	select {
	case err := <-done:
		finished = true
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("canceled init waited for the active self-update")
	}
	require.NoFileExists(t, filepath.Join(home, "bin", sdktools.CjvBinaryName()))
	require.Empty(t, output.String(), "canceled setup must not report successful initialization")
}
