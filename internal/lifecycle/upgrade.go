package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// upgradeToolchain installs a replacement and the old toolchain's component
// choices before moving references and retiring the old content. An existing
// replacement keeps its own component choices; only missing ones are added.
// UpdateInstalled and UpdateAll resolve the replacement and run this step.
func upgradeToolchain(ctx context.Context, currentName string, resolved ResolvedToolchain, d *Distribution, opts Options) (updated bool, retErr error) {
	if _, err := toolchain.ParseToolchainName(currentName); err != nil {
		return false, err
	}
	parsed, err := toolchain.ParseToolchainName(resolved.Name)
	if err != nil {
		return false, err
	}
	oldRoots, err := component.RootsFor(currentName)
	if err != nil {
		return false, err
	}
	if err := toolchain.RecoverHome(); err != nil {
		return false, err
	}
	if _, err := os.Lstat(oldRoots.TcDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, &cjverr.ToolchainNotInstalledError{Name: currentName}
		}
		return false, err
	}
	if currentName == resolved.Name {
		opts.report("AlreadyUpToDate", i18n.MsgData{"Version": currentName})
		return false, nil
	}
	intents, err := component.InstalledIntents(oldRoots)
	if err != nil {
		return false, err
	}
	// Reload: a previous upgrade in the same operation may have moved the
	// default and overrides.
	sf := d.File
	settings, err := sf.Load()
	if err != nil {
		return false, err
	}
	newRoots, err := component.RootsFor(resolved.Name)
	if err != nil {
		return false, err
	}
	_, existsErr := os.Lstat(newRoots.TcDir)
	if existsErr != nil && !errors.Is(existsErr, os.ErrNotExist) {
		return false, existsErr
	}
	newlyInstalled := errors.Is(existsErr, os.ErrNotExist)
	opts.report("UpdateFound", i18n.MsgData{"Current": currentName, "Latest": resolved.Name})
	// Keep the old default until the replacement and all desired components
	// are ready. Remove a newly created replacement on failure so resolving a
	// channel cannot select the incomplete higher version on the next attempt.
	if err := installResolved(ctx, d, resolved, false, false, opts); err != nil {
		return false, err
	}
	keepReplacement := false
	defer func() {
		if retErr != nil && newlyInstalled && !keepReplacement {
			// A failed publication/restore may have changed the persisted
			// settings even when the cached snapshot still names the old SDK.
			// Preserve a replacement if live references require it, or their
			// state cannot be read safely.
			required, err := replacementRequired(sf, resolved.Name)
			if err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("retain replacement %s: cannot verify settings: %w", resolved.Name, err))
				return
			}
			if required {
				return
			}
			if err := retireToolchain(newRoots, sf, settings, config.SettingsUpdate{}); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("remove incomplete replacement %s: %w", resolved.Name, err))
			}
		}
	}()
	var missing []component.Intent
	var names []component.Name
	for _, intent := range intents {
		if !component.IsInstalled(newRoots.TcDir, intent.Name) {
			missing = append(missing, intent)
			names = append(names, intent.Name)
		}
	}
	var retirementErr error
	err = component.ApplyChanges(newRoots, names, func() error {
		for _, intent := range missing {
			if intent.Source != "" {
				if _, err := component.Link(newRoots, intent.Name, intent.Source, false); err != nil {
					return err
				}
			} else if err := installComponents(ctx, d, resolved.Name, []string{string(intent.Name)}, false, opts); err != nil {
				return err
			}
		}
		update := config.SettingsUpdate{}
		if parsed.Target == "" {
			if settings.DefaultToolchain == currentName || !defaultToolchainExists(settings.DefaultToolchain) {
				update.DefaultToolchain = &resolved.Name
			}
			for dir, name := range settings.Overrides {
				if name == currentName {
					if update.Overrides == nil {
						update.Overrides = maps.Clone(settings.Overrides)
					}
					update.Overrides[dir] = resolved.Name
				}
			}
		}
		retirementErr = retireToolchain(oldRoots, sf, settings, update)
		var partial *retirementError
		if errors.As(retirementErr, &partial) && partial.KeepReplacement {
			// Commit the already working component set when recovery must
			// preserve the new SDK. Returning the error from this callback
			// would otherwise silently remove components selected by settings.
			keepReplacement = true
			return nil
		}
		return retirementErr
	})
	if err == nil {
		err = retirementErr
	}
	return err == nil, err
}

func replacementRequired(sf *config.SettingsFile, name string) (bool, error) {
	sf.Invalidate()
	settings, err := sf.Load()
	if err != nil {
		return true, err
	}
	if settings.DefaultToolchain == name {
		return true, nil
	}
	for _, selected := range settings.Overrides {
		if selected == name {
			return true, nil
		}
	}
	return false, nil
}
