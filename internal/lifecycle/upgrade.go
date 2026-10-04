package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// upgradeToolchain prepares every component before atomically replacing the
// channel's independent SDK, docs and stdx roots. Fixed versions are untouched.
func upgradeToolchain(ctx context.Context, identity string, rt ResolvedToolchain, d *Distribution, opts Options) (bool, error) {
	name, err := toolchain.ParseToolchainName(identity)
	if err != nil {
		return false, err
	}
	if name.IsCustom() || name.Version != "" {
		return false, fmt.Errorf("toolchain %s is not a tracking installation", identity)
	}
	if err := toolchain.RecoverHomeContext(ctx); err != nil {
		return false, err
	}
	lock, err := toolchain.LockHome(ctx)
	if err != nil {
		return false, err
	}
	roots, err := component.RootsFor(identity)
	if err != nil {
		_ = lock.Close()
		return false, err
	}
	current, err := toolchain.ReadInstallation(roots.TcDir)
	if err != nil {
		_ = lock.Close()
		return false, err
	}
	if expected, ok := d.installed[identity]; ok && expected != current {
		_ = lock.Close()
		return false, fmt.Errorf("toolchain %s changed during update; retry", identity)
	}
	intents, err := component.InstalledIntents(roots)
	if err == nil {
		opts.expectedComponents, err = installationComponentState(roots.TcDir)
	}
	if closeErr := lock.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return false, err
	}
	opts.selection, opts.expectedSet, opts.expected = identity, true, &current
	opts.prepare = func(ctx context.Context, staged component.Roots) error {
		release, err := toolchain.ParseToolchainName(rt.Name)
		if err != nil {
			return err
		}
		downloads, err := config.DownloadsDir()
		if err != nil {
			return err
		}
		for _, intent := range intents {
			if intent.Source != "" {
				_, err = component.Link(staged, intent.Name, intent.Source, false)
			} else {
				err = component.InstallFromSource(ctx, staged, release, intent.Name, rt.Tuple, downloads, false, d.Source, opts.sink())
			}
			if err != nil {
				return err
			}
		}
		return nil
	}
	opts.emit(progress.Event{Kind: progress.UpdateFound, Toolchain: identity, Replacement: rt.Name})
	if err := installResolved(ctx, d, rt, true, false, opts); err != nil {
		return false, err
	}
	return true, nil
}
func componentState(dir string) (string, error) {
	var state string
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return "", err
		}
		state += entry.Name() + "\x00" + string(data) + "\x00"
	}
	return state, nil
}

func installationComponentState(dir string) (string, error) {
	state, err := componentState(filepath.Join(dir, component.MetaDir))
	if err != nil {
		return "", err
	}
	roots, err := component.RootsFor(filepath.Base(dir))
	if err != nil {
		return "", err
	}
	intents, err := component.InstalledIntents(roots)
	if err != nil {
		return "", err
	}
	for _, intent := range intents {
		state += string(intent.Name) + "\x00" + intent.Source + "\x00"
	}
	return state, nil
}
