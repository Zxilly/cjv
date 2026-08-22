package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/fstx"
	"github.com/Zxilly/cjv/internal/i18n"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/Zxilly/cjv/internal/utils"
)

// ResolvedToolchain holds the result of toolchain resolution.
type ResolvedToolchain struct {
	Name              string
	URL               string
	SHA256            string
	ArchiveName       string
	Tuple             string
	NightlyReleaseTag string
	NightlyVersion    string
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
	destDir := filepath.Join(tcDir, resolvedName)
	isReinstall := false
	if _, err := os.Stat(destDir); err == nil {
		if !force {
			if opts.json() {
				return &cjverr.ToolchainAlreadyInstalledError{Name: resolvedName}
			}
			fmt.Println(i18n.T("ToolchainAlreadyInstalled", i18n.MsgData{"Name": resolvedName}))
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

	stagingDir := destDir + toolchain.StagingSuffix
	if err := utils.RemoveAllRetry(stagingDir); err != nil {
		return fmt.Errorf("failed to clean staging directory: %w", err)
	}
	defer func() {
		if retErr != nil {
			_ = utils.RemoveAllRetry(stagingDir) //nolint:errcheck
		}
	}()

	opts.note(i18n.T("Extracting", nil))
	if err := dist.InstallSDK(ctx, archivePath, stagingDir); err != nil {
		return err
	}
	if err := opts.validateInstallation(stagingDir, rt.Tuple); err != nil {
		return err
	}
	if rt.NightlyReleaseTag != "" || rt.NightlyVersion != "" {
		if err := toolchain.WriteNightlyReleaseMetadata(stagingDir, toolchain.NightlyReleaseMetadata{
			ReleaseTag: rt.NightlyReleaseTag,
			Version:    rt.NightlyVersion,
		}); err != nil {
			return err
		}
	}

	isFirstInstall := allowDefault && (settings.DefaultToolchain == "" || !defaultToolchainExists(settings.DefaultToolchain))
	if err := swapInstalledToolchain(stagingDir, destDir, isReinstall, func() error {
		if err := opts.ensureManagedBinary(); err != nil {
			return err
		}
		if err := opts.createProxyLinks(); err != nil {
			return err
		}
		if isFirstInstall {
			settings.DefaultToolchain = resolvedName
			if err := sf.Save(settings); err != nil {
				return err
			}
			opts.ensurePathConfigured()
		}
		return nil
	}); err != nil {
		return err
	}

	opts.green("ToolchainInstalled", i18n.MsgData{"Name": resolvedName})
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

func validateInstallation(dir, tuple string) error {
	binary := filepath.Join(dir, "bin", "cjc")
	if tuple != "" {
		if id, err := sdktarget.ParseIdentity(tuple); err == nil && strings.HasPrefix(id.HostTuple(), "win32-") {
			binary += ".exe"
		}
	} else if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if _, err := os.Stat(binary); err != nil {
		return fmt.Errorf("installation validation failed: %w", err)
	}
	return nil
}

func swapInstalledToolchain(stagingDir, destDir string, isReinstall bool, afterSwap func() error) (err error) {
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
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// EnsurePathConfigured is the default lifecycle hook for first install. CLI
// adapters inject the real shell/registry writer; proxy auto-install leaves
// PATH alone because cjv is already reachable.
func EnsurePathConfigured() {
}
