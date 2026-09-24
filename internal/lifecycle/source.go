package lifecycle

import (
	"context"
	"sync"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
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
// source, including the configured dist_server root.
func NewManifestFetcherForSettings(settings *config.Settings, opts Options) (*ManifestFetcher, error) {
	source, err := newDistributionSource(settings)
	if err != nil {
		return nil, err
	}
	return &ManifestFetcher{source: source, opts: opts}, nil
}

func newDistributionSource(settings *config.Settings) (*dist.Source, error) {
	return dist.NewSource(settings)
}

func (f *ManifestFetcher) Get(ctx context.Context) (*dist.Manifest, error) {
	f.Note()
	return f.source.Manifest(ctx)
}

// Note emits the operation-scoped manifest progress message once.
func (f *ManifestFetcher) Note() {
	f.noteOnce.Do(func() { f.opts.report("FetchingManifest", nil) })
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
	fetcher.Note()
	release, err := fetcher.source.ResolveToolchain(ctx, name.Channel, name.Version, tuple)
	if err != nil {
		return ResolvedToolchain{}, err
	}
	resolved := toolchain.ToolchainName{Channel: release.Channel, Version: release.Version}
	if id, parseErr := sdktarget.ParseIdentity(tuple); parseErr == nil && id.IsTargetVariant() {
		resolved.Target = tuple
	}
	if release.Channel == toolchain.Nightly && release.Download.SHA256 == "" {
		fetcher.opts.report("NightlyNoChecksum", nil)
	}
	result := ResolvedToolchain{
		Name:        resolved.String(),
		URL:         release.Download.URL,
		SHA256:      release.Download.SHA256,
		ArchiveName: release.Download.Name,
		Tuple:       tuple,
	}
	return result, nil
}
