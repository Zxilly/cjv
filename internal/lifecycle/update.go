package lifecycle

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// UpdateStatus tells how one installed toolchain fared in an update.
type UpdateStatus int

const (
	// UpdateSkipped marks custom or linked toolchains, which no channel can
	// replace.
	UpdateSkipped UpdateStatus = iota
	// UpdateUpToDate means the installed toolchain already is the channel head.
	UpdateUpToDate
	// UpdateApplied means the replacement is installed and the old toolchain
	// has been retired.
	UpdateApplied
	// UpdatePinned marks an explicit version: it is installed when missing and
	// never replaced by a newer one.
	UpdatePinned
	// UpdateFailed carries the error in UpdateOutcome.Err.
	UpdateFailed
)

// UpdateOutcome is the result for one installed toolchain. Replacement names
// the channel head the update resolved; it stays empty when none was resolved.
type UpdateOutcome struct {
	Name        string
	Replacement string
	Status      UpdateStatus
	Err         error
}

// UpdateReport aggregates UpdateAll over every installed toolchain.
type UpdateReport struct {
	// NoneInstalled is set when there was nothing to update.
	NoneInstalled bool
	Outcomes      []UpdateOutcome
}

// UpdateInstalled brings one installed toolchain to its channel head. A
// channel name ("lts") updates its tracking identity and cross SDKs, installing
// a missing channel. Explicit host and target versions are installed when
// missing and otherwise remain fixed.
func UpdateInstalled(ctx context.Context, name toolchain.ToolchainName, opts Options) (UpdateOutcome, error) {
	if name.IsCustom() {
		return UpdateOutcome{}, errors.New(i18n.T("UpdateCustomToolchain", i18n.MsgData{"Name": name.String()}))
	}
	switch {
	case name.Target != "":
		// Versioned target names are fixed versions just like host names.
		if err := Install(ctx, InstallRequest{Toolchain: name.String()}, opts); err != nil {
			return UpdateOutcome{}, err
		}
		return UpdateOutcome{Name: name.String(), Status: UpdatePinned}, nil
	case name.IsChannelOnly():
		d, err := OpenDistribution(opts)
		if err != nil {
			return UpdateOutcome{}, err
		}
		if err := recordLegacyInstallations(ctx, d, false); err != nil {
			return UpdateOutcome{}, err
		}
		resolved, err := d.Resolve(ctx, name, "")
		if err != nil {
			return UpdateOutcome{}, err
		}
		old := d.Settings.Installations[name.String()]
		if err := installSelected(ctx, d, resolved, true, false, true, opts); err != nil {
			return UpdateOutcome{}, err
		}
		outcome := UpdateOutcome{Name: old, Replacement: resolved.Name, Status: UpdateApplied}
		if old == "" {
			outcome.Name = name.String()
		}
		if old == resolved.Name {
			outcome.Status = UpdateUpToDate
			opts.emit(progress.Event{Kind: progress.AlreadyUpToDate, Toolchain: old})
		}
		return outcome, updateTrackedTargets(ctx, d, name.Channel, opts)
	default:
		// Specific version: install it (already installed is a no-op).
		if err := Install(ctx, InstallRequest{Toolchain: name.String()}, opts); err != nil {
			return UpdateOutcome{}, err
		}
		return UpdateOutcome{Name: name.String(), Status: UpdatePinned}, nil
	}
}

// UpdateAll updates recorded tracking identities, retaining explicit and legacy
// versions and skipping custom/linked SDKs. A failure does not stop the loop: the
// joined error is returned with the report, and the downloads staging area is
// purged either way.
func UpdateAll(ctx context.Context, opts Options) (UpdateReport, error) {
	defer func() {
		if n, purgeErr := purgeDownloadsDir(); purgeErr != nil {
			slog.Warn("failed to purge downloads dir", "removed", n, "error", purgeErr)
		} else if n > 0 {
			slog.Debug("purged downloads dir", "removed", n)
		}
	}()

	installed, err := toolchain.ListInstalled()
	if err != nil {
		return UpdateReport{}, err
	}
	if len(installed) == 0 {
		return UpdateReport{NoneInstalled: true}, nil
	}

	d, err := OpenDistribution(opts)
	if err != nil {
		return UpdateReport{}, err
	}
	if err := recordLegacyInstallations(ctx, d, true); err != nil {
		return UpdateReport{}, err
	}
	var report UpdateReport
	var errs []error
	var identities []string
	tracked := make(map[string]bool)
	for identity, current := range d.Settings.Installations {
		if isTrackingIdentity(identity, current) {
			identities = append(identities, identity)
			tracked[current] = true
		}
	}
	slices.Sort(identities)
	for _, identity := range identities {
		current := d.Settings.Installations[identity]
		parsed, err := toolchain.ParseToolchainName(current)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		outcome, err := upgradeChannelToolchain(ctx, d, parsed.Channel, current, parsed.Target, opts)
		if err != nil {
			errs = append(errs, err)
		}
		report.Outcomes = append(report.Outcomes, outcome)
	}
	for _, name := range installed {
		if tracked[name] {
			continue
		}
		parsed, err := toolchain.ParseToolchainName(name)
		if err != nil {
			slog.Warn("skipping toolchain", "name", name, "error", err)
			continue
		}
		if parsed.IsCustom() || parsed.Channel == toolchain.UnknownChannel {
			report.Outcomes = append(report.Outcomes, UpdateOutcome{Name: name, Status: UpdateSkipped})
			continue
		}
		report.Outcomes = append(report.Outcomes, UpdateOutcome{Name: name, Status: UpdatePinned})
	}
	return report, errors.Join(errs...)
}

func upgradeChannelToolchain(ctx context.Context, d *Distribution, channel toolchain.Channel, currentName, tuple string, opts Options) (UpdateOutcome, error) {
	resolved, err := d.Resolve(ctx, toolchain.ToolchainName{Channel: channel}, tuple)
	if err != nil {
		return UpdateOutcome{Name: currentName, Status: UpdateFailed, Err: err}, err
	}
	outcome := UpdateOutcome{Name: currentName, Replacement: resolved.Name, Status: UpdateUpToDate}
	opts.tracking = trackingIdentity(toolchain.ToolchainName{Channel: channel, Target: tuple})
	updated, err := upgradeToolchain(ctx, currentName, resolved, d, opts)
	if err != nil {
		outcome.Status, outcome.Err = UpdateFailed, err
		return outcome, err
	}
	if updated {
		outcome.Status = UpdateApplied
	}
	return outcome, nil
}

// installedForChannel returns the newest installed host version of channel,
// or "" when none is installed. Target variants are not considered.
func installedForChannel(channel toolchain.Channel) (string, error) {
	dir, err := toolchain.FindInstalled(toolchain.ToolchainName{Channel: channel})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return filepath.Base(dir), nil
}
