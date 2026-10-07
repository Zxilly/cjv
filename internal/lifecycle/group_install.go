package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/fsops"
	"github.com/Zxilly/cjv/internal/progress"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
	goversion "github.com/hashicorp/go-version"
)

type groupMember struct {
	identity        string
	release         ResolvedToolchain
	before          *toolchain.Installation
	state           string
	intents         []component.Intent
	original        []component.Intent
	prepared        component.Roots
	addedComponents bool
	drop            bool
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
	installLock, err := fsops.LockFile(ctx, filepath.Join(home, ".install.lock"))
	if err != nil {
		return false, err
	}
	defer installLock.Close() //nolint:errcheck
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
		roots, err := component.RootsFor(memberIdentity)
		if err != nil {
			return nil, err
		}
		m := &groupMember{identity: memberIdentity}
		record, err := toolchain.ReadInstallation(roots.TcDir)
		if errors.Is(err, os.ErrNotExist) {
			if _, existed := d.installed[memberIdentity]; existed {
				return nil, fmt.Errorf("toolchain %s was removed during installation; retry", memberIdentity)
			}
			return m, nil
		}
		if err != nil {
			return nil, err
		}
		if old, existed := d.installed[memberIdentity]; existed && old != record {
			return nil, fmt.Errorf("toolchain %s changed during installation; retry", memberIdentity)
		}
		m.before = &record
		m.state, err = installationComponentState(roots.TcDir)
		if err != nil {
			return nil, err
		}
		m.intents, err = component.InstalledIntents(roots)
		m.original = slices.Clone(m.intents)
		return m, err
	}
	host, err := snapshot(identity)
	if err != nil {
		_ = lock.Close()
		return false, err
	}
	if (req.NoUpdate || !tracking) && host.before != nil {
		release, err := toolchain.ParseToolchainName(host.before.Release)
		if err != nil {
			_ = lock.Close()
			return false, err
		}
		rt.Name, rt.Tuple, rt.SHA256, rt.URL = release.String(), host.before.Tuple, host.before.SHA256, ""
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
				m.addedComponents = true
			}
		}
	}
	if err := resolveGroup(ctx, d.Distribution, name, members, opts); err != nil {
		return false, err
	}
	downloads, err := config.DownloadsDir()
	if err != nil {
		return false, err
	}
	stage, err := os.MkdirTemp(downloads, ".cjv-group-*")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(stage) //nolint:errcheck
	var changed []*groupMember
	var archives []string
	for i, m := range members {
		if m.drop {
			if m.before != nil {
				changed = append(changed, m)
			}
			continue
		}
		reuse := m.before != nil && m.before.Release == m.release.Name && (m.before.Tuple == "" || m.before.Tuple == m.release.Tuple) && (m.release.URL == "" || m.before.SHA256 == m.release.SHA256)
		if reuse {
			if !m.addedComponents {
				continue
			}
		}
		base := filepath.Join(stage, fmt.Sprint(i))
		if m.before != nil && m.before.Release != m.release.Name {
			opts.emit(progress.Event{Kind: progress.UpdateFound, Toolchain: m.identity, Replacement: m.release.Name})
		}
		m.prepared = component.Roots{TcDir: filepath.Join(base, "sdk"), DocsDir: filepath.Join(base, "docs"), StdxDir: filepath.Join(base, "stdx")}
		if reuse {
			roots, err := component.RootsFor(m.identity)
			if err != nil {
				return false, err
			}
			if err := fsops.CopyTree(roots.TcDir, m.prepared.TcDir); err != nil {
				return false, err
			}
			for _, pair := range [][2]string{{roots.DocsDir, m.prepared.DocsDir}, {roots.StdxDir, m.prepared.StdxDir}} {
				if _, err := os.Lstat(pair[0]); err == nil {
					if err := fsops.CopyTree(pair[0], pair[1]); err != nil {
						return false, err
					}
				} else if !errors.Is(err, os.ErrNotExist) {
					return false, err
				}
			}
			for _, intent := range m.original {
				if !slices.ContainsFunc(m.intents, func(selected component.Intent) bool { return selected.Name == intent.Name }) {
					if err := component.Remove(m.prepared, intent.Name); err != nil {
						return false, err
					}
				}
			}
		} else {
			archive, err := dist.DownloadCachedWithName(ctx, m.release.URL, m.release.SHA256, downloads, m.release.ArchiveName, opts.sink())
			if err != nil {
				return false, err
			}
			archives = append(archives, archive)
			opts.emit(progress.Event{Kind: progress.Extracting})
			if err := dist.InstallSDK(ctx, archive, m.prepared.TcDir); err != nil {
				return false, err
			}
		}
		if err := validateInstallation(m.prepared.TcDir, m.release.Tuple); err != nil {
			return false, err
		}
		if err := toolchain.WriteInstallation(m.prepared.TcDir, toolchain.Installation{Release: m.release.Name, Tuple: m.release.Tuple, SHA256: m.release.SHA256}); err != nil {
			return false, err
		}
		concrete, _ := toolchain.ParseToolchainName(m.release.Name)
		for _, intent := range m.intents {
			if reuse && slices.ContainsFunc(m.original, func(old component.Intent) bool { return old == intent }) {
				continue
			}
			if intent.Source != "" {
				_, err = component.Link(m.prepared, intent.Name, intent.Source, false)
			} else {
				err = component.InstallFromSource(ctx, m.prepared, concrete, intent.Name, m.release.Tuple, downloads, false, d.Source, opts.sink())
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
		roots, err := component.RootsFor(m.identity)
		if err != nil {
			return false, err
		}
		current, err := toolchain.ReadInstallation(roots.TcDir)
		if m.before == nil {
			if !errors.Is(err, os.ErrNotExist) {
				return false, fmt.Errorf("toolchain %s changed during installation; retry", m.identity)
			}
		} else if state, stateErr := installationComponentState(roots.TcDir); err != nil || current != *m.before || stateErr != nil || state != m.state {
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
	var pairs [][2]string
	var identities []string
	for _, m := range changed {
		roots, err := component.RootsFor(m.identity)
		if err != nil {
			return false, err
		}
		if !m.drop && d.Settings.LinkMode == "hardlink" {
			if err := deduplicateSDK(m.prepared.TcDir, m.identity); err != nil {
				return false, err
			}
		}
		identities = append(identities, m.identity)
		pairs = append(pairs, [2]string{m.prepared.TcDir, roots.TcDir}, [2]string{m.prepared.DocsDir, roots.DocsDir}, [2]string{m.prepared.StdxDir, roots.StdxDir})
	}
	if err := publishPrepared(home, identities, pairs, publishDefault, opts); err != nil {
		return false, err
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
	for _, archive := range archives {
		_ = dist.CleanupDownload(archive)
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
		if before := members[0].before; before != nil && !opts.AllowDowngrade {
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
				if host.URL == "" && m.before != nil {
					previous, parseErr := toolchain.ParseToolchainName(m.before.Release)
					if parseErr != nil {
						return parseErr
					}
					if previous.Version == candidate.Version {
						m.release = ResolvedToolchain{Name: m.before.Release, Tuple: child.Target, SHA256: m.before.SHA256}
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
				preserved := m.before != nil && m.before.Release == m.release.Name && (m.before.Tuple == "" || m.before.Tuple == m.release.Tuple) && slices.ContainsFunc(m.original, func(old component.Intent) bool { return old == intent })
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
				m.addedComponents = !slices.Equal(m.original, selected)
			}
		}
		if unavailable == nil {
			return nil
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
	av, aerr := goversion.NewVersion(a)
	bv, berr := goversion.NewVersion(b)
	if aerr == nil && berr == nil {
		return av.LessThan(bv)
	}
	return a < b
}
