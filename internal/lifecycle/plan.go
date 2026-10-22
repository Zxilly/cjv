package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// PlannedMember is the effect of an update on one independently installed SDK.
type PlannedMember struct {
	Name       string   `json:"name"`
	Current    string   `json:"current,omitempty"`
	Selected   string   `json:"selected,omitempty"`
	Components []string `json:"components,omitempty"`
	Removed    bool     `json:"removed,omitempty"`
}

// UpdatePlan separates the newest published release from the complete release
// group that can actually be applied. It contains metadata only: planning never
// recovers, migrates, downloads archives, publishes settings or writes caches.
type UpdatePlan struct {
	Name            string          `json:"name"`
	Current         string          `json:"current,omitempty"`
	Latest          string          `json:"latest,omitempty"`
	Selected        string          `json:"selected,omitempty"`
	Status          string          `json:"status"`
	UpdateAvailable bool            `json:"update_available"`
	Blockers        []string        `json:"blockers,omitempty"`
	Members         []PlannedMember `json:"members,omitempty"`
	Error           string          `json:"error,omitempty"`
}

type UpdatePlanReport struct {
	SchemaVersion int          `json:"schema_version"`
	Toolchains    []UpdatePlan `json:"toolchains"`
	NoneInstalled bool         `json:"none_installed,omitempty"`
}

// PlanUpdate inspects the same compatibility selection used by UpdateInstalled.
// Its result is an observation, not a reservation; installation still revalidates
// the current records and component intents before publishing.
func PlanUpdate(ctx context.Context, name toolchain.ToolchainName, opts Options) (UpdatePlan, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = dist.WithCacheReadOnly(ctx)
	d, err := OpenDistribution(opts)
	if err != nil {
		return UpdatePlan{Name: name.String(), Error: err.Error()}, err
	}
	return planUpdate(ctx, d, name, opts)
}

// PlanUpdates lists each installed tracking host with its targets as one group.
// Explicit inputs can also describe missing installations. Legacy versioned
// directories remain pinned; read-only inspection does not materialize channels.
func PlanUpdates(ctx context.Context, inputs []string, opts Options) (UpdatePlanReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = dist.WithCacheReadOnly(ctx)
	d, err := OpenDistribution(opts)
	if err != nil {
		return UpdatePlanReport{}, err
	}
	report := UpdatePlanReport{SchemaVersion: 1, Toolchains: []UpdatePlan{}}
	all := len(inputs) == 0
	if all {
		inputs, err = toolchain.ListInstalled()
		if err != nil {
			return report, err
		}
		report.NoneInstalled = len(inputs) == 0
	}
	grouped := make(map[string]bool)
	if all {
		for _, input := range inputs {
			name, err := toolchain.ParseToolchainName(input)
			if err != nil || name.IsCustom() || name.Version != "" || name.Target != "" {
				continue
			}
			member, err := snapshotPlanMember(input)
			if err != nil || member.before == nil {
				continue
			}
			tuple := member.before.Tuple
			if tuple == "" {
				tuple = d.HostTuple
				if name.Host != "" {
					tuple = name.Host
				}
			}
			targets, err := trackedGroupTargets(name.Channel, tuple, true)
			if err != nil {
				return report, err
			}
			for _, target := range targets {
				grouped[target] = true
			}
		}
	}
	var errs []error
	for _, input := range inputs {
		if grouped[input] {
			continue
		}
		name, err := toolchain.ParseToolchainName(input)
		plan := UpdatePlan{Name: input}
		if err == nil {
			plan, err = planUpdate(ctx, d, name, opts)
		}
		if err != nil {
			plan.Status, plan.Error = "error", err.Error()
			errs = append(errs, fmt.Errorf("%s: %w", input, err))
		}
		report.Toolchains = append(report.Toolchains, plan)
	}
	return report, errors.Join(errs...)
}

func planUpdate(ctx context.Context, d *Distribution, name toolchain.ToolchainName, opts Options) (UpdatePlan, error) {
	plan := UpdatePlan{Name: name.String()}
	if err := ctx.Err(); err != nil {
		return plan, err
	}
	if name.IsCustom() {
		plan.Status = "skipped"
		return plan, nil
	}
	if name.Version != "" && !toolchain.IsVersionSelector(name.Version) {
		if dir, err := toolchain.FindInstalled(name); err == nil {
			record, err := toolchain.ReadInstallation(dir)
			if err != nil {
				return plan, err
			}
			plan.Name, plan.Current, plan.Selected, plan.Status = filepath.Base(dir), record.Release, record.Release, "pinned"
			return plan, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return plan, err
		}
	}
	selector := name
	if name.Target != "" && name.Version == "" {
		host, err := targetHostIdentity(name)
		if err != nil {
			return plan, err
		}
		member, err := snapshotPlanMember(host.String())
		if err != nil {
			return plan, err
		}
		if member.before != nil {
			selector, err = selectTargetRelease(selector, host, *member.before)
			if err != nil {
				return plan, err
			}
		}
	}
	resolved, err := d.Resolve(ctx, selector, name.Target)
	if err != nil {
		return plan, err
	}
	plan.Latest = resolved.Name
	identity := selectedIdentity(resolved, name.Version == "")
	plan.Name = identity
	host, err := snapshotPlanMember(identity)
	if err != nil {
		return plan, err
	}
	host.release = resolved
	if host.before != nil {
		plan.Current = host.before.Release
	}
	members := []*groupMember{host}
	targets, err := trackedGroupTargets(name.Channel, resolved.Tuple, name.Version == "" && name.Target == "")
	if err != nil {
		return plan, err
	}
	for _, target := range targets {
		member, err := snapshotPlanMember(target)
		if err != nil {
			return plan, err
		}
		members = append(members, member)
	}
	if err := resolveGroupPlan(ctx, d, name, members, opts, &plan.Blockers); err != nil {
		if !isUnavailable(err) {
			return plan, err
		}
		if !slices.Contains(plan.Blockers, err.Error()) {
			plan.Blockers = append(plan.Blockers, err.Error())
		}
		plan.Status, plan.Selected = "blocked", plan.Current
		return plan, nil
	}
	plan.Selected = members[0].release.Name
	for _, member := range members {
		entry := PlannedMember{Name: member.identity, Selected: member.release.Name, Removed: member.drop}
		if member.before != nil {
			entry.Current = member.before.Release
		}
		for _, intent := range member.intents {
			entry.Components = append(entry.Components, string(intent.Name))
		}
		plan.Members = append(plan.Members, entry)
		if member.drop || !member.canReuseSDK() || member.addedComponents {
			plan.UpdateAvailable = true
		}
	}
	plan.Status = "up-to-date"
	if plan.UpdateAvailable {
		plan.Status = "update"
		if plan.Current == "" {
			plan.Status = "install"
		}
	} else if len(plan.Blockers) > 0 {
		plan.Status = "blocked"
	}
	return plan, nil
}

func snapshotPlanMember(identity string) (*groupMember, error) {
	roots, err := component.RootsFor(identity)
	if err != nil {
		return nil, err
	}
	member := &groupMember{identity: identity}
	record, err := toolchain.ReadInstallation(roots.TcDir)
	if errors.Is(err, os.ErrNotExist) {
		return member, nil
	}
	if err != nil {
		return nil, err
	}
	member.before = &record
	member.intents, err = component.InstalledIntents(roots)
	member.original = slices.Clone(member.intents)
	return member, err
}
