package component

import (
	"fmt"
	"net/url"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/dist"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// ResolveAssetURL resolves the download URL for a component archive.
//
// Nightly builds construct the URL from the on-disk release metadata: the
// nightly_build repo publishes every component under a single release tag, so
// the URL is derivable. LTS / STS read the link straight from the version
// manifest instead — the upstream cangjie_stdx repo tags releases
// inconsistently (the release tag is sometimes the SDK version, sometimes the
// stdx asset version), so the manifest carries verbatim links rather than a
// reconstruction. mf is required for LTS / STS and ignored for nightly.
func ResolveAssetURL(spec Spec, tc toolchain.ToolchainName, tuple string, mf *dist.Manifest) (string, error) {
	if !spec.SupportsChannel(tc.Channel) {
		return "", &cjverr.ComponentNotAvailableForChannelError{
			Component: string(spec.Name),
			Channel:   tc.Channel.String(),
		}
	}
	if tc.Version == "" {
		return "", fmt.Errorf("component %q requires a resolved toolchain version", spec.Name)
	}
	if tc.Channel == toolchain.Nightly {
		return nightlyComponentURL(spec.Name, tc, tuple)
	}
	return manifestComponentURL(mf, spec.Name, tc, tuple)
}

// manifestComponentURL looks up the LTS / STS component link in the manifest.
// For stdx it keys on the archive platform token derived from tuple; docs /
// stdx-docs have a single archive per version.
func manifestComponentURL(mf *dist.Manifest, name Name, tc toolchain.ToolchainName, tuple string) (string, error) {
	if mf == nil {
		return "", fmt.Errorf("component %q requires the version manifest", name)
	}
	platform := ""
	if name == Stdx {
		p, err := stdxPlatform(tuple)
		if err != nil {
			return "", err
		}
		platform = p
	}
	info, err := mf.ComponentDownload(tc.Channel, tc.Version, string(name), platform)
	if err != nil {
		return "", err
	}
	return info.URL, nil
}

// nightlyComponentURL constructs the component URL for a nightly toolchain from
// the release metadata recorded at install time. The stdx asset carries the
// extra `.1` stdx revision suffix; docs does not.
func nightlyComponentURL(name Name, tc toolchain.ToolchainName, tuple string) (string, error) {
	nightly := nightlyReleaseAsset(tc)
	var asset string
	switch name {
	case Stdx:
		platform, err := stdxPlatform(tuple)
		if err != nil {
			return "", err
		}
		asset = fmt.Sprintf("cangjie-stdx-%s-%s.1.zip", platform, nightly.Version)
	case Docs:
		asset = fmt.Sprintf("cangjie-docs-html-%s.tar.gz", nightly.Version)
	case StdxDocs:
		asset = fmt.Sprintf("cangjie-stdx-docs-html-%s.1.tar.gz", nightly.Version)
	default:
		return "", &cjverr.UnknownComponentError{Name: string(name)}
	}
	return joinReleaseURL(dist.DefaultNightlyBaseURL, nightly.ReleaseTag, asset)
}

func stdxPlatform(tuple string) (string, error) {
	if tuple == "" {
		return "", fmt.Errorf("stdx requires a host tuple")
	}
	return sdktarget.StdxPlatformForTuple(tuple)
}

type nightlyAsset struct {
	ReleaseTag string
	Version    string
}

func nightlyReleaseAsset(tc toolchain.ToolchainName) nightlyAsset {
	asset := nightlyAsset{ReleaseTag: tc.Version, Version: tc.Version}
	if tc.Channel != toolchain.Nightly || tc.Version == "" {
		return asset
	}
	roots, err := RootsFor(tc.String())
	if err != nil {
		return asset
	}
	meta, err := toolchain.ReadNightlyReleaseMetadata(roots.TcDir)
	if err != nil {
		return asset
	}
	if meta.ReleaseTag != "" {
		asset.ReleaseTag = meta.ReleaseTag
	}
	if meta.Version != "" {
		asset.Version = meta.Version
	}
	return asset
}

func joinReleaseURL(base, tag, asset string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid base URL %q: %w", base, err)
	}
	return u.JoinPath(tag, asset).String(), nil
}
