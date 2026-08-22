package dist

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	pathpkg "path"
	"strings"
	"sync"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/toolchain"
)

const (
	distributionManifestName = "versions.json"
	nightlyManifestName      = "nightly.json"
)

// Source is the configured manifest-backed toolchain distribution source.
type Source struct {
	manifestURL string
	nightlyURL  string
	root        *url.URL
	fetchSHA256 func(context.Context, string) (string, error)

	once     sync.Once
	manifest *Manifest
	err      error

	nightlyOnce     sync.Once
	nightlyManifest *Manifest
	nightlyErr      error
}

// SourceOptions supplies the external checksum-sidecar operation used by
// nightly SDK entries whose manifest checksum is empty.
type SourceOptions struct {
	FetchNightlySHA256 func(context.Context, string) (string, error)
}

// ManifestFetchError reports an HTTP response from a manifest endpoint.
type ManifestFetchError struct {
	URL        string
	StatusCode int
}

func (e *ManifestFetchError) Error() string {
	return fmt.Sprintf("failed to fetch manifest: HTTP %d", e.StatusCode)
}

// NewSource resolves the active distribution source from settings.
func NewSource(settings *config.Settings) (*Source, error) {
	return NewSourceWithOptions(settings, SourceOptions{})
}

// NewSourceWithOptions resolves the source with an explicit nightly checksum
// operation. Lifecycle uses this as a test seam.
func NewSourceWithOptions(settings *config.Settings, opts SourceOptions) (*Source, error) {
	if settings == nil {
		return nil, fmt.Errorf("distribution source requires settings")
	}
	if opts.FetchNightlySHA256 == nil {
		opts.FetchNightlySHA256 = FetchNightlySHA256
	}
	rootValue := settings.ResolveDistServer()
	if rootValue == "" {
		root, err := manifestDirectory(settings.ManifestURL)
		if err != nil {
			return nil, err
		}
		return &Source{
			manifestURL: settings.ManifestURL,
			nightlyURL:  root.JoinPath(nightlyManifestName).String(),
			root:        root,
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
		nightlyURL:  root.JoinPath(nightlyManifestName).String(),
		root:        root,
		fetchSHA256: opts.FetchNightlySHA256,
	}, nil
}

// NightlyManifestURL returns the effective nightly channel endpoint.
func (s *Source) NightlyManifestURL() string {
	if s == nil {
		return ""
	}
	return s.nightlyURL
}

// ToolchainRelease is the source-level result for one concrete toolchain
// build. Download is ready for the shared downloader.
type ToolchainRelease struct {
	Channel  toolchain.Channel
	Version  string
	Download DownloadInfo
}

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
		if s.err == nil {
			s.err = s.resolveManifestURLs(s.manifest)
		}
	})
	return s.manifest, s.err
}

func (s *Source) manifestForChannel(ctx context.Context, channel toolchain.Channel) (*Manifest, error) {
	if channel != toolchain.Nightly {
		return s.Manifest(ctx)
	}
	s.nightlyOnce.Do(func() {
		var channelInfo *ChannelInfo
		channelInfo, s.nightlyErr = FetchChannelManifest(ctx, s.nightlyURL, toolchain.Nightly)
		if s.nightlyErr != nil {
			var fetchErr *ManifestFetchError
			if errors.As(s.nightlyErr, &fetchErr) && fetchErr.StatusCode == http.StatusNotFound {
				s.nightlyManifest, s.nightlyErr = s.Manifest(ctx)
			}
			return
		}
		s.nightlyManifest = &Manifest{}
		s.nightlyManifest.Channels.Nightly = channelInfo
		s.nightlyErr = s.resolveManifestURLs(s.nightlyManifest)
	})
	return s.nightlyManifest, s.nightlyErr
}

// ResolveToolchain selects one concrete toolchain artifact from the manifest.
func (s *Source) ResolveToolchain(ctx context.Context, channel toolchain.Channel, version, tuple string) (ToolchainRelease, error) {
	manifest, err := s.manifestForChannel(ctx, channel)
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
	download := *info
	if channel == toolchain.Nightly && download.SHA256 == "" {
		download.SHA256, err = s.fetchSHA256(ctx, download.URL)
		if err != nil {
			return ToolchainRelease{}, err
		}
	}
	return ToolchainRelease{
		Channel:  channel,
		Version:  version,
		Download: download,
	}, nil
}

// ResolveComponent returns the manifest component artifact for a concrete
// toolchain release.
func (s *Source) ResolveComponent(ctx context.Context, channel toolchain.Channel, version, component, platform string) (ComponentInfo, error) {
	manifest, err := s.manifestForChannel(ctx, channel)
	if err != nil {
		return ComponentInfo{}, err
	}
	info, err := manifest.ComponentDownload(channel, version, component, platform)
	if err != nil {
		return ComponentInfo{}, err
	}
	return *info, nil
}

// ChannelVersions returns the channel's latest version and the versions
// available for tuple.
func (s *Source) ChannelVersions(ctx context.Context, channel toolchain.Channel, tuple string) (string, []string, error) {
	manifest, err := s.manifestForChannel(ctx, channel)
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
	manifest, err := s.manifestForChannel(ctx, channel)
	if err != nil {
		return "", err
	}
	return latestVersionForTuple(manifest, channel, tuple)
}

// ChannelVersionsByTuple returns all manifest versions grouped by tuple.
func (s *Source) ChannelVersionsByTuple(ctx context.Context, channel toolchain.Channel) (latest string, versions map[string][]string, err error) {
	manifest, err := s.manifestForChannel(ctx, channel)
	if err != nil {
		return "", nil, err
	}
	latest, err = manifest.GetLatestVersion(channel)
	if err != nil {
		return "", nil, err
	}
	versions, err = manifest.VersionsByTuple(channel)
	return latest, versions, err
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

func manifestDirectory(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid manifest URL: %w", err)
	}
	u.Path = strings.TrimSuffix(pathpkg.Dir(u.Path), "/") + "/"
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = ""
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
		return nil, &ManifestFetchError{URL: manifestURL, StatusCode: resp.StatusCode}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize))
	if err != nil {
		return nil, err
	}
	return ParseManifest(data)
}

// FetchChannelManifest downloads and validates a single-channel manifest.
func FetchChannelManifest(ctx context.Context, manifestURL string, channel toolchain.Channel) (*ChannelInfo, error) {
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
		return nil, &ManifestFetchError{URL: manifestURL, StatusCode: resp.StatusCode}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize))
	if err != nil {
		return nil, err
	}
	channelInfo, channelErr := ParseChannelManifest(data, channel)
	if channelErr == nil {
		return channelInfo, nil
	}
	aggregated, aggregatedErr := ParseManifest(data)
	if aggregatedErr == nil {
		return aggregated.getChannel(channel)
	}
	return nil, channelErr
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
