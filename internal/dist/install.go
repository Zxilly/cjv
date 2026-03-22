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
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create toolchain directory: %w", err)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close() //nolint:errcheck // read-only

	// Extract to temp dir on the same filesystem as destDir to ensure
	// os.Rename works without falling back to a full copy.
	tmpDir, err := os.MkdirTemp(filepath.Dir(destDir), ".cjv-install-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck // best-effort cleanup

	err = extract.Archive(ctx, f, tmpDir, nil)
	if err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	srcDir := tmpDir
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return err
	}
	if len(entries) == 1 && entries[0].IsDir() {
		srcDir = filepath.Join(tmpDir, entries[0].Name())
	}

	return moveContents(srcDir, destDir)
}

func moveContents(srcDir, destDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		src := filepath.Join(srcDir, entry.Name())
		dst := filepath.Join(destDir, entry.Name())
		if err := utils.RenameRetry(src, dst); err != nil {
			// Rename fails across filesystems; fall back to copy
			if err2 := copyEntry(src, dst); err2 != nil {
				return fmt.Errorf("failed to move %s: rename: %w, copy: %w", entry.Name(), err, err2)
			}
		}
	}
	return nil
}

func copyEntry(src, dst string) error {
	// Use Lstat to detect symlinks without following them
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	// Preserve symlinks (distinguish file vs directory on Windows)
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		// Reject absolute symlink targets to prevent malicious archives
		// from creating symlinks that point outside the installation directory.
		if filepath.IsAbs(target) {
			return fmt.Errorf("refusing to create symlink with absolute target: %s -> %s", src, target)
		}
		// Check if symlink target is a directory (follow the link)
		if targetInfo, err := os.Stat(src); err == nil && targetInfo.IsDir() {
			return utils.SymlinkOrJunction(target, dst)
		}
		return os.Symlink(target, dst)
	}

	if info.IsDir() {
		return copyDir(src, dst)
	}
	return utils.CopyFile(src, dst, info.Mode())
}

func copyDir(src, dst string) error {
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
		if err := copyEntry(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}
