package component

import (
	"errors"
	"fmt"
	"log/slog"
)

// ApplyChanges restores the named components and their metadata if apply fails.
// Callers describe one operation, including a batch of installations, without
// managing backup lifetimes. A failed restore retains its backup for recovery.
func ApplyChanges(roots Roots, names []Name, apply func() error) error {
	snap, err := takeSnapshot(roots, names)
	if err != nil {
		return err
	}

	err = apply()
	if err != nil {
		if restoreErr := snap.restore(); restoreErr != nil {
			return errors.Join(err, &restoreError{backupDir: snap.tempDir, err: restoreErr})
		}
	}
	if cleanupErr := snap.cleanup(); cleanupErr != nil {
		slog.Warn("failed to clean component backup", "path", snap.tempDir, "error", cleanupErr)
	}
	return err
}

type restoreError struct {
	backupDir string
	err       error
}

func (e *restoreError) Error() string {
	return fmt.Sprintf("failed to restore components; backup retained at %s: %v", e.backupDir, e.err)
}

func (e *restoreError) Unwrap() error { return e.err }

// replaceComponent keeps archive installation and local linking on the same
// replacement path. Snapshotting even a first install preserves untracked
// files which the new contents may overwrite before a later failure.
func replaceComponent(roots Roots, name Name, replaceExisting bool, paths []string, place func() error) error {
	return ApplyChanges(roots, []Name{name}, func() error {
		if replaceExisting {
			if err := Remove(roots, name); err != nil {
				return fmt.Errorf("replace: remove existing %s: %w", name, err)
			}
		}
		if err := place(); err != nil {
			return err
		}
		return WriteManifest(roots.TcDir, name, paths)
	})
}
