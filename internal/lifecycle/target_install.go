package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/component"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// InstallTargetsForToolchain installs missing cross SDKs for the release already
// inside the selected host. Proxy auto-install must not upgrade that host merely
// because a project requests a target that has not been installed yet.
func InstallTargetsForToolchain(ctx context.Context, input string, targets []string, opts Options) error {
	if len(targets) == 0 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	targets, err := sdktarget.NormalizeList(targets)
	if err != nil {
		return err
	}
	d, err := OpenDistribution(opts)
	if err != nil {
		return err
	}
	dir, _, _, err := toolchain.FindActiveDir(input)
	if err != nil {
		return err
	}
	record, opts, err := captureDependency(ctx, filepath.Base(dir), opts)
	if err != nil {
		return err
	}
	host, err := toolchain.ParseToolchainName(record.Release)
	if err != nil {
		return err
	}
	if host.IsCustom() || host.Version == "" {
		return fmt.Errorf("toolchain %s has no release for cross SDK installation", input)
	}
	identity, err := toolchain.ParseToolchainName(filepath.Base(dir))
	if err != nil {
		return err
	}
	for _, target := range targets {
		tuple, err := d.TargetTuple(target)
		if err != nil {
			return err
		}
		rt, err := d.Resolve(ctx, host, tuple)
		if err != nil {
			return err
		}
		if err := installSelected(ctx, d, rt, identity.Version == "", false, false, opts); err != nil {
			return err
		}
	}
	return nil
}

// Capture the dependency while holding home, then recheck it during publication.
// An empty record also guards against a host appearing after target resolution.
func captureDependency(ctx context.Context, identity string, opts Options) (toolchain.Installation, Options, error) {
	lock, err := toolchain.LockHome(ctx)
	if err != nil {
		return toolchain.Installation{}, opts, err
	}
	defer lock.Close() //nolint:errcheck
	if err := lock.Recover(); err != nil {
		return toolchain.Installation{}, opts, err
	}
	dir, err := component.RootsFor(identity)
	if err != nil {
		return toolchain.Installation{}, opts, err
	}
	record, err := toolchain.ReadInstallation(dir.TcDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return toolchain.Installation{}, opts, err
	}
	opts.dependencies = maps.Clone(opts.dependencies)
	if opts.dependencies == nil {
		opts.dependencies = make(map[string]toolchain.Installation)
	}
	opts.dependencies[identity] = record
	return record, opts, nil
}

func selectTargetHost(ctx context.Context, selector toolchain.ToolchainName, opts Options) (toolchain.ToolchainName, Options, error) {
	record, opts, err := captureDependency(ctx, selector.Channel.String(), opts)
	if err != nil {
		return selector, opts, err
	}
	if record.Release != "" {
		host, err := toolchain.ParseToolchainName(record.Release)
		if err != nil {
			return selector, opts, err
		}
		selector.Version = host.Version
	}
	return selector, opts, nil
}

// validateDependencies requires the home lock.
func validateDependencies(opts Options) error {
	for identity, expected := range opts.dependencies {
		roots, err := component.RootsFor(identity)
		if err != nil {
			return err
		}
		current, err := toolchain.ReadInstallation(roots.TcDir)
		if expected.Release == "" && errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil || current != expected {
			return fmt.Errorf("toolchain %s changed during target installation; retry", identity)
		}
	}
	return nil
}
