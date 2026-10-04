package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

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
	parsed, err := toolchain.ParseToolchainName(name)
	if err != nil {
		return err
	}
	dir, err := removalDir(parsed)
	if err != nil {
		return err
	}
	name = filepath.Base(dir)
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
		installed, err := toolchain.ListInstalled()
		if err != nil {
			return err
		}
		for _, variant := range installed {
			p, err := toolchain.ParseToolchainName(variant)
			if err == nil && !p.IsCustom() && p.Channel == parsed.Channel && p.Version == "" && p.Target != "" && strings.HasPrefix(p.Target, hostTuple+"-") {
				if err := removeInstallation(variant); err != nil {
					return err
				}
			}
		}
	}
	return removeInstallation(name)
}

func removeInstallation(name string) error {
	lock, err := toolchain.LockHome(context.Background())
	if err != nil {
		return err
	}
	defer lock.Close() //nolint:errcheck
	if err := lock.Recover(); err != nil {
		return err
	}
	roots, err := component.RootsFor(name)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(roots.TcDir); err != nil {
		return err
	}
	sf, _, err := config.LoadDefaultSettings()
	if err != nil {
		return err
	}
	sf.Invalidate()
	settings, err := sf.Load()
	if err != nil {
		return err
	}
	update, err := referencesAfterRemoval(settings, name)
	if err != nil {
		return err
	}
	return retireLocked(roots, sf, settings, update)
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

// retireLocked requires the home lock to stay held through settings publication
// and transaction cleanup.
func retireLocked(roots component.Roots, sf *config.SettingsFile, before *config.Settings, update config.SettingsUpdate) (retErr error) {
	home, err := config.Home()
	if err != nil {
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
			retErr = errors.Join(retErr, err)
			return
		}
		if settingsAttempted {
			if err := sf.Save(before); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("restore toolchain settings: %w", err))
			}
		}
	}()
	// Clear or repoint references before moving content into the journal. An
	// interrupted removal can then restore content without dangling references.
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
