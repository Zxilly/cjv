package dist

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/toolchain"
)

const distributionManifestName = "versions.json"

// Source is the configured toolchain distribution source. Without a
// dist_server it preserves cjv's legacy split source model. With a dist_server,
// that manifest becomes authoritative for every channel.
type Source struct {
	manifestURL string
	root        *url.URL
	unified     bool
	gitCodeKey  string
	fetchLatest func(context.Context, string, string) (NightlyRelease, error)
	fetchSHA256 func(context.Context, string) (string, error)

	once     sync.Once
	manifest *Manifest
	err      error
}

// SourceOptions supplies the external operations used by the legacy GitCode
// nightly adapter. Unified sources resolve nightly through their manifest.
type SourceOptions struct {
	FetchLatestNightlyRelease func(context.Context, string, string) (NightlyRelease, error)
	FetchNightlySHA256        func(context.Context, string) (string, error)
}

// NewSource resolves the active distribution source from settings.
func NewSource(settings *config.Settings) (*Source, error) {
	return NewSourceWithOptions(settings, SourceOptions{})
}

// NewSourceWithOptions resolves the source with explicit legacy-nightly
// adapters. Lifecycle uses this to retain its existing test seam.
func NewSourceWithOptions(settings *config.Settings, opts SourceOptions) (*Source, error) {
	if settings == nil {
		return nil, fmt.Errorf("distribution source requires settings")
	}
	if opts.FetchLatestNightlyRelease == nil {
		opts.FetchLatestNightlyRelease = FetchLatestNightlyRelease
	}
	if opts.FetchNightlySHA256 == nil {
		opts.FetchNightlySHA256 = FetchNightlySHA256
	}
	rootValue := settings.ResolveDistServer()
	if rootValue == "" {
		return &Source{
			manifestURL: settings.ManifestURL,
			gitCodeKey:  settings.ResolveGitCodeAPIKey(),
			fetchLatest: opts.FetchLatestNightlyRelease,
			fetchSHA256: opts.FetchNightlySHA256,
		}, nil
	}

	root, err := parseDistributionRoot(rootValue)
	if err != nil {
		return nil, err
	}
	manifestURL := root.JoinPath(distributionManifestName).String()
	return &Source{
		manifestURL: manifestURL,
		root:        root,
		unified:     true,
		fetchLatest: opts.FetchLatestNightlyRelease,
		fetchSHA256: opts.FetchNightlySHA256,
	}, nil
}

// ToolchainRelease is the source-level result for one concrete toolchain
// build. Download is ready for the shared downloader; ReleaseTag preserves the
// upstream release identity when it differs from the SDK version.
type ToolchainRelease struct {
	Channel    toolchain.Channel
	Version    string
	ReleaseTag string
	Download   DownloadInfo
}

// Unified reports whether one configured manifest is authoritative for every
// channel. The alternative is the compatibility source model.
func (s *Source) Unified() bool { return s != nil && s.unified }

// ManifestURL returns the effective manifest endpoint.
func (s *Source) ManifestURL() string {
	if s == nil {
		return ""
	}
	return s.manifestURL
}

// Manifest fetches and validates the source manifest at most once.
func (s *Source) Manifest(ctx context.Context) (*Manifest, error) {
	if s == nil {
		return nil, fmt.Errorf("distribution source is nil")
	}
	s.once.Do(func() {
		s.manifest, s.err = FetchManifest(ctx, s.manifestURL)
		if s.err == nil && s.unified {
			s.err = s.resolveManifestURLs(s.manifest)
		}
	})
	return s.manifest, s.err
}

// ResolveToolchain selects one concrete toolchain artifact. Unified sources use
// the manifest for every channel; legacy sources retain the existing GitCode
// adapter for nightly.
func (s *Source) ResolveToolchain(ctx context.Context, channel toolchain.Channel, version, tuple string) (ToolchainRelease, error) {
	if channel == toolchain.Nightly && !s.unified {
		return s.resolveLegacyNightly(ctx, version, tuple)
	}

	manifest, err := s.Manifest(ctx)
	if err != nil {
		return ToolchainRelease{}, err
	}
	if channel == toolchain.UnknownChannel {
		channel, err = manifest.FindVersionChannel(version)
		if err != nil {
			return ToolchainRelease{}, err
		}
	}
	if version == "" {
		version, err = latestVersionForTuple(manifest, channel, tuple)
		if err != nil {
			return ToolchainRelease{}, err
		}
	}
	info, err := manifest.GetDownloadInfo(channel, version, tuple)
	if err != nil {
		return ToolchainRelease{}, err
	}
	releaseTag := info.ReleaseTag
	if channel == toolchain.Nightly && releaseTag == "" {
		releaseTag = version
	}
	return ToolchainRelease{
		Channel:    channel,
		Version:    version,
		ReleaseTag: releaseTag,
		Download:   *info,
	}, nil
}

// ResolveComponent returns the component artifact for a concrete toolchain
// release. Unified sources always consult their manifest. Legacy nightly keeps
// the historical GitCode release layout for backward compatibility.
func (s *Source) ResolveComponent(ctx context.Context, channel toolchain.Channel, version, component, platform, releaseTag string) (ComponentInfo, error) {
	if channel != toolchain.Nightly || s.unified {
		manifest, err := s.Manifest(ctx)
		if err != nil {
			return ComponentInfo{}, err
		}
		info, err := manifest.ComponentDownload(channel, version, component, platform)
		if err != nil {
			return ComponentInfo{}, err
		}
		return *info, nil
	}

	if releaseTag == "" {
		releaseTag = version
	}
	var name string
	switch component {
	case "stdx":
		if platform == "" {
			return ComponentInfo{}, fmt.Errorf("stdx requires a platform")
		}
		name = fmt.Sprintf("cangjie-stdx-%s-%s.1.zip", platform, version)
	case "docs":
		name = fmt.Sprintf("cangjie-docs-html-%s.tar.gz", version)
	case "stdx-docs":
		name = fmt.Sprintf("cangjie-stdx-docs-html-%s.1.tar.gz", version)
	default:
		return ComponentInfo{}, &cjverr.UnknownComponentError{Name: component}
	}
	base, err := url.Parse(DefaultNightlyBaseURL)
	if err != nil {
		return ComponentInfo{}, err
	}
	return ComponentInfo{Name: name, URL: base.JoinPath(releaseTag, name).String()}, nil
}

// ChannelVersions returns the channel's latest version and the versions
// available for tuple. Legacy nightly exposes only the latest GitCode release;
// unified sources expose the complete manifest history.
func (s *Source) ChannelVersions(ctx context.Context, channel toolchain.Channel, tuple string) (string, []string, error) {
	if channel == toolchain.Nightly && !s.unified {
		release, err := s.fetchLatest(ctx, DefaultNightlyAPIURL, s.gitCodeKey)
		if err != nil {
			return "", nil, err
		}
		return release.Version, []string{release.Version}, nil
	}
	manifest, err := s.Manifest(ctx)
	if err != nil {
		return "", nil, err
	}
	latest, err := manifest.GetLatestVersion(channel)
	if err != nil {
		return "", nil, err
	}
	versions, err := manifest.ListVersions(channel, tuple)
	if err != nil {
		return "", nil, err
	}
	return latest, versions, nil
}

// LatestAvailableVersion returns the newest channel version available for the
// requested tuple using metadata only.
func (s *Source) LatestAvailableVersion(ctx context.Context, channel toolchain.Channel, tuple string) (string, error) {
	if channel == toolchain.Nightly && !s.unified {
		release, err := s.fetchLatest(ctx, DefaultNightlyAPIURL, s.gitCodeKey)
		return release.Version, err
	}
	manifest, err := s.Manifest(ctx)
	if err != nil {
		return "", err
	}
	return latestVersionForTuple(manifest, channel, tuple)
}

// ChannelVersionsByTuple returns all manifest versions grouped by tuple. The
// ok result identifies sources that provide a platform catalog.
func (s *Source) ChannelVersionsByTuple(ctx context.Context, channel toolchain.Channel) (latest string, versions map[string][]string, ok bool, err error) {
	if channel == toolchain.Nightly && !s.unified {
		latest, list, fetchErr := s.ChannelVersions(ctx, channel, "")
		if fetchErr != nil {
			return "", nil, false, fetchErr
		}
		return latest, map[string][]string{"": list}, false, nil
	}
	manifest, err := s.Manifest(ctx)
	if err != nil {
		return "", nil, true, err
	}
	latest, err = manifest.GetLatestVersion(channel)
	if err != nil {
		return "", nil, true, err
	}
	versions, err = manifest.VersionsByTuple(channel)
	return latest, versions, true, err
}

func (s *Source) resolveLegacyNightly(ctx context.Context, version, tuple string) (ToolchainRelease, error) {
	release := NightlyRelease{TagName: version, Version: version}
	if version == "" {
		var err error
		release, err = s.fetchLatest(ctx, DefaultNightlyAPIURL, s.gitCodeKey)
		if err != nil {
			return ToolchainRelease{}, err
		}
	} else if s.gitCodeKey != "" {
		latest, err := s.fetchLatest(ctx, DefaultNightlyAPIURL, s.gitCodeKey)
		if err != nil {
			slog.Debug("failed to resolve pinned nightly release tag", "version", version, "error", err)
		} else if latest.Version == version || latest.TagName == version {
			release = latest
		}
	}

	assetURL, err := release.DownloadURL(DefaultNightlyBaseURL, tuple)
	if err != nil {
		return ToolchainRelease{}, err
	}
	sha256, err := s.fetchSHA256(ctx, assetURL)
	if err != nil {
		return ToolchainRelease{}, err
	}
	assetVersion := release.Version
	if assetVersion == "" {
		assetVersion = release.TagName
	}
	name, err := NightlyArchiveName(tuple, assetVersion)
	if err != nil {
		return ToolchainRelease{}, err
	}
	return ToolchainRelease{
		Channel:    toolchain.Nightly,
		Version:    assetVersion,
		ReleaseTag: release.tag(),
		Download: DownloadInfo{
			Name:   name,
			URL:    assetURL,
			SHA256: sha256,
		},
	}, nil
}

func latestVersionForTuple(manifest *Manifest, channel toolchain.Channel, tuple string) (string, error) {
	if tuple == "" {
		return manifest.GetLatestVersion(channel)
	}
	versions, err := manifest.ListVersions(channel, tuple)
	if err != nil {
		return "", err
	}
	if len(versions) > 0 {
		return versions[0], nil
	}
	latest, err := manifest.GetLatestVersion(channel)
	if err != nil {
		return "", err
	}
	return "", &cjverr.VersionNotAvailableError{Version: latest, Target: tuple}
}

func parseDistributionRoot(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid distribution server URL: %w", err)
	}
	if err := validateHTTPURL(u, "distribution server"); err != nil {
		return nil, err
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("distribution server URL requires an empty query and fragment")
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/"
	return u, nil
}

func (s *Source) resolveManifestURLs(m *Manifest) error {
	channels := []*ChannelInfo{&m.Channels.LTS, &m.Channels.STS}
	if m.Channels.Nightly != nil {
		channels = append(channels, m.Channels.Nightly)
	}
	for _, channel := range channels {
		for version, platforms := range channel.Versions {
			for platform, info := range platforms {
				resolved, err := s.resolveArtifactURL(info.URL)
				if err != nil {
					return fmt.Errorf("toolchain %s for %s: %w", version, platform, err)
				}
				info.URL = resolved
				platforms[platform] = info
			}
		}
		for version, components := range channel.Components {
			if components.Docs != nil {
				resolved, err := s.resolveArtifactURL(components.Docs.URL)
				if err != nil {
					return fmt.Errorf("docs %s: %w", version, err)
				}
				components.Docs.URL = resolved
			}
			if components.StdxDocs != nil {
				resolved, err := s.resolveArtifactURL(components.StdxDocs.URL)
				if err != nil {
					return fmt.Errorf("stdx-docs %s: %w", version, err)
				}
				components.StdxDocs.URL = resolved
			}
			for platform, info := range components.Stdx {
				resolved, err := s.resolveArtifactURL(info.URL)
				if err != nil {
					return fmt.Errorf("stdx %s for %s: %w", version, platform, err)
				}
				info.URL = resolved
				components.Stdx[platform] = info
			}
			channel.Components[version] = components
		}
	}
	return nil
}

func (s *Source) resolveArtifactURL(raw string) (string, error) {
	ref, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("invalid artifact URL %q: %w", raw, err)
	}
	if ref.IsAbs() {
		return ref.String(), nil
	}
	return s.root.ResolveReference(ref).String(), nil
}

// FetchManifest downloads and validates a toolchain manifest.
func FetchManifest(ctx context.Context, manifestURL string) (*Manifest, error) {
	u, err := url.Parse(manifestURL)
	if err != nil {
		return nil, fmt.Errorf("invalid manifest URL: %w", err)
	}
	if err := validateHTTPURL(u, "manifest"); err != nil {
		return nil, err
	}
	if u.Scheme == "http" {
		slog.Warn("fetching manifest over insecure HTTP", "url", manifestURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create manifest request: %w", err)
	}
	resp, err := HTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch manifest: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize))
	if err != nil {
		return nil, err
	}
	return ParseManifest(data)
}

func validateHTTPURL(u *url.URL, label string) error {
	if u == nil || u.Host == "" {
		return fmt.Errorf("invalid %s URL", label)
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		if isLoopbackHost(u.Hostname()) || os.Getenv(config.EnvAllowInsecureManifest) == "1" {
			return nil
		}
		return fmt.Errorf("refusing to use insecure HTTP %s %q: use HTTPS, or set %s=1 to trust an internal mirror", label, u.Host, config.EnvAllowInsecureManifest)
	default:
		return fmt.Errorf("invalid %s URL scheme %q: only https and http are supported", label, u.Scheme)
	}
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}
