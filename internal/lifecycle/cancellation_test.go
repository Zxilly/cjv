package lifecycle

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/selfupdate"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
)

func TestUpdateCancellationReleasesHomeWhileSelfUpdateIsLocked(t *testing.T) {
	home := updateHome(t)
	selectRelease(t, "1.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, Options{}))
	selectRelease(t, "2.0.0")
	managedLock, err := selfupdate.LockManaged(t.Context())
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	staged := make(chan struct{})
	afterStagingHook = func() error { close(staged); return nil }
	done := make(chan error, 1)
	finished := false
	name := parse(t, "sts")
	defer func() {
		cancel()
		_ = managedLock.Close()
		if !finished {
			<-done
		}
		afterStagingHook = nil
	}()
	go func() {
		_, err := UpdateInstalled(ctx, name, Options{})
		done <- err
	}()
	select {
	case <-staged:
	case err := <-done:
		finished = true
		t.Fatalf("update stopped before staging: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("update did not reach staging")
	}
	// Publication has no further I/O to wait on until managed finalization.
	// Let it reach the lock held by the unrelated updater before canceling.
	select {
	case err := <-done:
		finished = true
		t.Fatalf("update bypassed the managed lock: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	cancel()
	select {
	case err := <-done:
		finished = true
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("canceled update still waits for an unrelated self-update")
	}
	record, err := toolchain.ReadInstallation(filepath.Join(home, "toolchains", "sts"))
	require.NoError(t, err)
	require.Equal(t, "sts-1.0.0", record.Release)
	history, err := ListHistory(t.Context(), "sts")
	require.NoError(t, err)
	require.Empty(t, history, "canceled publication must also roll back its snapshot")
	_, settings, err := config.LoadDefaultSettings()
	require.NoError(t, err)
	require.Equal(t, "sts", settings.DefaultToolchain)

	// The unrelated updater remains locked; installation and home locks must
	// nevertheless be available to another operation after cancellation.
	lockCtx, stop := context.WithTimeout(t.Context(), time.Second)
	defer stop()
	homeLock, err := toolchain.LockHome(lockCtx)
	require.NoError(t, err)
	require.NoError(t, homeLock.Close())
	downloads, err := config.DownloadsDir()
	require.NoError(t, err)
	preparation, err := dist.BeginPreparation(lockCtx, downloads)
	require.NoError(t, err)
	require.NoError(t, preparation.Close())
}
