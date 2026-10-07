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
	"github.com/Zxilly/cjv/internal/fsops"
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
	installLock, err := fsops.LockFile(ctx, filepath.Join(home, ".install.lock"))
	if err != nil {
		return err
	}
	defer installLock.Close() //nolint:errcheck
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
	roots, err := component.RootsFor(resolvedName)
	if err != nil {
		return err
	}
	resolvedTC, err := toolchain.InstalledRelease(roots.TcDir)
	if err != nil {
		return err
	}
	if resolvedTC.IsCustom() {
		return &cjverr.ComponentRequiresHostError{Component: strings.Join(components, ", ")}
	}
	record, err := toolchain.ReadInstallation(roots.TcDir)
	if err != nil {
		return err
	}
	tuple := record.Tuple
	if tuple == "" {
		tuple = resolvedTC.Target
	}
	if tuple == "" {
		tuple = d.HostTuple
	}
	state, err := installationComponentState(roots.TcDir)
	if err != nil {
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
	downloadsDir, err := config.DownloadsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(downloadsDir, 0o755); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(downloadsDir, ".cjv-components-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage) //nolint:errcheck // private extraction scratch
	prepared := component.Roots{TcDir: filepath.Join(stage, "sdk"), DocsDir: filepath.Join(stage, "docs"), StdxDir: filepath.Join(stage, "stdx")}
	for _, c := range selected {
		d.note()
		if err := component.InstallFromSource(ctx, prepared, resolvedTC, c, tuple, downloadsDir, false, d.Source, opts.sink()); err != nil {
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
	current, err := toolchain.ReadInstallation(roots.TcDir)
	currentState, stateErr := installationComponentState(roots.TcDir)
	if err != nil || current != record || stateErr != nil || currentState != state {
		return fmt.Errorf("toolchain %s changed during component installation; retry", resolvedName)
	}
	return component.ApplyChanges(roots, selected, func() error {
		for _, c := range selected {
			if err := component.InstallPrepared(roots, prepared, c, force); err != nil {
				return err
			}
			opts.emit(progress.Event{Kind: progress.ComponentInstalled, Toolchain: resolvedName, Component: string(c)})
		}
		return nil
	})
}
