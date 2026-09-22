package lifecycle

import (
	"context"
	"errors"
	"fmt"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// Options carries the small adapter surface the lifecycle module needs from
// callers. The install implementation is shared by CLI commands and proxy
// auto-install; presentation stays outside the core module.
type Options struct {
	// Report receives progress messages; nil keeps library operations silent.
	Report               func(string, i18n.MsgData)
	EnsurePathConfigured func()
	// ComponentInstall, when set, replaces the real component installer and
	// gives orchestration tests a source-independent adapter.
	ComponentInstall     func(context.Context, component.Roots, toolchain.ToolchainName, component.Name, string, string, bool) error
	EnsureManagedBinary  func() (string, error)
	CreateProxyLinks     func() error
	ValidateInstallation func(dir, tuple string) error
}

func (o Options) report(message string, data i18n.MsgData) {
	if o.Report != nil {
		o.Report(message, data)
	}
}

func (o Options) ensurePathConfigured() {
	if o.EnsurePathConfigured != nil {
		o.EnsurePathConfigured()
		return
	}
	EnsurePathConfigured()
}

func (o Options) installComponent(ctx context.Context, roots component.Roots, tc toolchain.ToolchainName, name component.Name, tuple, downloadsDir string, force bool, fetcher *ManifestFetcher) error {
	if o.ComponentInstall != nil {
		return o.ComponentInstall(ctx, roots, tc, name, tuple, downloadsDir, force)
	}
	if o.Report != nil {
		fetcher.Note()
	}
	return component.InstallFromSource(ctx, roots, tc, name, tuple, downloadsDir, force, fetcher.source, func(stage string) {
		o.report(stage, i18n.MsgData{"Toolchain": tc.String(), "Component": string(name)})
	})
}

func (o Options) createProxyLinks() error {
	if o.CreateProxyLinks == nil {
		return nil
	}
	return o.CreateProxyLinks()
}

func (o Options) ensureManagedBinary() error {
	if o.EnsureManagedBinary == nil {
		return nil
	}
	_, err := o.EnsureManagedBinary()
	return err
}

func (o Options) validateInstallation(dir, tuple string) error {
	if o.ValidateInstallation != nil {
		return o.ValidateInstallation(dir, tuple)
	}
	return validateInstallation(dir, tuple)
}

// InstallToolchainWithOptions installs a toolchain with optional force re-install.
func InstallToolchainWithOptions(ctx context.Context, input string, force bool, opts Options) error {
	return InstallToolchainWithExtras(ctx, input, nil, nil, force, opts)
}

// InstallToolchainWithTargets installs the host toolchain plus optional cross SDK target variants.
func InstallToolchainWithTargets(ctx context.Context, input string, targets []string, force bool, opts Options) error {
	return InstallToolchainWithExtras(ctx, input, targets, nil, force, opts)
}

// InstallToolchainWithExtras installs the host toolchain plus optional cross
// SDK target variants and optional components.
func InstallToolchainWithExtras(ctx context.Context, input string, targets, components []string, force bool, opts Options) error {
	if ctx == nil {
		ctx = context.Background()
	}
	name, err := toolchain.ParseToolchainName(input)
	if err != nil {
		return err
	}
	if name.IsCustom() {
		return errors.New(i18n.T("InstallCustomToolchain", i18n.MsgData{"Name": input}))
	}

	sf, settings, err := LoadSettings()
	if err != nil {
		return err
	}

	normalizedTargets, err := sdktarget.NormalizeList(targets)
	if err != nil {
		return err
	}
	if name.Target != "" && len(normalizedTargets) > 0 {
		return fmt.Errorf("cannot combine target variant toolchain name %q with --target; pass the host toolchain name and --target instead", input)
	}

	fetcher, err := NewManifestFetcherForSettings(settings, opts)
	if err != nil {
		return err
	}
	resolved, err := ResolveAndLocate(ctx, name, settings, fetcher)
	if err != nil {
		return err
	}

	if name.Target != "" {
		if err := InstallResolvedNoDefault(ctx, resolved, settings, sf, force, opts); err != nil {
			return err
		}
	} else if err := InstallResolved(ctx, resolved, settings, sf, force, opts); err != nil {
		return err
	}

	targetBase := name
	if len(normalizedTargets) > 0 {
		hostResolved, err := toolchain.ParseToolchainName(resolved.Name)
		if err != nil {
			return err
		}
		targetBase = toolchain.ToolchainName{Channel: hostResolved.Channel, Version: hostResolved.Version}
	}

	var targetNames []string
	for _, target := range normalizedTargets {
		resolvedTarget, err := resolveTargetToolchain(ctx, targetBase, settings, fetcher, target)
		if err != nil {
			return err
		}
		if err := InstallResolvedNoDefault(ctx, resolvedTarget, settings, sf, force, opts); err != nil {
			return err
		}
		targetNames = append(targetNames, resolvedTarget.Name)
	}

	if len(components) > 0 {
		if len(normalizedTargets) > 0 {
			for _, targetName := range targetNames {
				if err := InstallComponentsList(ctx, targetName, components, force, false, fetcher, opts); err != nil {
					return err
				}
			}
		} else if err := InstallComponentsList(ctx, resolved.Name, components, force, false, fetcher, opts); err != nil {
			return err
		}
	}
	return nil
}

func resolveTargetToolchain(ctx context.Context, base toolchain.ToolchainName, settings *config.Settings, fetcher *ManifestFetcher, target string) (ResolvedToolchain, error) {
	return ResolveAndLocateWithTarget(ctx, base, settings, fetcher, target)
}

// LoadSettings loads the cached user settings file used by lifecycle operations.
func LoadSettings() (*config.SettingsFile, *config.Settings, error) {
	sf, err := config.DefaultSettingsFile()
	if err != nil {
		return nil, nil, err
	}
	settings, err := sf.Load()
	if err != nil {
		return nil, nil, err
	}
	return sf, settings, nil
}
