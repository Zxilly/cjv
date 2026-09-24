package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/fstx"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/reachable"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/Zxilly/cjv/internal/utils"
)

// ResolvedToolchain holds the result of toolchain resolution.
type ResolvedToolchain struct {
	Name        string
	URL         string
	SHA256      string
	ArchiveName string
	Tuple       string
}

func InstallResolved(ctx context.Context, rt ResolvedToolchain, settings *config.Settings, sf *config.SettingsFile, force bool, opts Options) error {
	return installResolvedWithDefault(ctx, rt, settings, sf, force, true, opts)
}

func InstallResolvedNoDefault(ctx context.Context, rt ResolvedToolchain, settings *config.Settings, sf *config.SettingsFile, force bool, opts Options) error {
	return installResolvedWithDefault(ctx, rt, settings, sf, force, false, opts)
}

func installResolvedWithDefault(ctx context.Context, rt ResolvedToolchain, settings *config.Settings, sf *config.SettingsFile, force bool, allowDefault bool, opts Options) (retErr error) {
	resolvedName := rt.Name
	tcDir, err := config.ToolchainsDir()
	if err != nil {
		return err
	}
	if err := toolchain.RecoverHome(); err != nil {
		return err
	}
	destDir := filepath.Join(tcDir, resolvedName)
	isReinstall := false
	if _, err := os.Stat(destDir); err == nil {
		if !force {
			opts.report("ToolchainAlreadyInstalled", i18n.MsgData{"Name": resolvedName})
			return nil
		}
		isReinstall = true
	}

	if err := config.EnsureDirs(); err != nil {
		return err
	}

	downloadsDir, err := config.DownloadsDir()
	if err != nil {
		return err
	}
	if u, err := url.Parse(rt.URL); err != nil || u.Path == "" {
		return fmt.Errorf("invalid toolchain download URL: %s", rt.URL)
	}

	archivePath, err := dist.DownloadCachedWithName(ctx, rt.URL, rt.SHA256, downloadsDir, rt.ArchiveName)
	if err != nil {
		return err
	}
	defer func() {
		if retErr == nil {
			_ = dist.CleanupDownload(archivePath) //nolint:errcheck
		}
	}()

	stagingDir := config.StagingDir(destDir)
	if err := utils.RemoveAllRetry(stagingDir); err != nil {
		return fmt.Errorf("failed to clean staging directory: %w", err)
	}
	defer func() {
		var recoveryErr *fstx.RecoveryError
		if retErr != nil && !errors.As(retErr, &recoveryErr) {
			_ = utils.RemoveAllRetry(stagingDir) //nolint:errcheck
		}
	}()

	opts.report("Extracting", nil)
	if err := dist.InstallSDK(ctx, archivePath, stagingDir); err != nil {
		return err
	}
	if err := validateInstallation(stagingDir, rt.Tuple); err != nil {
		return err
	}
	isFirstInstall := allowDefault && (settings.DefaultToolchain == "" || !defaultToolchainExists(settings.DefaultToolchain))
	var publishDefault func() error
	if isFirstInstall {
		publishDefault = func() error {
			if _, err := sf.Update(config.SettingsUpdate{DefaultToolchain: &resolvedName}); err != nil {
				return err
			}
			settings.DefaultToolchain = resolvedName
			if opts.ConfigurePath {
				reachable.ConfigurePath()
			}
			if afterPublishHook != nil {
				return afterPublishHook()
			}
			return nil
		}
	}
	if err := swapInstalledToolchain(stagingDir, destDir, isReinstall, finalizeInstalledToolchain, publishDefault); err != nil {
		return err
	}

	opts.report("ToolchainInstalled", i18n.MsgData{"Name": resolvedName})
	return nil
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
	commit := tx.Commit
	if publish != nil {
		commit = func() error { return tx.CommitWith(publish) }
	}
	if err := commit(); err != nil {
		return err
	}
	committed = true
	return nil
}
