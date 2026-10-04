package toolchain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/fsops"
	"github.com/Zxilly/cjv/internal/fstx"
)

// MigrateLegacy preserves version directories and materializes the old implicit
// channel selections once. It requires the home lock and recovered journals.
// No settings or project files are rewritten. A durable marker prevents a
// deliberately uninstalled channel from being recreated on the next startup.
func (lock *HomeLock) MigrateLegacy() error {
	home := filepath.Dir(lock.tcDir)
	marker := filepath.Join(home, ".cjv", "layout-v2")
	if _, err := os.Stat(marker); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	installed, err := ListInstalled()
	if err != nil {
		return err
	}
	latest := make(map[string]ToolchainName)
	for _, name := range installed {
		parsed, err := ParseToolchainName(name)
		if err != nil || parsed.IsCustom() || parsed.Channel == UnknownChannel || parsed.Version == "" {
			continue
		}
		key := ToolchainName{Channel: parsed.Channel, Target: parsed.Target}.String()
		if old, ok := latest[key]; !ok || compareSemVer(parsed.Version, old.Version) > 0 {
			latest[key] = parsed
		}
	}
	keys := make([]string, 0, len(latest))
	for key := range latest {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		dest := filepath.Join(lock.tcDir, key)
		if _, err := os.Lstat(dest); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := migrateInstallation(home, latest[key].String(), key); err != nil {
			return fmt.Errorf("migrate %s: %w", key, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
		return err
	}
	return fsops.WriteFileAtomic(marker, []byte("2\n"), 0o644)
}

func migrateInstallation(home, old, identity string) (retErr error) {
	tx, err := fstx.NewToolchainTransaction(home, identity)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		var recovery *fstx.RecoveryError
		if !errors.As(retErr, &recovery) {
			for _, base := range []string{config.ToolchainsSubdir, config.DocsSubdir, config.StdxSubdir} {
				_ = fsops.RemoveAllRetry(config.StagingDir(filepath.Join(home, base, identity)))
			}
		}
	}()
	defer func() {
		if !committed {
			retErr = errors.Join(retErr, tx.Rollback())
		}
	}()
	// Stage every root inside the transaction's permitted staging paths.
	for _, subdir := range []string{config.ToolchainsSubdir, config.DocsSubdir, config.StdxSubdir} {
		src := filepath.Join(home, subdir, old)
		if _, err := os.Lstat(src); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}
		dest := filepath.Join(home, subdir, identity)
		stage := config.StagingDir(dest)
		if err := fsops.RemoveAllRetry(stage); err != nil {
			return err
		}
		// Legacy files may have been edited by the user. Copy them rather than
		// linking them to the new channel's payload.
		source := src
		if info, err := os.Lstat(src); err != nil {
			return err
		} else if subdir == config.ToolchainsSubdir && info.Mode()&os.ModeSymlink != 0 {
			source, err = filepath.EvalSymlinks(src)
			if err != nil {
				return err
			}
		}
		if err := fsops.CopyTree(source, stage); err != nil {
			return err
		}
		if subdir == config.ToolchainsSubdir {
			if err := WriteInstallation(stage, Installation{Release: old}); err != nil {
				return err
			}
		}
		if err := tx.RenameFile(stage, dest); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}
