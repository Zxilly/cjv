package toolchain

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/fsops"
)

// HomeLock serializes recovery and lifecycle mutations in one CJV_HOME.
// It is not reentrant: a holder must use Recover instead of RecoverHome.
// Keep it until staging cleanup and transaction commit/rollback are finished.
type HomeLock struct {
	file      *os.File
	lookup    *os.File
	tcDir     string
	recovered bool
}

const (
	homeLockName   = ".lifecycle.lock"
	lookupLockName = ".lifecycle.lookup.lock"
	readPhaseName  = ".lifecycle.readable"
)

func LockHome(ctx context.Context) (*HomeLock, error) {
	home, err := config.Home()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return nil, err
	}
	file, err := fsops.LockFile(ctx, filepath.Join(home, homeLockName))
	if err != nil {
		return nil, err
	}
	lookup, err := fsops.LockFile(ctx, filepath.Join(home, lookupLockName))
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return finishHomeLock(home, file, lookup)
}

func finishHomeLock(home string, file, lookup *os.File) (*HomeLock, error) {
	lock := &HomeLock{file: file, lookup: lookup, tcDir: filepath.Join(home, config.ToolchainsSubdir)}
	// A crash may leave the read-phase marker behind. Every writer clears it
	// while holding the lookup gate, before recovery or any installed mutation.
	if err := lock.clearReadPhase(); err != nil {
		return nil, errors.Join(err, lock.Close())
	}
	return lock, nil
}

func (lock *HomeLock) Close() error {
	if lock == nil {
		return nil
	}
	var errs []error
	if lock.lookup != nil {
		errs = append(errs, lock.lookup.Close())
		lock.lookup = nil
	}
	if lock.file != nil {
		errs = append(errs, lock.file.Close())
		lock.file = nil
	}
	return errors.Join(errs...)
}

// WithStableReads permits active-toolchain lookups during expensive work that
// leaves every installed root unchanged. The writer lock stays held, so no other
// lifecycle mutation or recovery can interfere with a snapshot. Call only after
// Recover and outside a transaction's mutation/rollback interval.
//
// Readers hold the lookup gate through FindActiveDir. The writer reacquires it
// and clears the marker before returning to its caller, even on cancellation.
// Thus publication cannot overlap a lookup. Readers also verify that a live
// writer holds the home lock; a stale marker after a crash never bypasses recovery.
func (lock *HomeLock) WithStableReads(ctx context.Context, work func() error) (retErr error) {
	if lock == nil || lock.file == nil || lock.lookup == nil || !lock.recovered {
		return fmt.Errorf("stable reads require a recovered home lock")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	home := filepath.Dir(lock.tcDir)
	// Before legacy migration is complete, lookup may need to materialize a
	// channel. Keep the ordinary recovery path for that first invocation.
	if _, err := os.Stat(filepath.Join(home, ".cjv", "layout-v2")); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return work()
		}
		return err
	}
	if err := fsops.WriteFileAtomic(filepath.Join(home, readPhaseName), []byte("1\n"), 0o600); err != nil {
		return err
	}
	if err := lock.lookup.Close(); err != nil {
		return err
	}
	lock.lookup = nil
	defer func() {
		// Cancellation must not return control to a publisher without its gate.
		lookup, err := fsops.LockFile(context.WithoutCancel(ctx), filepath.Join(home, lookupLockName))
		if err == nil {
			lock.lookup = lookup
			err = lock.clearReadPhase()
		}
		retErr = errors.Join(retErr, err)
	}()
	return work()
}

func (lock *HomeLock) clearReadPhase() error {
	path := filepath.Join(filepath.Dir(lock.tcDir), readPhaseName)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode().IsRegular() && info.Size() == 2 {
		if data, err := os.ReadFile(path); err == nil && string(data) == "0\n" {
			return nil
		}
	}
	return fsops.WriteFileAtomic(path, []byte("0\n"), 0o600)
}

func readablePhase(home string) bool {
	path := filepath.Join(home, readPhaseName)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() != 2 {
		return false
	}
	data, err := os.ReadFile(path)
	return err == nil && string(data) == "1\n"
}

// lockActiveLookup returns either the ordinary home lock, or just a lookup gate
// during a live writer's stable read phase. It never waits for home while holding
// the gate: writers always acquire home first, then the gate.
func lockActiveLookup(ctx context.Context) (*HomeLock, *os.File, error) {
	home, err := config.Home()
	if err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return nil, nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	file, locked, err := fsops.TryLockFile(filepath.Join(home, homeLockName))
	if err != nil {
		return nil, nil, err
	}
	lookup, err := fsops.LockFile(ctx, filepath.Join(home, lookupLockName))
	if err != nil {
		if locked {
			_ = file.Close()
		}
		return nil, nil, err
	}
	if !locked {
		// A writer may have finished (or crashed) while this reader waited for
		// its gate. Recheck liveness before trusting the read-phase marker.
		file, locked, err = fsops.TryLockFile(filepath.Join(home, homeLockName))
		if err != nil {
			_ = lookup.Close()
			return nil, nil, err
		}
	}
	if locked {
		lock, err := finishHomeLock(home, file, lookup)
		return lock, nil, err
	}
	if readablePhase(home) {
		return nil, lookup, nil
	}
	if err := lookup.Close(); err != nil {
		return nil, nil, err
	}
	lock, err := LockHome(ctx)
	return lock, nil, err
}
