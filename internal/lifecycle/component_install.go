package lifecycle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// InstallComponentsForToolchain backs the proxy auto_install path: it resolves
// tcInput to an already-installed toolchain and installs the missing
// components, reporting progress to the caller's sink like any install.
func InstallComponentsForToolchain(ctx context.Context, tcInput string, components []string, opts Options) error {
	if len(components) == 0 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := toolchain.RecoverHomeContext(ctx); err != nil {
		return err
	}
	name, err := toolchain.ParseToolchainName(tcInput)
	if err != nil {
		return err
	}
	installedDir, err := toolchain.FindInstalled(name)
	if err != nil {
		return err
	}
	return InstallComponents(ctx, filepath.Base(installedDir), components, false, opts)
}

// InstallComponents installs components into the installed toolchain named
// resolvedName ("<channel>-<version>" or a target variant). The configured
// distribution source resolves component artifacts for every channel; a
// target variant downloads its own target stdx. An already installed
// component is reported and kept unless force is set.
func InstallComponents(ctx context.Context, resolvedName string, components []string, force bool, opts Options) error {
	resolvedTC, err := toolchain.ParseToolchainName(resolvedName)
	if err != nil {
		return err
	}
	if resolvedTC.IsCustom() {
		return &cjverr.ComponentRequiresHostError{Component: strings.Join(components, ", ")}
	}
	d, err := OpenDistribution(opts)
	if err != nil {
		return err
	}
	return installComponents(ctx, d, resolvedName, components, force, opts)
}

func installComponents(ctx context.Context, d *Distribution, resolvedName string, components []string, force bool, opts Options) error {
	if ctx == nil {
		ctx = context.Background()
	}
	parsed, err := component.NormalizeList(components)
	if err != nil {
		return err
	}
	home, err := config.Home()
	if err != nil {
		return err
	}
	// Follow SDK placement's lock order. Serialize shared download cache access,
	// but release the home lock throughout network requests and extraction.
	downloadsDir, err := config.DownloadsDir()
	if err != nil {
		return err
	}
	preparation, err := dist.BeginPreparation(ctx, downloadsDir)
	if err != nil {
		return err
	}
	defer preparation.Close() //nolint:errcheck // best-effort scratch cleanup
	lock, err := toolchain.LockHome(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if lock != nil {
			_ = lock.Close()
		}
	}()
	if err := lock.Recover(); err != nil {
		return err
	}
	if err := lock.MigrateLegacy(); err != nil {
		return err
	}
	snapshot, err := readInstallationSnapshot(resolvedName)
	if err != nil {
		return err
	}
	if snapshot.record == nil {
		return os.ErrNotExist
	}
	roots := snapshot.roots
	resolvedTC, err := toolchain.ParseToolchainName(snapshot.record.Release)
	if err != nil {
		return err
	}
	if resolvedTC.IsCustom() {
		return &cjverr.ComponentRequiresHostError{Component: strings.Join(components, ", ")}
	}
	tuple := snapshot.record.Tuple
	if tuple == "" {
		tuple = resolvedTC.Target
	}
	if tuple == "" {
		tuple = d.HostTuple
	}
	if err := snapshot.captureMetadata(); err != nil {
		return err
	}
	var selected []component.Name
	for _, c := range parsed {
		if !force && component.IsInstalled(roots.TcDir, c) {
			opts.emit(progress.Event{Kind: progress.ComponentAlreadyInstalled, Toolchain: resolvedName, Component: string(c)})
		} else {
			selected = append(selected, c)
		}
	}
	if len(selected) == 0 {
		return nil
	}
	if err := lock.Close(); err != nil {
		return err
	}
	lock = nil
	stage, err := preparation.TempDir(".cjv-components-*")
	if err != nil {
		return err
	}
	prepared := component.Roots{TcDir: filepath.Join(stage, "sdk"), DocsDir: filepath.Join(stage, "docs"), StdxDir: filepath.Join(stage, "stdx")}
	for _, c := range selected {
		d.note()
		if err := component.PrepareFromSource(ctx, prepared, resolvedTC, c, tuple, false, d.Source, preparation, opts.sink()); err != nil {
			return err
		}
	}
	lock, err = toolchain.LockHome(ctx)
	if err != nil {
		return err
	}
	if err := lock.Recover(); err != nil {
		return err
	}
	if !snapshot.matches() {
		return fmt.Errorf("toolchain %s changed during component installation; retry", resolvedName)
	}
	if err := publishComponentBatch(home, resolvedName, prepared, selected, force, opts); err != nil {
		return err
	}
	preparation.Complete()
	return nil
}
