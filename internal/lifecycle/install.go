package lifecycle

import (
	"context"
	"errors"
	"fmt"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/i18n"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// Options carries the small adapter surface the lifecycle module needs from
// callers. The install implementation is shared by CLI commands and proxy
// auto-install; presentation stays outside the core module.
type Options struct {
	// Report receives progress messages; nil keeps library operations silent.
	Report func(string, i18n.MsgData)
	// ConfigurePath adds CJV_HOME/bin to the user's PATH when the install
	// publishes the first default toolchain. `cjv install` sets it; `cjv init`
	// has already handled PATH itself, and proxy auto-install leaves PATH
	// alone because cjv is evidently reachable.
	ConfigurePath bool
	// ComponentInstall, when set, replaces the real component installer and
	// gives orchestration tests a source-independent adapter.
	ComponentInstall func(context.Context, component.Roots, toolchain.ToolchainName, component.Name, string, string, bool) error
}

func (o Options) report(message string, data i18n.MsgData) {
	if o.Report != nil {
		o.Report(message, data)
	}
}

func (o Options) installComponent(ctx context.Context, d *Distribution, roots component.Roots, tc toolchain.ToolchainName, name component.Name, tuple, downloadsDir string, force bool) error {
	if o.ComponentInstall != nil {
		return o.ComponentInstall(ctx, roots, tc, name, tuple, downloadsDir, force)
	}
	if o.Report != nil {
		d.note()
	}
	return component.InstallFromSource(ctx, roots, tc, name, tuple, downloadsDir, force, d.Source, func(stage string) {
		o.report(stage, i18n.MsgData{"Toolchain": tc.String(), "Component": string(name)})
	})
}

// InstallRequest names what an install brings into CJV_HOME.
type InstallRequest struct {
	// Toolchain is a channel ("lts"), a version ("1.0.5"), a channel-version
	// ("lts-1.0.5") or a target variant name ("lts-1.0.5-linux-x64-ohos").
	Toolchain string
	// Targets are cross SDK environments ("ohos") installed as variants of
	// the resolved host toolchain. They cannot be combined with a target
	// variant Toolchain.
	Targets []string
	// Components are installed into every toolchain this request installs:
	// the target variants when Targets is set, otherwise the host toolchain.
	Components []string
	// Force replaces an already installed toolchain instead of keeping it.
	Force bool
}

// Install resolves the request against the configured distribution source
// and places the host toolchain, its target variants and their components.
// The first host toolchain installed becomes the default; target variants
// never do. An already installed toolchain is reported and kept unless Force
// is set.
func Install(ctx context.Context, req InstallRequest, opts Options) error {
	if ctx == nil {
		ctx = context.Background()
	}
	name, err := toolchain.ParseToolchainName(req.Toolchain)
	if err != nil {
		return err
	}
	if name.IsCustom() {
		return errors.New(i18n.T("InstallCustomToolchain", i18n.MsgData{"Name": req.Toolchain}))
	}
	targets, err := sdktarget.NormalizeList(req.Targets)
	if err != nil {
		return err
	}
	if name.Target != "" && len(targets) > 0 {
		return fmt.Errorf("cannot combine target variant toolchain name %q with --target; pass the host toolchain name and --target instead", req.Toolchain)
	}

	d, err := OpenDistribution(opts)
	if err != nil {
		return err
	}
	resolved, err := d.Resolve(ctx, name, name.Target)
	if err != nil {
		return err
	}
	if err := installResolved(ctx, d, resolved, req.Force, name.Target == "", opts); err != nil {
		return err
	}

	// Target variants are pinned to the host's resolved version so
	// `envsetup --target` never sees a version skew.
	hostResolved, err := toolchain.ParseToolchainName(resolved.Name)
	if err != nil {
		return err
	}
	targetBase := toolchain.ToolchainName{Channel: hostResolved.Channel, Version: hostResolved.Version}
	installed := []string{resolved.Name}
	if len(targets) > 0 {
		installed = nil
	}
	for _, target := range targets {
		tuple, err := d.TargetTuple(target)
		if err != nil {
			return err
		}
		resolvedTarget, err := d.Resolve(ctx, targetBase, tuple)
		if err != nil {
			return err
		}
		if err := installResolved(ctx, d, resolvedTarget, req.Force, false, opts); err != nil {
			return err
		}
		installed = append(installed, resolvedTarget.Name)
	}

	if len(req.Components) == 0 {
		return nil
	}
	for _, tcName := range installed {
		if err := installComponents(ctx, d, tcName, req.Components, req.Force, opts); err != nil {
			return err
		}
	}
	return nil
}
