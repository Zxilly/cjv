package lifecycle

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// InstallComponentsForToolchain backs the proxy auto_install path: it resolves
// tcInput to an already-installed toolchain and installs missing components quietly.
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
	return InstallComponentsList(ctx, filepath.Base(installedDir), components, false, true, nil, opts)
}

// InstallComponentsList expects resolvedName as "<channel>-<version>". The
// configured manifest source resolves component artifacts for every channel.
func InstallComponentsList(ctx context.Context, resolvedName string, components []string, force, quiet bool, fetcher *ManifestFetcher, opts Options) error {
	if quiet {
		opts.Report = nil
	}
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
	_, settings, err := LoadSettings()
	if err != nil {
		return err
	}
	tuple := resolvedTC.Target
	if tuple == "" {
		tuple, err = dist.CurrentHostTuple(settings.DefaultHost)
		if err != nil {
			return err
		}
	}
	if fetcher == nil {
		fetcher, err = NewManifestFetcherForSettings(settings, opts)
		if err != nil {
			return err
		}
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
			if err := opts.installComponent(ctx, roots, resolvedTC, c, tuple, downloadsDir, force, fetcher); err != nil {
				var alreadyErr *cjverr.ComponentAlreadyInstalledError
				if errors.As(err, &alreadyErr) {
					opts.report("ComponentAlreadyInstalled", i18n.MsgData{"Toolchain": resolvedName, "Component": string(c)})
					continue
				}
				return err
			}
			if !quiet {
				opts.report("ComponentInstalled", i18n.MsgData{"Toolchain": resolvedName, "Component": string(c)})
			}
		}
		return nil
	})
}
