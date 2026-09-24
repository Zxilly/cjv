package lifecycle

import (
	"context"
	"sync"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// Distribution is what one operation resolves against: the user settings it
// was opened from, the distribution source those settings select, and the
// host tuple. `install`, `update`, `check`, `list-remote` and component
// installs all open it the same way, so they agree on the manifest, the
// dist_server root and the platform. Source metadata is fetched at most once
// per Distribution, and the manifest progress note is emitted at most once.
type Distribution struct {
	File      *config.SettingsFile
	Settings  *config.Settings
	Source    *dist.Source
	HostTuple string

	opts     Options
	noteOnce sync.Once
}

// OpenDistribution loads the user settings and resolves the distribution
// source and host tuple they select. opts carries the progress reporter for
// the operation's manifest note.
func OpenDistribution(opts Options) (*Distribution, error) {
	sf, settings, err := config.LoadDefaultSettings()
	if err != nil {
		return nil, err
	}
	source, err := dist.NewSource(settings)
	if err != nil {
		return nil, err
	}
	hostTuple, err := sdktarget.CurrentHostTuple(settings.DefaultHost)
	if err != nil {
		return nil, err
	}
	return &Distribution{File: sf, Settings: settings, Source: source, HostTuple: hostTuple, opts: opts}, nil
}

// TargetTuple composes the host tuple with a cross-compile environment such
// as "ohos". An empty environment yields the host tuple; a full tuple passed
// as environment is rejected.
func (d *Distribution) TargetTuple(environment string) (string, error) {
	return sdktarget.CurrentTargetTuple(d.Settings.DefaultHost, environment)
}

// note emits the operation-scoped manifest progress message once.
func (d *Distribution) note() {
	d.noteOnce.Do(func() { d.opts.report("FetchingManifest", nil) })
}

// Resolve turns a channel, version or channel-version request into the
// concrete release the source publishes for tuple. An empty tuple means the
// host; a target tuple yields a target variant name.
func (d *Distribution) Resolve(ctx context.Context, name toolchain.ToolchainName, tuple string) (ResolvedToolchain, error) {
	if tuple == "" {
		tuple = d.HostTuple
	}
	d.note()
	release, err := d.Source.ResolveToolchain(ctx, name.Channel, name.Version, tuple)
	if err != nil {
		return ResolvedToolchain{}, err
	}
	resolved := toolchain.ToolchainName{Channel: release.Channel, Version: release.Version}
	if id, parseErr := sdktarget.ParseIdentity(tuple); parseErr == nil && id.IsTargetVariant() {
		resolved.Target = tuple
	}
	if release.Channel == toolchain.Nightly && release.Download.SHA256 == "" {
		d.opts.report("NightlyNoChecksum", nil)
	}
	return ResolvedToolchain{
		Name:        resolved.String(),
		URL:         release.Download.URL,
		SHA256:      release.Download.SHA256,
		ArchiveName: release.Download.Name,
		Tuple:       tuple,
	}, nil
}
