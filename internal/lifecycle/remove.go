package lifecycle

import (
	"context"
	"errors"
	"fmt"
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

// RemoveToolchain retires an owned SDK and its component roots transactionally.
// On Unix, a linked SDK without component roots is unlinked directly after
// clearing its settings references; its user-owned target is never removed.
func RemoveToolchain(name string) error {
	if err := PrepareToolchainRemoval(name); err != nil {
		return err
	}
	parsed, err := toolchain.ParseToolchainName(name)
	if err != nil {
		return err
	}
	lock, err := toolchain.LockHome(context.Background())
	if err != nil {
		return err
	}
	defer lock.Close() //nolint:errcheck
	if err := lock.Recover(); err != nil {
		return err
	}
	dir, err := removalDir(parsed)
	if err != nil {
		return err
	}
	name = filepath.Base(dir)
	names := []string{name}
	if parsed.IsChannelOnly() {
		record, err := toolchain.ReadInstallation(dir)
		if err != nil {
			return err
		}
		hostTuple := record.Tuple
		if hostTuple == "" {
			d, err := OpenDistribution(Options{})
			if err != nil {
				return err
			}
			hostTuple = d.HostTuple
		}
		variants, err := trackedGroupTargets(parsed.Channel, hostTuple, true)
		if err != nil {
			return err
		}
		names = append(names, variants...)
	}
	return removeInstallationsLocked(names)
}

func removeInstallationsLocked(names []string) error {
	roots := make([]component.Roots, 0, len(names))
	for _, name := range names {
		member, err := component.RootsFor(name)
		if err != nil {
			return err
		}
		if _, err := os.Lstat(member.TcDir); err != nil {
			return err
		}
		roots = append(roots, member)
	}
	sf, _, err := config.LoadDefaultSettings()
	if err != nil {
		return err
	}
	if len(roots) == 1 {
		info, err := os.Lstat(roots[0].TcDir)
		if err != nil {
			return err
		}
		if unlink := linkedToolchainRemover(info); unlink != nil {
			return unlinkToolchainLocked(roots[0], sf, nil, config.SettingsUpdate{}, unlink)
		}
	}
	return retireGroupLocked(names, roots, sf)
}

// PrepareToolchainRemoval recovers interrupted changes before checking whether
// a toolchain exists. CLI callers use this before prompting, so the source is
// present and inspectable when the user confirms removal. RemoveToolchain also
// repeats it to protect direct callers and revalidate after confirmation.
func PrepareToolchainRemoval(name string) error {
	if _, err := toolchain.ParseToolchainName(name); err != nil {
		return err
	}
	if err := toolchain.RecoverHome(); err != nil {
		return err
	}
	parsed, err := toolchain.ParseToolchainName(name)
	if err != nil {
		return err
	}
	dir, err := removalDir(parsed)
	if err != nil {
		return err
	}
	roots, err := component.RootsFor(filepath.Base(dir))
	if err != nil {
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

func removalDir(name toolchain.ToolchainName) (string, error) {
	// Lstat below must still permit unlinking a broken custom SDK link.
	if name.IsCustom() || name.Channel != toolchain.UnknownChannel && name.Version != "" && name.Host == "" && !toolchain.IsVersionSelector(name.Version) {
		return config.ToolchainDirFor(name.String())
	}
	return toolchain.FindInstalled(name)
}

// retireGroupLocked requires the home lock to stay held through settings publication
// and transaction cleanup.
func retireGroupLocked(names []string, roots []component.Roots, sf *config.SettingsFile) (retErr error) {
	home, err := config.Home()
	if err != nil {
		return err
	}
	tx, err := fstx.NewToolchainGroupTransaction(home, names)
	if err != nil {
		return err
	}
	committed := false
	var undo func() error
	defer func() {
		if committed {
			return
		}
		if err := tx.Rollback(); err != nil {
			// The old content may still be in its journal. Never repoint
			// settings to it until recovery has put every root back.
			retErr = errors.Join(retErr, err)
			return
		}
		if undo != nil {
			if err := undo(); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("restore toolchain settings: %w", err))
			}
		}
	}()
	// Clear or repoint references before moving content into the journal. An
	// interrupted removal can then restore content without dangling references.
	installed, err := toolchain.ListInstalled()
	if err != nil {
		return err
	}
	_, undo, err = sf.MutateWithUndo(func(settings *config.Settings) (config.SettingsUpdate, error) {
		return referencesAfterRemovalsFromInstalled(settings, names, installed), nil
	})
	if err != nil {
		return err
	}
	for _, member := range roots {
		for _, path := range []string{member.DocsDir, member.StdxDir, member.TcDir} {
			if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
				continue
			} else if err != nil {
				return err
			}
			if err := tx.RemoveDir(path); err != nil {
				return err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func referencesAfterRemovalsFromInstalled(settings *config.Settings, names, installed []string) config.SettingsUpdate {
	update := config.SettingsUpdate{}
	if slices.Contains(names, settings.DefaultToolchain) {
		next := nextDefaultAfterRemovals(names, installed)
		update.DefaultToolchain = &next
	}
	for dir, tc := range settings.Overrides {
		if slices.Contains(names, tc) {
			if update.OverrideChanges == nil {
				update.OverrideChanges = make(map[string]*string)
			}
			update.OverrideChanges[dir] = nil
		}
	}
	return update
}

func nextDefaultAfterRemovals(names, installed []string) string {
	idx := slices.IndexFunc(installed, func(candidate string) bool {
		parsed, err := toolchain.ParseToolchainName(candidate)
		return !slices.Contains(names, candidate) && err == nil && parsed.Target == ""
	})
	if idx < 0 {
		return ""
	}
	return installed[idx]
}
