package lifecycle

import (
	"context"
	"errors"
	"log/slog"

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
	case name.Target != "" && name.Version != "":
		// Versioned target names are fixed versions just like host names.
		if err := Install(ctx, InstallRequest{Toolchain: name.String()}, opts); err != nil {
			return UpdateOutcome{}, err
		}
		return UpdateOutcome{Name: name.String(), Status: UpdatePinned}, nil
	case name.Version == "":
		d, err := OpenDistribution(opts)
		if err != nil {
			return UpdateOutcome{}, err
		}
		selector := name
		if name.Target != "" {
			selector, opts, err = selectTargetHost(ctx, name, opts)
			if err != nil {
				return UpdateOutcome{}, err
			}
		}
		resolved, err := d.Resolve(ctx, selector, name.Target)
		if err != nil {
			return UpdateOutcome{}, err
		}
		old := toolchain.Installation{}
		if dir, err := toolchain.FindInstalled(name); err == nil {
			record, err := toolchain.ReadInstallation(dir)
			if err != nil {
				return UpdateOutcome{}, err
			}
			old = record
		}
		if err := installSelected(ctx, d, resolved, true, false, name.Target == "", opts); err != nil {
			return UpdateOutcome{}, err
		}
		outcome := UpdateOutcome{Name: name.String(), Replacement: resolved.Name, Status: UpdateApplied}
		if old.Release == resolved.Name && old.Tuple == resolved.Tuple && old.SHA256 == resolved.SHA256 {
			outcome.Status = UpdateUpToDate
			opts.emit(progress.Event{Kind: progress.AlreadyUpToDate, Toolchain: name.String()})
		}
		if name.Target != "" {
			return outcome, nil
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
		if n, purgeErr := purgeDownloadsDirContext(ctx); purgeErr != nil {
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
	// OpenDistribution has completed the one-time legacy migration.
	installed, err = toolchain.ListInstalled()
	if err != nil {
		return UpdateReport{}, err
	}
	var report UpdateReport
	var errs []error
	for _, identity := range installed {
		parsed, err := toolchain.ParseToolchainName(identity)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if parsed.IsCustom() || parsed.Channel == toolchain.UnknownChannel {
			report.Outcomes = append(report.Outcomes, UpdateOutcome{Name: identity, Status: UpdateSkipped})
			continue
		}
		if parsed.Version != "" {
			report.Outcomes = append(report.Outcomes, UpdateOutcome{Name: identity, Status: UpdatePinned})
			continue
		}
		outcome, err := upgradeChannelToolchain(ctx, d, parsed.Channel, identity, parsed.Target, opts)
		report.Outcomes = append(report.Outcomes, outcome)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return report, errors.Join(errs...)
}

func upgradeChannelToolchain(ctx context.Context, d *Distribution, channel toolchain.Channel, currentName, tuple string, opts Options) (UpdateOutcome, error) {
	selector := toolchain.ToolchainName{Channel: channel}
	if tuple != "" {
		var err error
		selector, opts, err = selectTargetHost(ctx, selector, opts)
		if err != nil {
			return UpdateOutcome{}, err
		}
	}
	resolved, err := d.Resolve(ctx, selector, tuple)
	if err != nil {
		return UpdateOutcome{Name: currentName, Status: UpdateFailed, Err: err}, err
	}
	outcome := UpdateOutcome{Name: currentName, Replacement: resolved.Name, Status: UpdateUpToDate}
	dir, err := toolchain.FindInstalled(toolchain.ToolchainName{Channel: channel, Target: tuple})
	if err != nil {
		return outcome, err
	}
	current, err := toolchain.ReadInstallation(dir)
	if err != nil {
		return outcome, err
	}
	if current.Release == resolved.Name && current.Tuple == resolved.Tuple && current.SHA256 == resolved.SHA256 {
		return outcome, nil
	}
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
