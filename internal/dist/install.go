package dist

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/codeclysm/extract/v4"

	"github.com/Zxilly/cjv/internal/fsops"
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

	return fsops.MoveTree(srcDir, destDir)
}
