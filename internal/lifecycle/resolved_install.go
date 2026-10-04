package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/fsops"
	"github.com/Zxilly/cjv/internal/fstx"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/reachable"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// ResolvedToolchain holds the result of toolchain resolution.
type ResolvedToolchain struct {
	Name        string
	URL         string
	SHA256      string
	ArchiveName string
	Tuple       string
}

// acquisition is how one SDK archive reaches the downloads area and how it
// becomes the staged SDK tree. It is the only part of placing a toolchain
// that differs between a manifest release, a CI bundle fetched from a URL
// and a local archive; everything around it is placeToolchain.
type acquisition struct {
	// fetch returns the archive path and whether cjv owns that file, in which
	// case it is removed once the toolchain is committed. A failed placement
	// keeps it for the next retry.
	fetch func(ctx context.Context, downloadsDir string) (archivePath string, owned bool, err error)
	// extract materializes the SDK tree at stagingDir from archivePath.
	extract func(ctx context.Context, archivePath, stagingDir string) error
	prepare func(context.Context, component.Roots) error
}

// installResolved places one manifest release. setDefault allows the first
// host toolchain to become the default; target variants pass false. An
// already installed release without force is reported and kept.
func installResolved(ctx context.Context, d *Distribution, rt ResolvedToolchain, force, setDefault bool, opts Options) error {
	identity := rt.Name
	if opts.selection != "" {
		identity = opts.selection
	}
	acq := acquisition{
		fetch: func(ctx context.Context, downloadsDir string) (string, bool, error) {
			if u, err := url.Parse(rt.URL); err != nil || u.Path == "" {
				return "", false, fmt.Errorf("invalid toolchain download URL: %s", rt.URL)
			}
			path, err := dist.DownloadCachedWithName(ctx, rt.URL, rt.SHA256, downloadsDir, rt.ArchiveName, opts.sink())
			return path, true, err
		},
		extract: dist.InstallSDK,
		prepare: func(ctx context.Context, roots component.Roots) error {
			if err := toolchain.WriteInstallation(roots.TcDir, toolchain.Installation{Release: rt.Name, Tuple: rt.Tuple, SHA256: rt.SHA256}); err != nil {
				return err
			}
			if opts.prepare != nil {
				return opts.prepare(ctx, roots)
			}
			return nil
		},
	}
	var publish func() error
	if setDefault {
		publish = func() error { return publishFirstDefault(d, identity, opts) }
	}
	err := placeToolchain(ctx, identity, force, rt.Tuple, acq, publish, opts)
	var already *cjverr.ToolchainAlreadyInstalledError
	if errors.As(err, &already) {
		opts.emit(progress.Event{Kind: progress.ToolchainAlreadyInstalled, Toolchain: identity})
		if publish != nil {
			lock, err := toolchain.LockHome(ctx)
			if err != nil {
				return err
			}
			defer lock.Close() //nolint:errcheck
			if err := lock.Recover(); err != nil {
				return err
			}
			path, err := config.ToolchainDirFor(identity)
			if err != nil {
				return err
			}
			if _, err := os.Stat(path); err != nil {
				return err
			}
			return publish()
		}
		return nil
	}
	return err
}

// placeToolchain prepares a private SDK tree under downloads/, then publishes
// it under the home mutation lock. Downloads and decompression do not block
// proxy recovery; a separate install lock protects the shared archive cache.
// The destination is rechecked after preparation. Staging and all journal
// changes stay locked until commit/rollback and residue cleanup finish.
func placeToolchain(ctx context.Context, name string, force bool, tuple string, acq acquisition, publish func() error, opts Options) (retErr error) {
	parsed, err := toolchain.ParseToolchainName(name)
	if err != nil {
		return err
	}
	name = parsed.String()
	if err := config.EnsureDirs(); err != nil {
		return err
	}
	tcDir, err := config.ToolchainsDir()
	if err != nil {
		return err
	}
	// Always take the install lock before the home lock. Recovery, links and
	// removal only need the home lock, so they can proceed during downloads.
	installLock, err := fsops.LockFile(ctx, filepath.Join(filepath.Dir(tcDir), ".install.lock"))
	if err != nil {
		return err
	}
	defer installLock.Close() //nolint:errcheck // protect cache consumption and cleanup
	destDir := filepath.Join(tcDir, name)
	lock, _, err := lockPlacement(ctx, destDir, force)
	if err != nil {
		return err
	}
	if err := lock.Close(); err != nil {
		return err
	}

	downloadsDir, err := config.DownloadsDir()
	if err != nil {
		return err
	}
	archivePath, owned, err := acq.fetch(ctx, downloadsDir)
	if err != nil {
		return err
	}
	if owned {
		defer func() {
			if retErr == nil {
				_ = dist.CleanupDownload(archivePath) //nolint:errcheck // best-effort
			}
		}()
	}
	preparedRoot, err := os.MkdirTemp(downloadsDir, ".cjv-stage-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(preparedRoot) //nolint:errcheck // private extraction scratch
	preparedDir := filepath.Join(preparedRoot, "sdk")
	opts.emit(progress.Event{Kind: progress.Extracting})
	if err := acq.extract(ctx, archivePath, preparedDir); err != nil {
		return err
	}
	if err := validateInstallation(preparedDir, tuple); err != nil {
		return err
	}
	preparedRoots := component.Roots{TcDir: preparedDir, DocsDir: filepath.Join(preparedRoot, "docs"), StdxDir: filepath.Join(preparedRoot, "stdx")}
	if acq.prepare != nil {
		if err := acq.prepare(ctx, preparedRoots); err != nil {
			return err
		}
	}

	lock, isReinstall, err := lockPlacement(ctx, destDir, force)
	if err != nil {
		return err
	}
	defer lock.Close() //nolint:errcheck // release after staging and transaction cleanup
	if err := validateDependencies(opts); err != nil {
		return err
	}
	if opts.expectedSet {
		current, err := toolchain.ReadInstallation(destDir)
		if opts.expected == nil {
			if _, err := os.Lstat(destDir); !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("toolchain %s changed during installation; retry", name)
			}
		} else if state, stateErr := installationComponentState(destDir); err != nil || current != *opts.expected || stateErr != nil || state != opts.expectedComponents {
			return fmt.Errorf("toolchain %s changed during installation; retry", name)
		}
	}
	if acq.prepare != nil {
		return publishOfficial(preparedRoots, name, isReinstall, publish, opts)
	}
	stagingDir := config.StagingDir(destDir)
	defer func() {
		var recoveryErr *fstx.RecoveryError
		if retErr != nil && !errors.As(retErr, &recoveryErr) {
			_ = fsops.RemoveAllRetry(stagingDir) //nolint:errcheck // best-effort
		}
	}()
	if err := fsops.RemoveAllRetry(stagingDir); err != nil {
		return fmt.Errorf("failed to clean staging directory: %w", err)
	}
	if err := fsops.RenameRetry(preparedDir, stagingDir); err != nil {
		// downloads/ may be on a different volume from toolchains/.
		if _, err := fsops.MoveTree(preparedDir, stagingDir); err != nil {
			return fmt.Errorf("failed to stage SDK: %w", err)
		}
	}
	if afterStagingHook != nil {
		if err := afterStagingHook(); err != nil {
			return err
		}
	}
	if err := swapInstalledToolchain(stagingDir, destDir, isReinstall, finalizeInstalledToolchain, publish); err != nil {
		return err
	}
	opts.emit(progress.Event{Kind: progress.ToolchainInstalled, Toolchain: name})
	return nil
}

// lockPlacement recovers and checks the destination atomically with respect
// to other home mutations. A successful caller owns the returned lock.
func lockPlacement(ctx context.Context, destDir string, force bool) (*toolchain.HomeLock, bool, error) {
	lock, err := toolchain.LockHome(ctx)
	if err != nil {
		return nil, false, err
	}
	if err := lock.Recover(); err != nil {
		_ = lock.Close() //nolint:errcheck // preserve the recovery error
		return nil, false, err
	}
	if _, err := os.Stat(destDir); err == nil {
		if !force {
			_ = lock.Close() //nolint:errcheck // release on refused placement
			return nil, false, &cjverr.ToolchainAlreadyInstalledError{Name: filepath.Base(destDir)}
		}
		return lock, true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		_ = lock.Close() //nolint:errcheck // preserve the stat error
		return nil, false, fmt.Errorf("failed to stat %s: %w", destDir, err)
	}
	return lock, false, nil
}

func defaultToolchainExists(name string) bool {
	parsed, err := toolchain.ParseToolchainName(name)
	if err != nil {
		return false
	}
	_, err = toolchain.FindInstalled(parsed)
	return err == nil
}

// validateInstallation checks that the extracted SDK carries the compiler at
// the place the proxy will look for it. tuple names the SDK's platform so a
// cross-target SDK is checked against its own executable naming rather than
// the running OS; empty means the host.
func validateInstallation(dir, tuple string) error {
	var err error
	if tuple == "" {
		_, err = sdktools.ResolveInstalledToolBinary(dir, "cjc")
	} else {
		_, err = sdktools.ResolveInstalledToolBinaryForTuple(dir, "cjc", tuple)
	}
	if err != nil {
		return fmt.Errorf("installation validation failed: %w", err)
	}
	return nil
}

// finalizeInstalledToolchain runs once the new toolchain is in place and before
// the transaction commits: the running cjv becomes the managed binary under
// CJV_HOME/bin and every SDK tool gets its proxy link, so the toolchain is
// reachable through the proxies whether it was installed by `cjv install` or
// by proxy auto-install.
func finalizeInstalledToolchain() error {
	if err := reachable.Ensure(reachable.Policy{}); err != nil {
		return err
	}
	if afterFinalizeHook != nil {
		return afterFinalizeHook()
	}
	return nil
}

// afterStagingHook lets tests pause once the staging tree is materialized.
// Production never sets it.
var afterStagingHook func() error

// afterFinalizeHook lets tests observe or fail the window between placing the
// toolchain and committing the transaction. Production never sets it.
var afterFinalizeHook func() error

// afterPublishHook lets tests observe or interrupt the window between
// publishing the first default toolchain and completing the transaction.
// Production never sets it.
var afterPublishHook func() error

func swapInstalledToolchain(stagingDir, destDir string, isReinstall bool, afterSwap, publish func() error) (err error) {
	if isReinstall {
		if err := preserveComponentMetadata(stagingDir, destDir); err != nil {
			return err
		}
	}
	tx, txErr := fstx.NewTransaction(destDir)
	if txErr != nil {
		return fmt.Errorf("failed to begin install transaction: %w", txErr)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		if rbErr := tx.Rollback(); rbErr != nil {
			err = errors.Join(err, fmt.Errorf("rollback after failed install also failed: %w", rbErr))
		}
	}()

	if isReinstall {
		if err := tx.RemoveDir(destDir); err != nil {
			return fmt.Errorf("failed to remove existing toolchain: %w", err)
		}
	}
	if err := tx.RenameFile(stagingDir, destDir); err != nil {
		return fmt.Errorf("failed to place new toolchain: %w", err)
	}
	if err := afterSwap(); err != nil {
		return fmt.Errorf("failed to finalize installation: %w", err)
	}
	publicationErr, err := commitInstallation(tx, publish)
	if err != nil {
		return err
	}
	committed = true
	return publicationErr
}

// publishFirstDefault runs under the home lock and restores its settings if
// publication fails. It also serves already-installed SDKs without downloads.
func publishFirstDefault(d *Distribution, identity string, opts Options) error {
	d.File.Invalidate()
	before, err := d.File.Load()
	if err != nil {
		return err
	}
	if before.DefaultToolchain == "" || !defaultToolchainExists(before.DefaultToolchain) {
		if _, err := d.File.Update(config.SettingsUpdate{DefaultToolchain: &identity}); err != nil {
			return err
		}
		if opts.ConfigurePath {
			reachable.ConfigurePath()
		}
	}
	if afterPublishHook != nil {
		if err := afterPublishHook(); err != nil {
			if restoreErr := d.File.Save(before); restoreErr != nil {
				return &publishedSettingsError{errors.Join(err, fmt.Errorf("restore toolchain settings: %w", restoreErr))}
			}
			return err
		}
	}
	return nil
}
