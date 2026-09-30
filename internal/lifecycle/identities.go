package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// Target identities are internal keys; public selectors remain channel names
// plus --target, or explicit versioned variant names.
func trackingIdentity(name toolchain.ToolchainName) string {
	key := name.Channel.String()
	if name.Target != "" {
		key += "/" + name.Target
	}
	return key
}

// Older installations did not record intent. Preserve every unowned SDK as
// a fixed version rather than guessing that a project no longer needs it.
func recordLegacyInstallations(ctx context.Context, d *Distribution, persist bool) error {
	lock, err := toolchain.LockHome(ctx)
	if err != nil {
		return err
	}
	defer lock.Close() //nolint:errcheck
	if err := lock.Recover(); err != nil {
		return err
	}
	d.File.Invalidate()
	settings, err := d.File.Load()
	if err != nil {
		return err
	}
	installed, err := toolchain.ListInstalled()
	if err != nil {
		return err
	}
	choices := maps.Clone(settings.Installations)
	d.legacyPins = make(map[string]string)
	if choices == nil {
		choices = make(map[string]string)
	}
	for _, name := range installed {
		parsed, err := toolchain.ParseToolchainName(name)
		if err != nil || parsed.IsCustom() {
			continue
		}
		owned := false
		for _, concrete := range choices {
			if concrete == name {
				owned = true
				break
			}
		}
		if !owned {
			choices[name] = name
			d.legacyPins[name] = name
		}
	}
	settings.Installations = choices
	if persist {
		if _, err := d.File.Update(config.SettingsUpdate{Installations: choices}); err != nil {
			return err
		}
	}
	d.Settings = settings
	return nil
}

func installSelected(ctx context.Context, d *Distribution, rt ResolvedToolchain, tracking, force, setDefault bool, opts Options) error {
	parsed, err := toolchain.ParseToolchainName(rt.Name)
	if err != nil {
		return err
	}
	identity := rt.Name
	if tracking {
		identity = trackingIdentity(parsed)
		settings := d.Settings
		if old := settings.Installations[identity]; old != "" && old != rt.Name {
			opts.tracking = identity
			if _, err := upgradeToolchain(ctx, old, rt, d, opts); err != nil {
				return err
			}
			// An install can additionally request a forced refresh.
			if !force {
				d.Settings, err = d.File.Load()
				return err
			}
		}
	}
	opts.selection = identity
	return installResolved(ctx, d, rt, force, setDefault, opts)
}

func retainInstallation(settings *config.Settings, concrete, advancing string) bool {
	for identity, name := range settings.Installations {
		if name == concrete && identity != advancing {
			return true
		}
	}
	// Exact selectors express a fixed version even if it was first downloaded
	// for a channel. Project files remain outside cjv's ownership.
	if selectsConcrete(settings.DefaultToolchain, concrete) {
		return true
	}
	for _, name := range settings.Overrides {
		if selectsConcrete(name, concrete) {
			return true
		}
	}
	return false
}

func selectsConcrete(selector, concrete string) bool {
	selected, err := toolchain.ParseToolchainName(selector)
	if err != nil || selected.IsCustom() || selected.Version == "" {
		return false
	}
	installed, err := toolchain.ParseToolchainName(concrete)
	return err == nil && selected.Version == installed.Version && selected.Target == installed.Target &&
		(selected.Channel == toolchain.UnknownChannel || selected.Channel == installed.Channel)
}

func isTrackingIdentity(identity, concrete string) bool {
	channel, _, _ := strings.Cut(identity, "/")
	_, ok := toolchain.ParseChannel(channel)
	return ok && identity != concrete
}

func updateTrackedTargets(ctx context.Context, d *Distribution, channel toolchain.Channel, opts Options) error {
	settings, err := d.File.Load()
	if err != nil {
		return err
	}
	var identities []string
	for identity, concrete := range settings.Installations {
		if isTrackingIdentity(identity, concrete) && strings.HasPrefix(identity, channel.String()+"/") {
			identities = append(identities, identity)
		}
	}
	slices.Sort(identities)
	var errs []error
	for _, identity := range identities {
		parsed, err := toolchain.ParseToolchainName(settings.Installations[identity])
		if err == nil {
			_, err = upgradeChannelToolchain(ctx, d, parsed.Channel, parsed.String(), parsed.Target, opts)
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Check ownership again while holding the same lock used for explicit pins.
// A pin created during the download must protect the old SDK too.
func advanceTracking(ctx context.Context, roots component.Roots, d *Distribution, identity, current, replacement string) error {
	lock, err := toolchain.LockHome(ctx)
	if err != nil {
		return err
	}
	defer lock.Close() //nolint:errcheck
	if err := lock.Recover(); err != nil {
		return err
	}
	d.File.Invalidate()
	settings, err := d.File.Load()
	if err != nil {
		return err
	}
	if settings.Installations[identity] != current {
		return fmt.Errorf("toolchain %s changed during update; retry the update", identity)
	}
	choices := maps.Clone(settings.Installations)
	choices[identity] = replacement
	update := config.SettingsUpdate{Installations: choices}
	if retainInstallation(settings, current, identity) {
		choices[current] = current
		_, err := d.File.Update(update)
		return err
	}
	return retireLocked(roots, d.File, settings, update)
}
