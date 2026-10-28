package lifecycle

import (
	"context"
	"errors"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// ComponentRemoval records successful removals even when other components fail.
type ComponentRemoval struct {
	Removed []component.Name
}

// RemoveComponents removes each requested component independently. A failed or
// missing component does not prevent later removals. A nil result means setup
// failed before any removal was attempted; otherwise the result accompanies any
// errors so callers can report partial success.
func RemoveComponents(ctx context.Context, identity string, names []component.Name, opts Options) (*ComponentRemoval, error) {
	lock, roots, err := lockComponentRoots(ctx, identity)
	if err != nil {
		return nil, err
	}
	defer lock.Close() //nolint:errcheck
	result := &ComponentRemoval{}
	var errs []error
	for _, name := range names {
		if !component.IsInstalled(roots.TcDir, name) {
			errs = append(errs, &cjverr.ComponentNotInstalledError{Toolchain: identity, Component: string(name)})
			continue
		}
		opts.emit(progress.Event{Kind: progress.RemovingComponent, Component: string(name)})
		if err := component.Remove(roots, name); err != nil {
			errs = append(errs, err)
			continue
		}
		result.Removed = append(result.Removed, name)
	}
	return result, errors.Join(errs...)
}

// LinkComponent attaches a user-owned source to an installed SDK while holding
// the home mutation lock through component validation, placement and rollback.
// Custom SDKs installed into owned directories support links too.
func LinkComponent(ctx context.Context, identity string, name component.Name, source string, force bool, opts Options) (string, error) {
	lock, roots, err := lockComponentRoots(ctx, identity)
	if err != nil {
		return "", err
	}
	defer lock.Close() //nolint:errcheck
	opts.emit(progress.Event{Kind: progress.InstallingComponent, Component: string(name)})
	return component.Link(roots, name, source, force)
}

func lockComponentRoots(ctx context.Context, identity string) (*toolchain.HomeLock, component.Roots, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := toolchain.ParseToolchainName(identity); err != nil {
		return nil, component.Roots{}, err
	}
	lock, err := toolchain.LockHome(ctx)
	if err != nil {
		return nil, component.Roots{}, err
	}
	if err := lock.Recover(); err != nil {
		_ = lock.Close()
		return nil, component.Roots{}, err
	}
	roots, err := component.RootsFor(identity)
	if err != nil {
		_ = lock.Close()
		return nil, component.Roots{}, err
	}
	return lock, roots, nil
}
