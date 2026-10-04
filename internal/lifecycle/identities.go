package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/toolchain"
)

func selectedIdentity(rt ResolvedToolchain, tracking bool) string {
	if !tracking {
		return rt.Name
	}
	name, _ := toolchain.ParseToolchainName(rt.Name)
	name.Version = ""
	return name.String()
}
func installSelected(ctx context.Context, d *Distribution, rt ResolvedToolchain, tracking, force, setDefault bool, opts Options) error {
	opts.selection = selectedIdentity(rt, tracking)
	if !tracking {
		return installResolved(ctx, d, rt, force, setDefault, opts)
	}
	roots, err := component.RootsFor(opts.selection)
	if err != nil {
		return err
	}
	_, err = os.Lstat(roots.TcDir)
	if errors.Is(err, os.ErrNotExist) {
		if _, existed := d.installed[opts.selection]; existed {
			return fmt.Errorf("toolchain %s was removed during installation; retry", opts.selection)
		}
		opts.expectedSet = true
		return installResolved(ctx, d, rt, false, setDefault, opts)
	}
	if err != nil {
		return err
	}
	installed, err := toolchain.ReadInstallation(roots.TcDir)
	if err != nil {
		return err
	}
	if installed.Release == rt.Name && installed.Tuple == rt.Tuple && installed.SHA256 == rt.SHA256 && !force {
		if setDefault || len(opts.dependencies) > 0 {
			lock, err := toolchain.LockHome(ctx)
			if err != nil {
				return err
			}
			defer lock.Close() //nolint:errcheck
			if err := lock.Recover(); err != nil {
				return err
			}
			if _, err := os.Stat(roots.TcDir); err != nil {
				return err
			}
			if err := validateDependencies(opts); err != nil {
				return err
			}
			if setDefault {
				if err := publishFirstDefault(d, opts.selection, opts); err != nil {
					return err
				}
			}
		}
		opts.emit(progress.Event{Kind: progress.AlreadyUpToDate, Toolchain: opts.selection})
		return nil
	}
	_, err = upgradeToolchain(ctx, opts.selection, rt, d, opts)
	if err != nil || !setDefault {
		return err
	}
	lock, err := toolchain.LockHome(ctx)
	if err != nil {
		return err
	}
	defer lock.Close() //nolint:errcheck
	if err := lock.Recover(); err != nil {
		return err
	}
	return publishFirstDefault(d, opts.selection, opts)
}
func updateTrackedTargets(ctx context.Context, d *Distribution, channel toolchain.Channel, opts Options) error {
	installed, err := toolchain.ListInstalled()
	if err != nil {
		return err
	}
	var errs []error
	for _, identity := range installed {
		name, err := toolchain.ParseToolchainName(identity)
		if err != nil || name.IsCustom() || name.Channel != channel || name.Version != "" || name.Target == "" {
			continue
		}
		_, err = upgradeChannelToolchain(ctx, d, channel, identity, name.Target, opts)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
