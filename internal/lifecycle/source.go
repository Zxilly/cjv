package lifecycle

import (
	"context"
	"sync"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/i18n"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// ManifestFetcher owns one operation-scoped distribution source. Source
// metadata is fetched at most once, while presentation remains in lifecycle.
type ManifestFetcher struct {
	source   *dist.Source
	opts     Options
	noteOnce sync.Once
}

func NewManifestFetcher(url string, opts Options) *ManifestFetcher {
	settings := config.DefaultSettings()
	settings.ManifestURL = url
	source, _ := newDistributionSource(&settings)
	return &ManifestFetcher{source: source, opts: opts}
}

// NewManifestFetcherForSettings builds the operation-scoped distribution
// source, including unified dist_server handling.
func NewManifestFetcherForSettings(settings *config.Settings, opts Options) (*ManifestFetcher, error) {
	source, err := newDistributionSource(settings)
	if err != nil {
		return nil, err
	}
	return &ManifestFetcher{source: source, opts: opts}, nil
}

func newDistributionSource(settings *config.Settings) (*dist.Source, error) {
	return dist.NewSourceWithOptions(settings, dist.SourceOptions{
		FetchLatestNightlyRelease: func(ctx context.Context, apiURL, apiKey string) (dist.NightlyRelease, error) {
			return FetchLatestNightlyRelease(ctx, apiURL, apiKey)
		},
		FetchNightlySHA256: func(ctx context.Context, assetURL string) (string, error) {
			return FetchNightlySHA256(ctx, assetURL)
		},
	})
}

func (f *ManifestFetcher) Get(ctx context.Context) (*dist.Manifest, error) {
	f.noteOnce.Do(func() { f.opts.note(i18n.T("FetchingManifest", nil)) })
	return f.source.Manifest(ctx)
}

// UsesManifestFor reports whether channel is resolved through this source's
// manifest. Legacy nightly is the only exception.
func (f *ManifestFetcher) UsesManifestFor(channel toolchain.Channel) bool {
	return channel != toolchain.Nightly || f.source.Unified()
}

func ResolveAndLocate(ctx context.Context, name toolchain.ToolchainName, settings *config.Settings, fetcher *ManifestFetcher) (ResolvedToolchain, error) {
	return ResolveAndLocateWithTarget(ctx, name, settings, fetcher, "")
}

func ResolveAndLocateWithTarget(ctx context.Context, name toolchain.ToolchainName, settings *config.Settings, fetcher *ManifestFetcher, target string) (ResolvedToolchain, error) {
	tuple := name.Target
	if tuple == "" {
		var err error
		tuple, err = dist.CurrentTargetTuple(settings.DefaultHost, target)
		if err != nil {
			return ResolvedToolchain{}, err
		}
	}
	return ResolveAndLocatePlatform(ctx, name, settings, fetcher, tuple)
}

func ResolveAndLocatePlatform(ctx context.Context, name toolchain.ToolchainName, settings *config.Settings, fetcher *ManifestFetcher, tuple string) (ResolvedToolchain, error) {
	if tuple == "" {
		var err error
		tuple, err = dist.CurrentHostTuple(settings.DefaultHost)
		if err != nil {
			return ResolvedToolchain{}, err
		}
	}
	if name.Channel == toolchain.Nightly && name.Version == "" && !fetcher.source.Unified() {
		fetcher.opts.note(i18n.T("FetchingNightly", nil))
	}
	release, err := fetcher.source.ResolveToolchain(ctx, name.Channel, name.Version, tuple)
	if err != nil {
		return ResolvedToolchain{}, err
	}
	resolved := toolchain.ToolchainName{Channel: release.Channel, Version: release.Version}
	if id, parseErr := sdktarget.ParseIdentity(tuple); parseErr == nil && id.IsTargetVariant() {
		resolved.Target = tuple
	}
	if release.Channel == toolchain.Nightly && release.Download.SHA256 == "" {
		fetcher.opts.note(i18n.T("NightlyNoChecksum", nil))
	}
	result := ResolvedToolchain{
		Name:        resolved.String(),
		URL:         release.Download.URL,
		SHA256:      release.Download.SHA256,
		ArchiveName: release.Download.Name,
		Tuple:       tuple,
	}
	if release.Channel == toolchain.Nightly {
		result.NightlyReleaseTag = release.ReleaseTag
		result.NightlyVersion = release.Version
	}
	return result, nil
}

// FetchNightlySHA256 is a package-level seam for tests that resolve nightly toolchains.
var FetchNightlySHA256 = dist.FetchNightlySHA256

// FetchLatestNightlyRelease is a package-level seam for tests that resolve
// pinned nightly asset versions back to their GitCode release tag.
var FetchLatestNightlyRelease = dist.FetchLatestNightlyRelease

func resolveNightly(ctx context.Context, name toolchain.ToolchainName, settings *config.Settings, tuple string, opts Options) (ResolvedToolchain, error) {
	if tuple == "" {
		var err error
		tuple, err = dist.CurrentHostTuple(settings.DefaultHost)
		if err != nil {
			return ResolvedToolchain{}, err
		}
	}
	if name.Version == "" {
		opts.note(i18n.T("FetchingNightly", nil))
	}
	source, err := newDistributionSource(settings)
	if err != nil {
		return ResolvedToolchain{}, err
	}
	release, err := source.ResolveToolchain(ctx, toolchain.Nightly, name.Version, tuple)
	if err != nil {
		return ResolvedToolchain{}, err
	}
	resolved := toolchain.ToolchainName{Channel: toolchain.Nightly, Version: release.Version}
	if id, parseErr := sdktarget.ParseIdentity(tuple); parseErr == nil && id.IsTargetVariant() {
		resolved.Target = tuple
	}
	if release.Download.SHA256 == "" {
		opts.note(i18n.T("NightlyNoChecksum", nil))
	}
	return ResolvedToolchain{
		Name:              resolved.String(),
		URL:               release.Download.URL,
		SHA256:            release.Download.SHA256,
		ArchiveName:       release.Download.Name,
		Tuple:             tuple,
		NightlyReleaseTag: release.ReleaseTag,
		NightlyVersion:    release.Version,
	}, nil
}

func resolveNightlyRelease(ctx context.Context, release dist.NightlyRelease, tuple string, opts Options) (ResolvedToolchain, error) {
	version := release.Version
	releaseTag := release.TagName
	if releaseTag == "" {
		releaseTag = version
	}
	resolved := toolchain.ToolchainName{Channel: toolchain.Nightly, Version: version}
	if id, err := sdktarget.ParseIdentity(tuple); err == nil && id.IsTargetVariant() {
		resolved.Target = tuple
	}

	url, err := (dist.NightlyRelease{TagName: releaseTag, Version: version}).DownloadURL(dist.DefaultNightlyBaseURL, tuple)
	if err != nil {
		return ResolvedToolchain{}, err
	}
	sha256, err := FetchNightlySHA256(ctx, url)
	if err != nil {
		return ResolvedToolchain{}, err
	}
	if sha256 == "" {
		opts.note(i18n.T("NightlyNoChecksum", nil))
	}
	return ResolvedToolchain{
		Name:              resolved.String(),
		URL:               url,
		SHA256:            sha256,
		Tuple:             tuple,
		NightlyReleaseTag: releaseTag,
		NightlyVersion:    version,
	}, nil
}
