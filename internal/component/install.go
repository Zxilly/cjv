package component

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/fsops"
	"github.com/Zxilly/cjv/internal/progress"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// InstallFromSource downloads and unpacks a component for the given toolchain
// through the configured distribution source. tuple is required for stdx (a
// host tuple selects the host stdx, a target tuple selects the matching
// cross-compile target stdx) and ignored for docs / stdx-docs. force=true
// reinstalls over an existing manifest. sink receives the FetchingComponent
// and InstallingComponent stages and the download progress; nil reports
// nothing.
func InstallFromSource(ctx context.Context, roots Roots, tc toolchain.ToolchainName, name Name, tuple, downloadsDir string, force bool, source *dist.Source, sink progress.Sink) (retErr error) {
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
	}, progress.Or(sink))
}

func installWithResolver(ctx context.Context, roots Roots, tc toolchain.ToolchainName, name Name, tuple, downloadsDir string, force bool, resolve func(Spec) (dist.ComponentInfo, error), sink progress.Sink) (retErr error) {
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

	sink.Report(progress.Event{Kind: progress.FetchingComponent, Toolchain: tc.String(), Component: string(name)})
	archivePath, err := dist.DownloadCached(ctx, asset.URL, asset.SHA256, downloadsDir, sink)
	if err != nil {
		return err
	}
	// Drop the staged archive on success; failures keep it for the next retry.
	defer func() {
		if retErr == nil {
			_ = dist.CleanupDownload(archivePath) //nolint:errcheck // best-effort
		}
	}()

	sink.Report(progress.Event{Kind: progress.InstallingComponent, Toolchain: tc.String(), Component: string(name)})

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
		_, err := fsops.MoveTree(stageDir, destDir)
		return err
	})
}

// stdxPlatform maps the SDK tuple to the stdx archive platform token the
// manifest is keyed by (e.g. "linux-arm64" -> "linux-aarch64",
// "linux-x64-ohos" -> "ohos-aarch64").
func stdxPlatform(tuple string) (string, error) {
	if tuple == "" {
		return "", fmt.Errorf("stdx requires a host tuple")
	}
	return sdktarget.StdxPlatformForTuple(tuple)
}
