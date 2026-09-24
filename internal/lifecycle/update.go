package lifecycle

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/i18n"
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

// Applied returns the outcomes whose replacement is now installed.
func (r UpdateReport) Applied() []UpdateOutcome {
	var applied []UpdateOutcome
	for _, o := range r.Outcomes {
		if o.Status == UpdateApplied {
			applied = append(applied, o)
		}
	}
	return applied
}

// UpdateInstalled brings one installed toolchain to its channel head. A
// channel name ("lts") updates the newest installed host version of that
// channel; a target variant name ("sts-1.0.0-<tuple>") updates that variant;
// an explicit version is installed when missing and otherwise left alone.
func UpdateInstalled(ctx context.Context, name toolchain.ToolchainName, opts Options) (UpdateOutcome, error) {
	if name.IsCustom() {
		return UpdateOutcome{}, errors.New(i18n.T("UpdateCustomToolchain", i18n.MsgData{"Name": name.String()}))
	}
	switch {
	case name.Target != "":
		currentName := name.String()
		if _, err := toolchain.FindInstalled(name); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return UpdateOutcome{}, &cjverr.ToolchainNotInstalledError{Name: currentName}
			}
			return UpdateOutcome{}, err
		}
		return updateChannelToolchain(ctx, name.Channel, currentName, name.Target, opts)
	case name.IsChannelOnly():
		installed, err := installedForChannel(name.Channel)
		if err != nil {
			return UpdateOutcome{}, err
		}
		if installed == "" {
			return UpdateOutcome{}, &cjverr.ToolchainNotInstalledError{Name: name.String()}
		}
		return updateChannelToolchain(ctx, name.Channel, installed, "", opts)
	default:
		// Specific version: install it (already installed is a no-op).
		if err := Install(ctx, InstallRequest{Toolchain: name.String()}, opts); err != nil {
			return UpdateOutcome{}, err
		}
		return UpdateOutcome{Name: name.String(), Status: UpdatePinned}, nil
	}
}

// UpdateAll updates every installed channel toolchain and its target variants,
// skipping custom and linked ones. A failure does not stop the loop: the
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
	var report UpdateReport
	var errs []error
	for _, name := range installed {
		parsed, err := toolchain.ParseToolchainName(name)
		if err != nil {
			slog.Warn("skipping toolchain", "name", name, "error", err)
			continue
		}
		if parsed.IsCustom() || parsed.Channel == toolchain.UnknownChannel {
			report.Outcomes = append(report.Outcomes, UpdateOutcome{Name: name, Status: UpdateSkipped})
			continue
		}
		// Each upgrade reloads the references saved by the previous one
		// through the Distribution's SettingsFile.
		outcome, err := upgradeChannelToolchain(ctx, d, parsed.Channel, name, parsed.Target, opts)
		if err != nil {
			slog.Warn("failed to update toolchain", "name", name, "error", err)
			errs = append(errs, err)
		}
		report.Outcomes = append(report.Outcomes, outcome)
	}
	return report, errors.Join(errs...)
}

// updateChannelToolchain opens the distribution for a single-toolchain
// update, then upgrades currentName to the channel head for tuple (empty
// means the host).
func updateChannelToolchain(ctx context.Context, channel toolchain.Channel, currentName, tuple string, opts Options) (UpdateOutcome, error) {
	d, err := OpenDistribution(opts)
	if err != nil {
		return UpdateOutcome{}, err
	}
	return upgradeChannelToolchain(ctx, d, channel, currentName, tuple, opts)
}

func upgradeChannelToolchain(ctx context.Context, d *Distribution, channel toolchain.Channel, currentName, tuple string, opts Options) (UpdateOutcome, error) {
	resolved, err := d.Resolve(ctx, toolchain.ToolchainName{Channel: channel}, tuple)
	if err != nil {
		return UpdateOutcome{Name: currentName, Status: UpdateFailed, Err: err}, err
	}
	outcome := UpdateOutcome{Name: currentName, Replacement: resolved.Name, Status: UpdateUpToDate}
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
