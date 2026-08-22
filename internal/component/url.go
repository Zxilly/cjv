package component

import (
	"fmt"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/dist"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// ResolveAssetURL resolves the download URL for a component archive.
//
// Every standard channel reads component URLs from the version manifest.
func ResolveAssetURL(spec Spec, tc toolchain.ToolchainName, tuple string, mf *dist.Manifest) (string, error) {
	info, err := ResolveAssetInfo(spec, tc, tuple, mf)
	if err != nil {
		return "", err
	}
	return info.URL, nil
}

// ResolveAssetInfo resolves the complete component artifact descriptor.
func ResolveAssetInfo(spec Spec, tc toolchain.ToolchainName, tuple string, mf *dist.Manifest) (dist.ComponentInfo, error) {
	if !spec.SupportsChannel(tc.Channel) {
		return dist.ComponentInfo{}, &cjverr.ComponentNotAvailableForChannelError{
			Component: string(spec.Name),
			Channel:   tc.Channel.String(),
		}
	}
	if tc.Version == "" {
		return dist.ComponentInfo{}, fmt.Errorf("component %q requires a resolved toolchain version", spec.Name)
	}
	return manifestComponentURL(mf, spec.Name, tc, tuple)
}

// manifestComponentURL looks up the component link in the manifest.
// For stdx it keys on the archive platform token derived from tuple; docs /
// stdx-docs have a single archive per version.
func manifestComponentURL(mf *dist.Manifest, name Name, tc toolchain.ToolchainName, tuple string) (dist.ComponentInfo, error) {
	if mf == nil {
		return dist.ComponentInfo{}, fmt.Errorf("component %q requires the version manifest", name)
	}
	platform := ""
	if name == Stdx {
		p, err := stdxPlatform(tuple)
		if err != nil {
			return dist.ComponentInfo{}, err
		}
		platform = p
	}
	info, err := mf.ComponentDownload(tc.Channel, tc.Version, string(name), platform)
	if err != nil {
		return dist.ComponentInfo{}, err
	}
	return *info, nil
}

func stdxPlatform(tuple string) (string, error) {
	if tuple == "" {
		return "", fmt.Errorf("stdx requires a host tuple")
	}
	return sdktarget.StdxPlatformForTuple(tuple)
}
