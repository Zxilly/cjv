package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/progress"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// groupMember is a compatibility plan against one immutable installed snapshot.
// Scratch paths belong to preparedGroupMember and never enter read-only plans.
type groupMember struct {
	identity  string
	installed installationSnapshot
	release   ResolvedToolchain
	intents   []component.Intent
	drop      bool
	reinstall bool
}

type preparedGroupMember struct {
	*groupMember
	prepared component.Roots
}

func (m *groupMember) componentsChanged() bool {
	return !slices.Equal(m.installed.intents, m.intents)
}

// canReuseSDK must agree during component selection and preparation: a fresh
// SDK also needs fresh copies of its published components.
func (m *groupMember) canReuseSDK() bool {
	return !m.reinstall && m.installed.record != nil && m.installed.record.Release == m.release.Name &&
		(m.installed.record.Tuple == "" || m.installed.record.Tuple == m.release.Tuple) &&
		(m.release.URL == "" || strings.EqualFold(m.installed.record.SHA256, m.release.SHA256))
}

// installGroup prepares a host and all its tracking targets before changing any
// installed root. The home lock is only held for snapshots and publication.
func installGroup(ctx context.Context, d *installationDistribution, name toolchain.ToolchainName, rt ResolvedToolchain, req InstallRequest, opts Options) (bool, error) {
	tracking := name.Version == ""
	trackingHost := tracking && name.Target == ""
	if name.Target != "" {
		opts.preserveDefault = true
	}
	identity := selectedIdentity(rt, tracking)
	home, err := config.Home()
	if err != nil {
		return false, err
	}
	if err := config.EnsureDirs(); err != nil {
		return false, err
	}
	downloads, err := config.DownloadsDir()
	if err != nil {
		return false, err
	}
	preparation, err := dist.BeginPreparation(ctx, downloads)
	if err != nil {
		return false, err
	}
	defer preparation.Close() //nolint:errcheck // best-effort scratch cleanup
	lock, err := toolchain.LockHome(ctx)
	if err != nil {
		return false, err
	}
	if err := lock.Recover(); err != nil {
		_ = lock.Close()
		return false, err
	}
	if err := validateDependencies(opts); err != nil {
		_ = lock.Close()
		return false, err
	}
	var members []*groupMember
	snapshot := func(memberIdentity string) (*groupMember, error) {
		m, err := snapshotGroupMember(memberIdentity)
		if err != nil {
			return nil, err
		}
		m.reinstall = req.Reinstall
		if old, existed := d.installed[memberIdentity]; existed {
			if m.installed.record == nil {
				return nil, fmt.Errorf("toolchain %s was removed during installation; retry", memberIdentity)
			}
			if old != *m.installed.record {
				return nil, fmt.Errorf("toolchain %s changed during installation; retry", memberIdentity)
			}
		}
		return m, m.installed.captureMetadata()
	}
	host, err := snapshot(identity)
	if err != nil {
		_ = lock.Close()
		return false, err
	}
	if !req.Reinstall && (req.NoUpdate || !tracking) && host.installed.record != nil {
		release, err := toolchain.ParseToolchainName(host.installed.record.Release)
		if err != nil {
			_ = lock.Close()
			return false, err
		}
		rt.Name, rt.Tuple, rt.SHA256, rt.URL = release.String(), host.installed.record.Tuple, host.installed.record.SHA256, ""
		if rt.Tuple == "" {
			rt.Tuple = release.Target
			if rt.Tuple == "" {
				rt.Tuple = release.Host
			}
			if rt.Tuple == "" {
				rt.Tuple = d.HostTuple
			}
		}
	}
	host.release = rt
	members = append(members, host)
	tracked, err := trackedGroupTargets(name.Channel, rt.Tuple, trackingHost)
	if err != nil {
		_ = lock.Close()
		return false, err
	}
	targetNames := append([]string(nil), tracked...)
	for _, target := range req.Targets {
		tuple, err := sdktargetTuple(rt.Tuple, target)
		if err != nil {
			_ = lock.Close()
			return false, err
		}
		child := name
		child.Target = tuple
		if !tracking {
			concrete, _ := toolchain.ParseToolchainName(rt.Name)
			child.Channel, child.Version = concrete.Channel, concrete.Version
		}
		targetNames = append(targetNames, child.String())
	}
	slices.Sort(targetNames)
	targetNames = slices.Compact(targetNames)
	for _, targetName := range targetNames {
		m, err := snapshot(targetName)
		if err != nil {
			_ = lock.Close()
			return false, err
		}
		members = append(members, m)
	}
	if err := lock.Close(); err != nil {
		return false, err
	}
	requested, err := component.NormalizeList(req.Components)
	if err != nil {
		return false, err
	}
	for i, m := range members {
		if len(req.Targets) > 0 && i == 0 {
			continue
		}
		for _, c := range requested {
			if !slices.ContainsFunc(m.intents, func(intent component.Intent) bool { return intent.Name == c }) {
				m.intents = append(m.intents, component.Intent{Name: c})
			}
		}
	}
	selection := name
	if req.Reinstall && host.installed.record != nil {
		// Repair selects the recorded artifact even when a newer compatible
		// nightly exists. Keep tracking identities while pinning this solve.
		concrete, err := toolchain.ParseToolchainName(host.release.Name)
		if err != nil {
			return false, err
		}
		selection.Version = concrete.Version
	}
	if err := resolveGroup(ctx, d.Distribution, selection, members, opts); err != nil {
		return false, err
	}
	stage, err := preparation.TempDir(".cjv-group-*")
	if err != nil {
		return false, err
	}
	var changed []*preparedGroupMember
	for i, member := range members {
		m := &preparedGroupMember{groupMember: member}
		if m.drop {
			if m.installed.record != nil {
				changed = append(changed, m)
			}
			continue
		}
		reuse := m.canReuseSDK()
		if reuse {
			if !m.componentsChanged() {
				continue
			}
		}
		base := filepath.Join(stage, fmt.Sprint(i))
		if m.installed.record != nil && m.installed.record.Release != m.release.Name {
			opts.emit(progress.Event{Kind: progress.UpdateFound, Toolchain: m.identity, Replacement: m.release.Name})
		}
		m.prepared = component.Roots{TcDir: filepath.Join(base, "sdk"), DocsDir: filepath.Join(base, "docs"), StdxDir: filepath.Join(base, "stdx")}
		if !reuse {
			archive, err := preparation.Download(ctx, m.release.URL, m.release.SHA256, m.release.ArchiveName, opts.sink())
			if err != nil {
				return false, err
			}
			opts.emit(progress.Event{Kind: progress.Extracting})
			if err := dist.InstallSDK(ctx, archive, m.prepared.TcDir); err != nil {
				return false, err
			}
			if err := validateInstallation(m.prepared.TcDir, m.release.Tuple); err != nil {
				return false, err
			}
			if err := toolchain.WriteInstallation(m.prepared.TcDir, toolchain.Installation{Release: m.release.Name, Tuple: m.release.Tuple, SHA256: m.release.SHA256}); err != nil {
				return false, err
			}
		}
		concrete, _ := toolchain.ParseToolchainName(m.release.Name)
		for _, intent := range m.intents {
			if reuse && slices.ContainsFunc(m.installed.intents, func(old component.Intent) bool { return old == intent }) {
				continue
			}
			if intent.Source != "" {
				_, err = component.Link(m.prepared, intent.Name, intent.Source, false)
			} else {
				err = component.PrepareFromSource(ctx, m.prepared, concrete, intent.Name, m.release.Tuple, false, d.Source, preparation, opts.sink())
			}
			if err != nil {
				return false, err
			}
		}
		changed = append(changed, m)
	}
	lock, err = toolchain.LockHome(ctx)
	if err != nil {
		return false, err
	}
	defer lock.Close() //nolint:errcheck
	if err := lock.Recover(); err != nil {
		return false, err
	}
	if err := validateDependencies(opts); err != nil {
		return false, err
	}
	nowTargets, err := trackedGroupTargets(name.Channel, rt.Tuple, trackingHost)
	if err != nil {
		return false, err
	}
	if !slices.Equal(nowTargets, tracked) {
		return false, fmt.Errorf("toolchain %s targets changed during installation; retry", identity)
	}
	for _, m := range members {
		if !m.installed.matches() {
			return false, fmt.Errorf("toolchain %s changed during installation; retry", m.identity)
		}
	}
	publishDefault := func() error {
		if opts.preserveDefault {
			return nil
		}
		return publishFirstDefault(d.Distribution, identity, opts)
	}
	if len(changed) == 0 {
		if err := publishDefault(); err != nil {
			return false, err
		}
		kind := progress.AlreadyUpToDate
		if !tracking {
			kind = progress.ToolchainAlreadyInstalled
		}
		opts.emit(progress.Event{Kind: kind, Toolchain: identity})
		return false, nil
	}
	var publications []toolchainPublication
	for _, m := range changed {
		roots, err := component.RootsFor(m.identity)
		if err != nil {
			return false, err
		}
		if !m.drop && !m.canReuseSDK() && d.Settings.LinkMode == "hardlink" {
			if err := deduplicateSDK(m.prepared.TcDir, m.identity); err != nil {
				return false, err
			}
		}
		publication := toolchainPublication{identity: m.identity, roots: toolchainRoots(m.prepared, roots), removed: m.drop}
		if !m.drop && m.canReuseSDK() {
			publication.roots, err = prepareGroupComponents(preparation, m, roots)
			if err != nil {
				return false, err
			}
		}
		publications = append(publications, publication)
	}
	if trackingHost && !req.Reinstall && host.installed.record != nil && slices.ContainsFunc(changed, func(m *preparedGroupMember) bool {
		return m.installed.record != nil && (m.drop || !m.canReuseSDK())
	}) {
		if err := lock.WithStableReads(ctx, func() error {
			var err error
			opts.history, err = prepareHistorySnapshot(ctx, home, identity, rt.Tuple, members)
			return err
		}); err != nil {
			if opts.history != nil {
				opts.history.cleanup(err)
			}
			return false, err
		}
	}
	publishErr := publishPrepared(ctx, home, publications, publishDefault, opts)
	if opts.history != nil {
		opts.history.cleanup(publishErr)
	}
	if publishErr != nil {
		return false, publishErr
	}
	// Advance only the records this operation published. Other snapshots must
	// continue detecting changes made by concurrent commands.
	for _, m := range changed {
		if m.drop {
			delete(d.installed, m.identity)
		} else {
			d.installed[m.identity] = toolchain.Installation{Release: m.release.Name, Tuple: m.release.Tuple, SHA256: m.release.SHA256}
		}
	}
	preparation.Complete()
	if opts.history != nil {
		if err := lock.WithStableReads(ctx, func() error {
			_, err := pruneHistoryLocked(ctx, home, identity, DefaultHistoryRetention)
			return err
		}); err != nil {
			slog.Warn("could not prune toolchain history", "toolchain", identity, "error", err)
		}
	}
	return true, nil
}

func trackedGroupTargets(channel toolchain.Channel, hostTuple string, tracking bool) ([]string, error) {
	if !tracking {
		return nil, nil
	}
	installed, err := toolchain.ListInstalled()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, identity := range installed {
		name, err := toolchain.ParseToolchainName(identity)
		if err == nil && name.Channel == channel && name.Version == "" && name.Target != "" && strings.HasPrefix(name.Target, hostTuple+"-") {
			names = append(names, identity)
		}
	}
	return names, nil
}

func sdktargetTuple(host, suffix string) (string, error) {
	id, err := sdktarget.ParseIdentity(host)
	if err != nil {
		return "", err
	}
	target, err := id.WithEnvironment(suffix)
	return target.Tuple(), err
}

func resolveGroup(ctx context.Context, d *Distribution, name toolchain.ToolchainName, members []*groupMember, opts Options) error {
	return resolveGroupPlan(ctx, d, name, members, opts, nil)
}

// resolveGroupPlan is the metadata-only solver shared by installation and
// inspection. notes explains why a newer candidate cannot be selected.
func resolveGroupPlan(ctx context.Context, d *Distribution, name toolchain.ToolchainName, members []*groupMember, opts Options, notes *[]string) error {
	host := members[0].release
	concrete, err := toolchain.ParseToolchainName(host.Name)
	if err != nil {
		return err
	}
	versions := []string{concrete.Version}
	if name.Channel == toolchain.Nightly && name.Version == "" && name.Target == "" && host.URL != "" && !opts.AllowMissing {
		_, versions, err = d.Source.ChannelVersions(ctx, name.Channel, host.Tuple)
		if err != nil {
			return err
		}
	}
	var unavailable error
	for _, version := range versions {
		if before := members[0].installed.record; before != nil && !opts.AllowDowngrade {
			old, _ := toolchain.ParseToolchainName(before.Release)
			if versionLess(version, old.Version) {
				break
			}
		}
		candidate := concrete
		candidate.Version = version
		if version != concrete.Version {
			members[0].release, err = d.Resolve(ctx, candidate, host.Tuple)
			if err != nil {
				return err
			}
		}
		unavailable = nil
		for i, m := range members {
			if i > 0 {
				child, _ := toolchain.ParseToolchainName(m.identity)
				if host.URL == "" && m.installed.record != nil {
					previous, parseErr := toolchain.ParseToolchainName(m.installed.record.Release)
					if parseErr != nil {
						return parseErr
					}
					if previous.Version == candidate.Version {
						m.release = ResolvedToolchain{Name: m.installed.record.Release, Tuple: child.Target, SHA256: m.installed.record.SHA256}
						err = nil
					} else {
						m.release, err = d.Resolve(ctx, candidate, child.Target)
					}
				} else {
					m.release, err = d.Resolve(ctx, candidate, child.Target)
				}
				if err != nil && opts.AllowMissing && isUnavailable(err) {
					slog.Warn("dropping unavailable target", "toolchain", m.identity, "release", version)
					m.drop = true
					continue
				}
				if err != nil {
					unavailable = err
					break
				}
			}
			var selected []component.Intent
			for _, intent := range m.intents {
				preserved := m.canReuseSDK() && slices.ContainsFunc(m.installed.intents, func(old component.Intent) bool { return old == intent })
				if intent.Source == "" && !preserved {
					platform := ""
					if intent.Name == component.Stdx {
						id, parseErr := sdktarget.ParseIdentity(m.release.Tuple)
						if parseErr != nil {
							return parseErr
						}
						platform, err = id.StdxPlatform()
						if err != nil {
							return err
						}
					}
					_, err = d.Source.ResolveComponent(ctx, candidate.Channel, version, string(intent.Name), platform)
					if err != nil && opts.AllowMissing && isUnavailable(err) {
						slog.Warn("dropping unavailable component", "toolchain", m.identity, "component", intent.Name, "release", version)
						continue
					}
					if err != nil {
						unavailable = err
						break
					}
				}
				selected = append(selected, intent)
			}
			if unavailable != nil {
				break
			}
			if opts.AllowMissing {
				m.intents = selected
			}
		}
		if unavailable == nil {
			return nil
		}
		if notes != nil && !slices.Contains(*notes, unavailable.Error()) {
			*notes = append(*notes, unavailable.Error())
		}
		if !isUnavailable(unavailable) {
			return unavailable
		}
	}
	if unavailable == nil {
		unavailable = fmt.Errorf("no compatible release without downgrading %s", members[0].identity)
	}
	return unavailable
}

func isUnavailable(err error) bool {
	return errors.As(err, new(*cjverr.ComponentNotPublishedError)) || errors.As(err, new(*cjverr.VersionNotAvailableError)) || errors.As(err, new(*cjverr.VersionNotFoundError))
}

func versionLess(a, b string) bool {
	return toolchain.CompareVersions(a, b) < 0
}

func selectedIdentity(rt ResolvedToolchain, tracking bool) string {
	if !tracking {
		return rt.Name
	}
	name, _ := toolchain.ParseToolchainName(rt.Name)
	name.Version = ""
	return name.String()
}
