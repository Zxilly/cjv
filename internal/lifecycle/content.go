package lifecycle

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/fstx"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// preserveComponentMetadata keeps the ownership records for external component
// roots when replacing an SDK under the same name. Archives cannot take over
// existing component records: those records describe the files already on disk.
func preserveComponentMetadata(stagingDir, installedDir string) error {
	names, err := component.ListInstalled(installedDir)
	if err != nil {
		return err
	}
	for _, name := range names {
		paths, err := component.ReadManifest(installedDir, name)
		if err != nil {
			return err
		}
		if err := component.WriteManifest(stagingDir, name, paths); err != nil {
			return err
		}
	}
	return nil
}

// RemoveToolchain retires the SDK, its external component roots, and settings
// references together. Linked SDKs and components are removed as links only.
func RemoveToolchain(name string) error {
	if err := PrepareToolchainRemoval(name); err != nil {
		return err
	}
	roots, err := component.RootsFor(name)
	if err != nil {
		return err
	}
	sf, settings, err := LoadSettings()
	if err != nil {
		return err
	}
	update, err := referencesAfterRemoval(settings, name)
	if err != nil {
		return err
	}
	return retireToolchain(roots, sf, settings, update)
}

// PrepareToolchainRemoval recovers interrupted changes before checking whether
// a toolchain exists. CLI callers use this before prompting, so the source is
// present and inspectable when the user confirms removal. RemoveToolchain also
// repeats it to protect direct callers and revalidate after confirmation.
func PrepareToolchainRemoval(name string) error {
	if _, err := toolchain.ParseToolchainName(name); err != nil {
		return err
	}
	roots, err := component.RootsFor(name)
	if err != nil {
		return err
	}
	if err := fstx.Recover(filepath.Dir(roots.TcDir)); err != nil {
		return err
	}
	if _, err := os.Lstat(roots.TcDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &cjverr.ToolchainNotInstalledError{Name: name}
		}
		return err
	}
	return nil
}

// retireToolchain removes all managed content in one journal. Failed staging or
// settings writes restore the old installation; failed recovery retains its
// journal so normal startup can retry without losing its ownership records.
func retireToolchain(roots component.Roots, sf *config.SettingsFile, before *config.Settings, update config.SettingsUpdate) (retErr error) {
	home, err := config.Home()
	if err != nil {
		return err
	}
	if err := fstx.Recover(filepath.Dir(roots.TcDir)); err != nil {
		return err
	}
	tx, err := fstx.NewToolchainTransaction(home, filepath.Base(roots.TcDir))
	if err != nil {
		return err
	}
	committed, settingsAttempted := false, false
	defer func() {
		if committed {
			return
		}
		if err := tx.Rollback(); err != nil {
			// The old content may still be in its journal. Never repoint
			// settings to it until recovery has put every root back.
			retErr = &retirementError{Err: errors.Join(retErr, err), KeepReplacement: settingsAttempted}
			return
		}
		if settingsAttempted {
			if err := sf.Save(before); err != nil {
				retErr = &retirementError{Err: errors.Join(retErr, fmt.Errorf("restore toolchain settings: %w", err)), KeepReplacement: true}
			}
		}
	}()
	// Publish references while both versions are still usable. Interrupted
	// retirement can then restore old content without ever removing the SDK
	// selected by settings. The journal owns content recovery, not settings.
	settingsAttempted = true
	if _, err := sf.Update(update); err != nil {
		return err
	}
	for _, path := range []string{roots.DocsDir, roots.StdxDir, roots.TcDir} {
		if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}
		if err := tx.RemoveDir(path); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// retirementError keeps a usable replacement when settings or content could
// not be restored. Callers must not undo its components or delete its SDK.
type retirementError struct {
	Err             error
	KeepReplacement bool
}

func (e *retirementError) Error() string { return e.Err.Error() }
func (e *retirementError) Unwrap() error { return e.Err }

func referencesAfterRemoval(settings *config.Settings, name string) (config.SettingsUpdate, error) {
	update := config.SettingsUpdate{}
	if settings.DefaultToolchain == name {
		next, err := nextDefaultAfterRemoval(name)
		if err != nil {
			return update, err
		}
		update.DefaultToolchain = &next
	}
	for dir, tc := range settings.Overrides {
		if tc == name {
			if update.Overrides == nil {
				update.Overrides = maps.Clone(settings.Overrides)
			}
			delete(update.Overrides, dir)
		}
	}
	return update, nil
}

func nextDefaultAfterRemoval(name string) (string, error) {
	installed, err := toolchain.ListInstalled()
	if err != nil {
		return "", err
	}
	idx := slices.IndexFunc(installed, func(candidate string) bool {
		parsed, err := toolchain.ParseToolchainName(candidate)
		return candidate != name && err == nil && parsed.Target == ""
	})
	if idx < 0 {
		return "", nil
	}
	return installed[idx], nil
}
