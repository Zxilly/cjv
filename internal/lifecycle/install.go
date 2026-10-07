package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/progress"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// Options carries what an operation needs from its caller. The install
// implementation is shared by CLI commands and proxy auto-install;
// presentation stays outside the core module.
type Options struct {
	// Progress receives every progress event of the operation, including the
	// download transfers; nil keeps the operation silent. The caller picks
	// the adapter: progress.Text for humans, progress.Discard for JSON mode.
	Progress progress.Sink
	// ConfigurePath adds CJV_HOME/bin to the user's PATH when the install
	// publishes the first default toolchain. `cjv install` sets it; `cjv init`
	// has already handled PATH itself, and proxy auto-install leaves PATH
	// alone because cjv is evidently reachable.
	ConfigurePath bool
	// AllowMissing explicitly permits dropping unavailable components/targets.
	AllowMissing   bool
	AllowDowngrade bool
	// Internal dependencies are revalidated before publishing target SDKs.
	dependencies    map[string]toolchain.Installation
	preserveDefault bool
}

// sink returns the progress adapter to emit to, never nil.
func (o Options) sink() progress.Sink {
	return progress.Or(o.Progress)
}

func (o Options) emit(e progress.Event) {
	o.sink().Report(e)
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
	// Force permits skipping unavailable components and target SDKs. It does
	// not reinstall an unchanged SDK; updates use Options.AllowMissing too.
	Force    bool
	NoUpdate bool
}

// Install resolves the request against the configured distribution source
// and places the host toolchain, its target variants and their components.
// The first host toolchain installed becomes the default; target variants
// never do. An unchanged installed toolchain is reported and kept.
func Install(ctx context.Context, req InstallRequest, opts Options) error {
	if req.NoUpdate && req.Force {
		return fmt.Errorf("cannot combine force with no-update")
	}
	opts.AllowMissing = opts.AllowMissing || req.Force
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

	d, err := openInstallationDistribution(ctx, opts)
	if err != nil {
		return err
	}
	req.Targets = targets
	if req.NoUpdate {
		if dir, err := toolchain.FindInstalled(name); err == nil {
			record, err := toolchain.ReadInstallation(dir)
			if err != nil {
				return err
			}
			tuple := record.Tuple
			if tuple == "" {
				tuple = d.HostTuple
				if name.Host != "" {
					tuple = name.Host
				}
				if name.Target != "" {
					tuple = name.Target
				}
			}
			_, err = installGroup(ctx, d, name, ResolvedToolchain{Name: record.Release, Tuple: tuple, SHA256: record.SHA256}, req, opts)
			return err
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	// A fixed version already on disk needs no remote manifest to remain
	// installed. The directory itself is its installation identity.
	if name.Version != "" && !toolchain.IsVersionSelector(name.Version) && len(targets) == 0 && len(req.Components) == 0 {
		if dir, err := toolchain.FindInstalled(name); err == nil {
			identity, err := toolchain.ParseToolchainName(filepath.Base(dir))
			if err != nil {
				return err
			}
			record, err := toolchain.ReadInstallation(dir)
			if err != nil {
				return err
			}
			tuple := record.Tuple
			if tuple == "" {
				tuple = d.HostTuple
				if identity.Host != "" {
					tuple = identity.Host
				}
				if identity.Target != "" {
					tuple = identity.Target
				}
			}
			_, err = installGroup(ctx, d, identity, ResolvedToolchain{Name: record.Release, Tuple: tuple, SHA256: record.SHA256}, req, opts)
			return err
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	selector := name
	if name.Target != "" && name.Version == "" {
		selector, opts, err = selectTargetHost(ctx, name, opts)
		if err != nil {
			return err
		}
	}
	resolved, err := d.Resolve(ctx, selector, name.Target)
	if err != nil {
		return err
	}
	_, err = installGroup(ctx, d, name, resolved, req, opts)
	return err
}
