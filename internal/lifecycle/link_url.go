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
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/Zxilly/cjv/internal/utils"
)

// InstallToolchainFromURL downloads an SDK archive from url and materializes it
// as a custom-named toolchain that cjv OWNS (unlike the local `toolchain link`,
// which only references a user-owned directory). The archive is expected in the
// cangjie-build CI format: an outer .zip containing a required cangjie-sdk-*
// inner archive and an optional cangjie-stdx-* inner archive. A URL pointing
// directly at a bare SDK archive is also supported. Cross-OS installs are not
// supported. The default toolchain is never changed.
func InstallToolchainFromURL(ctx context.Context, name, url, sha256 string, force, noStdx bool, opts Options) (retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := config.EnsureDirs(); err != nil {
		return err
	}
	downloadsDir, err := config.DownloadsDir()
	if err != nil {
		return err
	}
	tcDir, err := config.ToolchainsDir()
	if err != nil {
		return err
	}
	destDir := filepath.Join(tcDir, name)
	isReinstall := false
	if _, err := os.Stat(destDir); err == nil {
		if !force {
			return &cjverr.ToolchainAlreadyInstalledError{Name: name}
		}
		isReinstall = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to stat %s: %w", destDir, err)
	}

	// An optional sha256 verifies the download; otherwise we rely on the transport
	// (TLS for https — a plain http URL is the user's risk) plus the archive-magic
	// sniff in DownloadCachedWithName.
	opts.note(i18n.T("LinkDownloadingURL", i18n.MsgData{"URL": url}))
	archivePath, err := dist.DownloadCachedWithName(ctx, url, sha256, downloadsDir, name)
	if err != nil {
		return err
	}
	defer func() {
		if retErr == nil {
			_ = dist.CleanupDownload(archivePath) //nolint:errcheck // best-effort
		}
	}()

	// Extract the outer archive into a temp dir under downloads/ (NOT toolchains/),
	// so ExtractFlattened's own .cjv-install-* scratch dir never pollutes the
	// toolchain listing.
	outerTmp, err := os.MkdirTemp(downloadsDir, ".cjv-link-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(outerTmp) //nolint:errcheck // best-effort cleanup
	if _, err := dist.ExtractFlattened(ctx, archivePath, outerTmp, false); err != nil {
		return err
	}

	innerSDK, innerStdx, bareSDKDir, err := locateInnerArchives(outerTmp)
	if err != nil {
		return err
	}

	stagingDir := destDir + toolchain.StagingSuffix
	if err := utils.RemoveAllRetry(stagingDir); err != nil {
		return fmt.Errorf("failed to clean staging directory: %w", err)
	}
	defer func() {
		if retErr != nil {
			_ = utils.RemoveAllRetry(stagingDir) //nolint:errcheck // best-effort
		}
	}()

	opts.note(i18n.T("Extracting", nil))
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
		if err := utils.RenameRetry(bareSDKDir, stagingDir); err != nil {
			if err := dist.MoveTreeContents(bareSDKDir, stagingDir); err != nil {
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

	// tuple is always "" — URL install validates against the host OS only and
	// does not support cross-OS SDKs.
	if err := opts.validateInstallation(stagingDir, ""); err != nil {
		return err
	}

	// Transactional swap into place. afterSwap ensures the managed cjv binary
	// and proxy links exist; the default toolchain is deliberately NOT changed.
	if err := swapInstalledToolchain(stagingDir, destDir, isReinstall, func() error {
		if err := opts.ensureManagedBinary(); err != nil {
			return err
		}
		return opts.createProxyLinks()
	}); err != nil {
		return err
	}

	// The SDK swap replaced toolchains/<name> (and with it the component manifest
	// under .cjv/components) but left the sibling stdx/<name> tree from the prior
	// install intact. On a --force re-link, clear it so a changed stdx bundle does
	// not leave orphaned stale libraries behind (and so the component would not be
	// detected as already-installed against a now-missing manifest).
	if isReinstall {
		if stdxDir, err := config.StdxDirFor(name); err == nil {
			_ = utils.RemoveAllRetry(stdxDir) //nolint:errcheck // best-effort
		}
	}

	// Install bundled stdx as a component of this toolchain, if present. The SDK
	// is already committed at this point; if stdx fails we keep the working SDK
	// (matching `install -c stdx` half-failure semantics) but surface recovery
	// guidance, since a plain retry would hit ToolchainAlreadyInstalledError.
	if innerStdx != "" && !noStdx {
		opts.note(i18n.T("LinkInstallingStdx", i18n.MsgData{"Name": name}))
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
	}

	opts.green("ToolchainInstalled", i18n.MsgData{"Name": name})
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
