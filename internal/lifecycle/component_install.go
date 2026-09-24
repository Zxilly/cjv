package lifecycle

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// InstallComponentsForToolchain backs the proxy auto_install path: it resolves
// tcInput to an already-installed toolchain and installs missing components
// quietly, whatever reporter the caller supplied.
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
	opts.Report = nil
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
			if err := opts.installComponent(ctx, d, roots, resolvedTC, c, tuple, downloadsDir, force); err != nil {
				var alreadyErr *cjverr.ComponentAlreadyInstalledError
				if errors.As(err, &alreadyErr) {
					opts.report("ComponentAlreadyInstalled", i18n.MsgData{"Toolchain": resolvedName, "Component": string(c)})
					continue
				}
				return err
			}
			opts.report("ComponentInstalled", i18n.MsgData{"Toolchain": resolvedName, "Component": string(c)})
		}
		return nil
	})
}
