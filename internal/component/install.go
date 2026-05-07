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

	if !force && IsInstalled(roots.TcDir, name) {
		return &cjverr.ComponentAlreadyInstalledError{
			Toolchain: filepath.Base(roots.TcDir),
			Component: string(name),
		}
	}

	assetURL, err := ResolveAssetURL(spec, tc, platformKey)
	if err != nil {
		return err
	}
	checksum, err := componentChecksum(ctx, tc, assetURL, name)
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
	if err := dist.DownloadFileCached(ctx, assetURL, archivePath, checksum, downloadsDir); err != nil {
		return err
	}

	destDir := spec.InstallRoot(roots)
	if err := os.MkdirAll(filepath.Dir(destDir), 0o755); err != nil {
		return err
	}
	stageDir, err := os.MkdirTemp(filepath.Dir(destDir), ".cjv-component-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stageDir) //nolint:errcheck // best-effort cleanup

	fmt.Println(i18n.T("InstallingComponent", i18n.MsgData{"Component": string(name)}))

	paths, err := dist.ExtractFlattened(ctx, archivePath, stageDir, spec.StripTopLevel)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("component %q archive contained no files", name)
	}

	var snap *Snapshot
	var moved []string
	defer func() {
		if retErr == nil {
			return
		}
		_ = removePaths(roots, name, moved)         //nolint:errcheck // best-effort rollback
		_ = cleanupComponentMeta(roots.TcDir, name) //nolint:errcheck // best-effort rollback
		if snap != nil {
			_ = snap.Restore() //nolint:errcheck // best-effort rollback
		}
	}()

	if force && IsInstalled(roots.TcDir, name) {
		snap, err = TakeSnapshot(roots, []Name{name})
		if err != nil {
			return err
		}
		defer snap.Cleanup() //nolint:errcheck // best-effort cleanup
		if err := Remove(roots, name); err != nil {
			return fmt.Errorf("reinstall: remove existing %s: %w", name, err)
		}
	}

	moved, err = moveStagedFiles(stageDir, destDir, paths)
	if err != nil {
		return err
	}
	return WriteManifest(roots.TcDir, name, paths)
}

func componentChecksum(ctx context.Context, tc toolchain.ToolchainName, assetURL string, name Name) (string, error) {
	checksum := dist.FetchNightlySHA256(ctx, assetURL)
	if checksum == "" && tc.Channel != toolchain.Nightly {
		return "", fmt.Errorf("component %q checksum not found at %s.sha256", name, assetURL)
	}
	return checksum, nil
}

func moveStagedFiles(stageDir, destDir string, paths []string) ([]string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, err
	}
	moved := make([]string, 0, len(paths))
	for _, rel := range paths {
		src := filepath.Join(stageDir, filepath.FromSlash(rel))
		dst := filepath.Join(destDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return moved, err
		}
		if _, err := os.Lstat(dst); err == nil {
			if err := os.RemoveAll(dst); err != nil {
				return moved, err
			}
		}
		if err := os.Rename(src, dst); err != nil {
			return moved, err
		}
		moved = append(moved, rel)
	}
	return moved, nil
}

func cleanupComponentMeta(tcDir string, name Name) error {
	err1 := os.Remove(metaPath(tcDir, "manifest-"+string(name)))
	if err1 != nil && !os.IsNotExist(err1) {
		return err1
	}
	return removeFromComponentsIndex(tcDir, name)
}
