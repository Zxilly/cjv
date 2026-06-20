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
// tuple is required for stdx (a host tuple selects the host stdx, a target
// tuple selects the matching cross-compile target stdx) and ignored for
// docs / stdx-docs. mf is the version manifest the LTS / STS download link is
// read from; it may be nil for nightly toolchains, whose URLs are constructed.
// force=true reinstalls over an existing manifest.
func Install(ctx context.Context, roots Roots, tc toolchain.ToolchainName, name Name, tuple, downloadsDir string, force bool, mf *dist.Manifest) (retErr error) {
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

	assetURL, err := ResolveAssetURL(spec, tc, tuple, mf)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(downloadsDir, 0o755); err != nil {
		return err
	}
	if parsed, err := url.Parse(assetURL); err != nil || parsed.Path == "" {
		return fmt.Errorf("invalid component asset URL: %s", assetURL)
	}

	fmt.Println(i18n.T("FetchingComponent", i18n.MsgData{
		"Component": string(name),
		"Toolchain": filepath.Base(roots.TcDir),
	}))
	archivePath, err := dist.DownloadCached(ctx, assetURL, "", downloadsDir)
	if err != nil {
		return err
	}
	// Drop the staged archive on success; failures keep it for the next retry.
	defer func() {
		if retErr == nil {
			_ = dist.CleanupDownload(archivePath) //nolint:errcheck // best-effort
		}
	}()

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
// the files into place, and writes the manifest. On a force reinstall over an
// existing component it snapshots first and restores on failure. It is the
// shared tail of Install and InstallFromArchive.
func stageAndInstall(ctx context.Context, roots Roots, spec Spec, name Name, archivePath string, force, alreadyInstalled bool) (retErr error) {
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

	var snap *Snapshot
	var moved []string
	// On failure: undo the move, drop the manifest, then restore the snapshot.
	// The backup is dropped (snap.Cleanup) LAST in both paths, so Restore always
	// runs before its backup is deleted. snap.Cleanup is nil-safe.
	defer func() {
		if retErr == nil {
			_ = snap.Cleanup() //nolint:errcheck // best-effort cleanup (nil-safe)
			return
		}
		_ = removePaths(roots, name, moved)         //nolint:errcheck // best-effort rollback
		_ = cleanupComponentMeta(roots.TcDir, name) //nolint:errcheck // best-effort rollback
		if snap != nil {
			_ = snap.Restore() //nolint:errcheck // best-effort rollback
			_ = snap.Cleanup() //nolint:errcheck // best-effort cleanup
		}
	}()

	if force && alreadyInstalled {
		snap, err = TakeSnapshot(roots, []Name{name})
		if err != nil {
			return err
		}
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
	err1 := os.Remove(manifestPath(tcDir, name))
	if err1 != nil && !os.IsNotExist(err1) {
		return err1
	}
	return removeFromComponentsIndex(tcDir, name)
}
