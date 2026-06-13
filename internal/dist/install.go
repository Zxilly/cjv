package dist

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/codeclysm/extract/v4"

	"github.com/Zxilly/cjv/internal/utils"
)

func InstallSDK(ctx context.Context, archivePath, destDir string) error {
	_, err := ExtractFlattened(ctx, archivePath, destDir, true)
	return err
}

// ExtractFlattened unpacks archivePath into destDir, returning the relative
// forward-slash paths of every file and symlink created (directories are
// not recorded). When stripTopLevel is true and the archive has a single
// top-level directory, that directory is unwrapped (matches how SDK and
// stdx archives are shipped); when false the archive is merged verbatim
// (used for docs archives, which are already flat).
func ExtractFlattened(ctx context.Context, archivePath, destDir string, stripTopLevel bool) ([]string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create install directory: %w", err)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close() //nolint:errcheck // read-only

	tmpDir, err := os.MkdirTemp(filepath.Dir(destDir), ".cjv-install-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck // best-effort cleanup

	if err := extract.Archive(ctx, f, tmpDir, nil); err != nil {
		return nil, fmt.Errorf("failed to extract archive: %w", err)
	}

	srcDir := tmpDir
	if stripTopLevel {
		entries, err := os.ReadDir(tmpDir)
		if err != nil {
			return nil, err
		}
		if len(entries) == 1 && entries[0].IsDir() {
			srcDir = filepath.Join(tmpDir, entries[0].Name())
		}
	}

	return moveContentsRecording(srcDir, destDir)
}

// MoveTreeContents moves every entry under srcDir into destDir, preferring a
// rename per entry and falling back to a cross-filesystem copy. It stages an
// already-extracted tree without re-reading the source archive.
func MoveTreeContents(srcDir, destDir string) error {
	_, err := moveContentsRecording(srcDir, destDir)
	return err
}

// moveContentsRecording overwrites existing entries — component archives
// may legitimately replace SDK static assets.
func moveContentsRecording(srcDir, destDir string) ([]string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, err
	}

	var paths []string
	if err := walkAndMove(srcDir, destDir, "", &paths); err != nil {
		return nil, err
	}
	return paths, nil
}

func walkAndMove(srcRoot, destRoot, relDir string, paths *[]string) error {
	srcDir := filepath.Join(srcRoot, relDir)
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		relPath := entry.Name()
		if relDir != "" {
			relPath = relDir + string(filepath.Separator) + entry.Name()
		}
		src := filepath.Join(srcRoot, relPath)
		dst := filepath.Join(destRoot, relPath)

		info, err := os.Lstat(src)
		if err != nil {
			return err
		}

		if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return err
			}
			if err := walkAndMove(srcRoot, destRoot, relPath, paths); err != nil {
				return err
			}
			continue
		}

		if err := placeEntry(srcRoot, src, dst); err != nil {
			return err
		}
		*paths = append(*paths, filepath.ToSlash(relPath))
	}
	return nil
}

// placeEntry prefers rename, falling back to copy on cross-filesystem moves.
// The destination is removed first so reinstalls overwrite cleanly.
func placeEntry(srcRoot, src, dst string) error {
	if err := validateSymlinkTarget(srcRoot, src); err != nil {
		return err
	}
	if _, err := os.Lstat(dst); err == nil {
		if err := utils.RemoveAllRetry(dst); err != nil {
			return fmt.Errorf("failed to overwrite %s: %w", dst, err)
		}
	}
	if err := utils.RenameRetry(src, dst); err == nil {
		return nil
	}
	return copyEntry(srcRoot, src, dst)
}

func copyEntry(srcRoot, src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		if err := validateSymlinkTarget(srcRoot, src); err != nil {
			return err
		}
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if targetInfo, err := os.Stat(src); err == nil && targetInfo.IsDir() {
			return utils.SymlinkOrJunction(target, dst)
		}
		if err := os.Symlink(target, dst); err == nil {
			return nil
		} else {
			// Creating a symlink can fail without privileges (e.g. Windows
			// without Developer Mode). The target was validated to stay inside
			// the install root above, so materialize it by copying the resolved
			// file instead of aborting the whole install. A dangling symlink
			// has nothing to copy, so surface the original error.
			resolved, statErr := os.Stat(src)
			if statErr != nil {
				return err
			}
			if resolved.IsDir() {
				return utils.SymlinkOrJunction(target, dst)
			}
			return utils.CopyFile(src, dst, resolved.Mode().Perm())
		}
	}

	if info.IsDir() {
		return copyDir(srcRoot, src, dst)
	}
	return utils.CopyFile(src, dst, info.Mode())
}

// validateSymlinkTarget rejects symlinks whose resolved target escapes srcRoot.
// Parent-traversing targets are allowed as long as the resolved path stays
// inside srcRoot.
func validateSymlinkTarget(srcRoot, src string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return nil
	}
	target, err := os.Readlink(src)
	if err != nil {
		return err
	}
	if filepath.IsAbs(target) {
		return fmt.Errorf("refusing to create symlink with absolute target: %s -> %s", src, target)
	}
	resolved := filepath.Join(filepath.Dir(src), target)
	if !utils.IsPathUnder(srcRoot, resolved) {
		return fmt.Errorf("refusing to create symlink whose target escapes the install root: %s -> %s", src, target)
	}
	return nil
}

func copyDir(srcRoot, src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if err := copyEntry(srcRoot, srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}
