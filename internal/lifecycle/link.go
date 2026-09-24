package lifecycle

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/fsops"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// InstallToolchainFromURL downloads an SDK archive from url and materializes it
// as a custom-named toolchain that cjv OWNS (unlike a local directory `toolchain
// link`, which only references a user-owned directory). The archive is expected
// in the cangjie-build CI format: an outer .zip containing a required
// cangjie-sdk-* inner archive and an optional cangjie-stdx-* inner archive. A URL
// pointing directly at a bare SDK archive is also supported. Cross-OS installs
// are not supported. The default toolchain is never changed.
func InstallToolchainFromURL(ctx context.Context, name, url, sha256 string, force, noStdx bool, opts Options) error {
	return installLinkedToolchain(ctx, name, force, noStdx, opts, func(ctx context.Context, downloadsDir string) (string, bool, error) {
		// An optional sha256 verifies the download; otherwise we rely on the
		// transport (TLS for https — a plain http URL is the user's risk) plus the
		// archive-magic sniff in DownloadCachedWithName. The staged file is owned
		// by cjv and cleaned up on success.
		opts.report("LinkDownloadingURL", i18n.MsgData{"URL": url})
		archivePath, err := dist.DownloadCachedWithName(ctx, url, sha256, downloadsDir, name)
		return archivePath, true, err
	})
}

// InstallToolchainFromZip materializes a cjv-owned toolchain from a local archive
// file (zip or tar.gz) using the same extraction and bundled-stdx logic as the
// URL install. The archive is verified — against sha256 when non-empty, otherwise
// by its magic header — but, unlike a download, it is never moved or deleted: it
// stays where the user put it. The default toolchain is not changed.
func InstallToolchainFromZip(ctx context.Context, name, archivePath, sha256 string, force, noStdx bool, opts Options) error {
	return installLinkedToolchain(ctx, name, force, noStdx, opts, func(_ context.Context, _ string) (string, bool, error) {
		opts.report("LinkUsingArchive", i18n.MsgData{"Path": archivePath})
		if err := dist.VerifyArchive(archivePath, sha256); err != nil {
			return "", false, err
		}
		// owned=false: a user-supplied archive must never be cleaned up.
		return archivePath, false, nil
	})
}

// LinkToolchainDir references a user-owned SDK directory as the custom
// toolchain name. Nothing is copied: the toolchain entry is a symlink (or a
// junction on Windows) to dir, so a directory link has no staging tree or
// transaction of its own. It still goes through the same home recovery,
// compiler check and finalization as an installed toolchain, so the managed
// binary and proxy links are in place when it returns. A link whose
// finalization fails is removed again, leaving nothing half-configured. An
// existing toolchain of that name is never replaced.
func LinkToolchainDir(name, dir string) error {
	if _, err := sdktools.ResolveInstalledToolBinary(dir, "cjc"); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("LinkNotSDK", nil), err)
	}
	tcDir, err := config.ToolchainsDir()
	if err != nil {
		return err
	}
	if err := toolchain.RecoverHome(); err != nil {
		return err
	}
	linkPath := filepath.Join(tcDir, name)
	if _, err := os.Lstat(linkPath); err == nil {
		return &cjverr.ToolchainAlreadyInstalledError{Name: name}
	}
	if err := os.MkdirAll(tcDir, 0o755); err != nil {
		return err
	}
	if err := fsops.SymlinkOrJunction(dir, linkPath); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("LinkCreateFailed", nil), err)
	}
	if err := finalizeInstalledToolchain(); err != nil {
		_ = os.Remove(linkPath) //nolint:errcheck // best-effort rollback
		return fmt.Errorf("failed to finalize installation: %w", err)
	}
	return nil
}

// installLinkedToolchain holds the logic shared by the URL and local-archive link
// paths. fetch obtains the SDK archive (downloading it, or vetting a local file)
// and reports whether cjv owns that file and may delete it on success. The
// placement itself is the shared pipeline; what this adds is the CI bundle
// layout (outer archive, inner SDK/stdx archives, bare-archive fallback), the
// cross-OS guard, and the bundled-stdx install after the SDK is committed.
func installLinkedToolchain(ctx context.Context, name string, force, noStdx bool, opts Options, fetch func(ctx context.Context, downloadsDir string) (string, bool, error)) (retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	// The outer archive is extracted into a temp dir under downloads/ (NOT
	// toolchains/), so ExtractFlattened's own .cjv-install-* scratch dir never
	// pollutes the toolchain listing. It outlives the placement because the
	// bundled stdx archive inside it is installed after the SDK is committed.
	var outerTmp, innerStdx string
	defer func() {
		if outerTmp != "" {
			_ = os.RemoveAll(outerTmp) //nolint:errcheck // best-effort cleanup
		}
	}()
	acq := acquisition{
		fetch: fetch,
		extract: func(ctx context.Context, archivePath, stagingDir string) error {
			var err error
			outerTmp, err = os.MkdirTemp(filepath.Dir(archivePath), ".cjv-link-*")
			if err != nil {
				return err
			}
			if _, err := dist.ExtractFlattened(ctx, archivePath, outerTmp, false); err != nil {
				return err
			}
			innerSDK, stdx, bareSDKDir, err := locateInnerArchives(outerTmp)
			if err != nil {
				return err
			}
			innerStdx = stdx
			switch {
			case innerSDK != "":
				if err := dist.InstallSDK(ctx, innerSDK, stagingDir); err != nil {
					return err
				}
			case bareSDKDir != "":
				// The bare archive's top-level dir is already extracted in outerTmp; move
				// it into staging (a cheap rename on the shared CJV_HOME volume) rather
				// than extracting again. Across volumes the rename fails, so fall back to
				// moving the already-extracted tree entry-by-entry (copy) — not a second
				// full decompression of the archive.
				if err := fsops.RenameRetry(bareSDKDir, stagingDir); err != nil {
					if _, err := fsops.MoveTree(bareSDKDir, stagingDir); err != nil {
						return fmt.Errorf("failed to stage SDK: %w", err)
					}
				}
			default:
				return errors.New(i18n.T("LinkNoSDKArchive", nil))
			}
			// Cross-OS guard. Read the target OS from the staged cjc executable's magic
			// (ELF/Mach-O/PE) rather than the archive filename: it is authoritative and
			// works for both the nested-archive and bare-archive paths.
			if archOS := sdkBinaryOS(stagingDir); archOS != "" && archOS != runtime.GOOS {
				return errors.New(i18n.T("LinkCrossOSUnsupported", i18n.MsgData{
					"Target": archOS,
					"Host":   runtime.GOOS,
				}))
			}
			return nil
		},
	}
	// tuple is always "": a linked SDK is validated against the host OS only
	// and never sets the default toolchain.
	if err := placeToolchain(ctx, name, force, "", acq, nil, opts); err != nil {
		return err
	}

	// Install bundled stdx as a component of this toolchain, if present. The SDK
	// is already committed at this point; if stdx fails we keep the working SDK
	// (matching `install -c stdx` half-failure semantics) but surface recovery
	// guidance, since a plain retry would hit ToolchainAlreadyInstalledError.
	if innerStdx == "" || noStdx {
		return nil
	}
	opts.report("LinkInstallingStdx", i18n.MsgData{"Name": name})
	roots, err := component.RootsFor(name)
	if err != nil {
		return err
	}
	guidance := i18n.T("LinkStdxFailedAfterSDK", i18n.MsgData{"Name": name})
	if err := component.InstallFromArchive(ctx, roots, component.Stdx, innerStdx, force); err != nil {
		return fmt.Errorf("%s: %w", guidance, err)
	}
	for _, sub := range []string{"dynamic", "static"} {
		if _, err := os.Stat(filepath.Join(roots.StdxDir, sub)); err != nil {
			// Roll back the half-written stdx (wrong-layout tree + manifest +
			// index entry) so it does not falsely report as installed.
			_ = component.Remove(roots, component.Stdx) //nolint:errcheck // best-effort rollback
			return fmt.Errorf("%s: %s", i18n.T("LinkStdxMissingDirs", nil), guidance)
		}
	}
	return nil
}

// locateInnerArchives scans the extracted outer archive's top level. It returns
// the inner SDK archive path (required), the inner stdx archive path (optional),
// and, when there is no inner SDK archive but the top level is exactly one
// directory, the path to that directory (the direct bare-SDK fallback).
func locateInnerArchives(outerTmp string) (innerSDK, innerStdx, bareSDKDir string, err error) {
	entries, err := os.ReadDir(outerTmp)
	if err != nil {
		return "", "", "", err
	}
	var dirs []string
	for _, e := range entries {
		nm := e.Name()
		full := filepath.Join(outerTmp, nm)
		if e.IsDir() {
			// Skip archive-tool metadata (macOS Finder's __MACOSX) and hidden
			// dirs so a repackaged bare archive still resolves to its one SDK dir.
			if nm == "__MACOSX" || strings.HasPrefix(nm, ".") {
				continue
			}
			dirs = append(dirs, full)
			continue
		}
		switch {
		case isArchiveName(nm, "cangjie-sdk-"):
			innerSDK = full
		case isArchiveName(nm, "cangjie-stdx-"):
			innerStdx = full
		}
	}
	if innerSDK == "" && len(dirs) == 1 {
		bareSDKDir = dirs[0]
	}
	return innerSDK, innerStdx, bareSDKDir, nil
}

func isArchiveName(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".zip") ||
		strings.HasSuffix(lower, ".tar.gz") ||
		strings.HasSuffix(lower, ".tgz")
}

// sdkBinaryOS returns the GOOS that the SDK's cjc executable targets, read from
// the file's magic bytes, or "" when no cjc binary is present or its format is
// unrecognized.
func sdkBinaryOS(sdkDir string) string {
	for _, name := range []string{"cjc", "cjc.exe"} {
		f, err := os.Open(filepath.Join(sdkDir, "bin", name))
		if err != nil {
			continue
		}
		var hdr [4]byte
		n, _ := io.ReadFull(f, hdr[:])
		_ = f.Close() //nolint:errcheck // read-only
		if goos := osFromMagic(hdr[:n]); goos != "" {
			return goos
		}
	}
	return ""
}

// osFromMagic maps an executable's leading bytes to a GOOS: ELF -> linux,
// Mach-O -> darwin, PE (MZ) -> windows. The Mach-O magics cover 32/64-bit in
// both byte orders plus the universal (fat) header. See gore's file.go.
func osFromMagic(b []byte) string {
	if len(b) >= 4 && b[0] == 0x7f && b[1] == 'E' && b[2] == 'L' && b[3] == 'F' {
		return "linux"
	}
	if len(b) >= 2 && b[0] == 'M' && b[1] == 'Z' {
		return "windows"
	}
	if len(b) >= 4 {
		switch binary.BigEndian.Uint32(b) {
		case 0xFEEDFACE, 0xFEEDFACF, 0xCEFAEDFE, 0xCFFAEDFE, 0xCAFEBABE, 0xBEBAFECA:
			return "darwin"
		}
	}
	return ""
}
