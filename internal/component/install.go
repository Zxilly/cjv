package component

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// Install downloads and unpacks a component for the given toolchain.
// platformKey is required for stdx (host key, no target suffix) and ignored
// for docs / stdx-docs. force=true reinstalls over an existing manifest.
func Install(ctx context.Context, roots Roots, tc toolchain.ToolchainName, name Name, platformKey, downloadsDir string, force bool) (retErr error) {
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

	if force {
		if err := Remove(roots, name); err != nil {
			return fmt.Errorf("reinstall: remove existing %s: %w", name, err)
		}
	} else if IsInstalled(roots.TcDir, name) {
		return &cjverr.ComponentAlreadyInstalledError{
			Toolchain: filepath.Base(roots.TcDir),
			Component: string(name),
		}
	}

	assetURL, err := ResolveAssetURL(spec, tc, platformKey)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(downloadsDir, 0o755); err != nil {
		return err
	}
	parsed, err := url.Parse(assetURL)
	if err != nil || parsed.Path == "" {
		return fmt.Errorf("invalid component asset URL: %s", assetURL)
	}
	archivePath := filepath.Join(downloadsDir, filepath.Base(parsed.Path))

	fmt.Println(i18n.T("FetchingComponent", i18n.MsgData{
		"Component": string(name),
		"Toolchain": filepath.Base(roots.TcDir),
	}))
	if err := dist.DownloadFileCached(ctx, assetURL, archivePath, "", downloadsDir); err != nil {
		return err
	}

	destDir := spec.InstallRoot(roots)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	fmt.Println(i18n.T("InstallingComponent", i18n.MsgData{"Component": string(name)}))

	paths, err := dist.ExtractFlattened(ctx, archivePath, destDir, spec.StripTopLevel)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("component %q archive contained no files", name)
	}

	defer func() {
		if retErr == nil {
			return
		}
		_ = removePaths(roots, name, paths) //nolint:errcheck // best-effort rollback
	}()

	return WriteManifest(roots.TcDir, name, paths)
}
