package lifecycle

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
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
	resolvedTC, err := toolchain.ParseToolchainName(resolvedName)
	if err != nil {
		return err
	}
	if resolvedTC.IsCustom() {
		return &cjverr.ComponentRequiresHostError{Component: strings.Join(components, ", ")}
	}
	parsed, err := component.NormalizeList(components)
	if err != nil {
		return err
	}
	tuple := resolvedTC.Target
	if tuple == "" {
		tuple = d.HostTuple
	}
	downloadsDir, err := config.DownloadsDir()
	if err != nil {
		return err
	}
	roots, err := component.RootsFor(resolvedName)
	if err != nil {
		return err
	}
	return component.ApplyChanges(roots, parsed, func() error {
		for _, c := range parsed {
			d.note()
			if err := component.InstallFromSource(ctx, roots, resolvedTC, c, tuple, downloadsDir, force, d.Source, opts.sink()); err != nil {
				var alreadyErr *cjverr.ComponentAlreadyInstalledError
				if errors.As(err, &alreadyErr) {
					opts.emit(progress.Event{Kind: progress.ComponentAlreadyInstalled, Toolchain: resolvedName, Component: string(c)})
					continue
				}
				return err
			}
			opts.emit(progress.Event{Kind: progress.ComponentInstalled, Toolchain: resolvedName, Component: string(c)})
		}
		return nil
	})
}
