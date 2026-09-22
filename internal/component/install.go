package component

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// Install downloads and unpacks a component for the given toolchain.
// tuple is required for stdx (a host tuple selects the host stdx, a target
// tuple selects the matching cross-compile target stdx) and ignored for
// docs / stdx-docs. mf supplies component URLs for every standard channel.
// force=true reinstalls over an existing manifest.
func Install(ctx context.Context, roots Roots, tc toolchain.ToolchainName, name Name, tuple, downloadsDir string, force bool, mf *dist.Manifest) (retErr error) {
	return installWithResolver(ctx, roots, tc, name, tuple, downloadsDir, force, func(spec Spec) (dist.ComponentInfo, error) {
		return ResolveAssetInfo(spec, tc, tuple, mf)
	}, nil)
}

// InstallFromSource installs a component through the configured manifest
// distribution source.
func InstallFromSource(ctx context.Context, roots Roots, tc toolchain.ToolchainName, name Name, tuple, downloadsDir string, force bool, source *dist.Source, report func(string)) (retErr error) {
	return installWithResolver(ctx, roots, tc, name, tuple, downloadsDir, force, func(spec Spec) (dist.ComponentInfo, error) {
		platform := ""
		if name == Stdx {
			var err error
			platform, err = stdxPlatform(tuple)
			if err != nil {
				return dist.ComponentInfo{}, err
			}
		}
		return source.ResolveComponent(ctx, tc.Channel, tc.Version, string(name), platform)
	}, report)
}

func installWithResolver(ctx context.Context, roots Roots, tc toolchain.ToolchainName, name Name, tuple, downloadsDir string, force bool, resolve func(Spec) (dist.ComponentInfo, error), report func(string)) (retErr error) {
	spec, err := SpecFor(name)
	if err != nil {
		return err
	}
	if !spec.SupportsChannel(tc.Channel) {
		return &cjverr.ComponentNotAvailableForChannelError{
			Component: string(spec.Name),
			Channel:   tc.Channel.String(),
		}
	}

	alreadyInstalled := IsInstalled(roots.TcDir, name)
	if !force && alreadyInstalled {
		return &cjverr.ComponentAlreadyInstalledError{
			Toolchain: filepath.Base(roots.TcDir),
			Component: string(name),
		}
	}

	asset, err := resolve(spec)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(downloadsDir, 0o755); err != nil {
		return err
	}
	if parsed, err := url.Parse(asset.URL); err != nil || parsed.Path == "" {
		return fmt.Errorf("invalid component asset URL: %s", asset.URL)
	}

	if report != nil {
		report("FetchingComponent")
	}
	archivePath, err := dist.DownloadCached(ctx, asset.URL, asset.SHA256, downloadsDir)
	if err != nil {
		return err
	}
	// Drop the staged archive on success; failures keep it for the next retry.
	defer func() {
		if retErr == nil {
			_ = dist.CleanupDownload(archivePath) //nolint:errcheck // best-effort
		}
	}()

	if report != nil {
		report("InstallingComponent")
	}

	return stageAndInstall(ctx, roots, spec, name, archivePath, force, alreadyInstalled)
}

// InstallFromArchive installs a component from a local archive file already on
// disk, bypassing URL/channel resolution and version checks. It is used by URL
// toolchain install to materialize a component bundled inside the downloaded
// artifact (e.g. the stdx archive shipped alongside the SDK).
func InstallFromArchive(ctx context.Context, roots Roots, name Name, archivePath string, force bool) error {
	spec, err := SpecFor(name)
	if err != nil {
		return err
	}

	alreadyInstalled := IsInstalled(roots.TcDir, name)
	if !force && alreadyInstalled {
		return &cjverr.ComponentAlreadyInstalledError{
			Toolchain: filepath.Base(roots.TcDir),
			Component: string(name),
		}
	}

	return stageAndInstall(ctx, roots, spec, name, archivePath, force, alreadyInstalled)
}

// stageAndInstall extracts archivePath into the component's install root, moves
// the files into place, and writes the manifest through the same replacement
// operation as local linking. It is the shared tail of Install and InstallFromArchive.
func stageAndInstall(ctx context.Context, roots Roots, spec Spec, name Name, archivePath string, force, alreadyInstalled bool) error {
	destDir := spec.InstallRoot(roots)
	if err := os.MkdirAll(filepath.Dir(destDir), 0o755); err != nil {
		return err
	}
	stageDir, err := os.MkdirTemp(filepath.Dir(destDir), ".cjv-component-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stageDir) //nolint:errcheck // best-effort cleanup

	paths, err := dist.ExtractFlattened(ctx, archivePath, stageDir, spec.StripTopLevel)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("component %q archive contained no files", name)
	}

	return replaceComponent(roots, name, force && alreadyInstalled, paths, func() error {
		return moveStagedFiles(stageDir, destDir, paths)
	})
}

func moveStagedFiles(stageDir, destDir string, paths []string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	for _, rel := range paths {
		src := filepath.Join(stageDir, filepath.FromSlash(rel))
		dst := filepath.Join(destDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if _, err := os.Lstat(dst); err == nil {
			if err := os.RemoveAll(dst); err != nil {
				return err
			}
		}
		if err := os.Rename(src, dst); err != nil {
			return err
		}
	}
	return nil
}
