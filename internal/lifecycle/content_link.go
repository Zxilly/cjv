package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
)

type linkRemovalSettings interface {
	Update(config.SettingsUpdate) (bool, error)
	Save(*config.Settings) error
}

// unlinkToolchainLocked requires the home lock. Only an SDK link with no
// same-name component roots can use this irreversible removal path. Component
// editing rejects linked SDKs, so existing side roots are ambiguous leftovers,
// not permission to perform a partial, mixed transactional removal.
func unlinkToolchainLocked(roots component.Roots, sf linkRemovalSettings, before *config.Settings, update config.SettingsUpdate, unlink func(*os.Root, string) error) (retErr error) {
	home, err := config.Home()
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(home)
	if err != nil {
		return err
	}
	defer root.Close() //nolint:errcheck
	// Match the transaction's ownership boundary: never traverse a linked
	// parent directory, even when it points elsewhere within this home.
	for _, subdir := range []string{config.ToolchainsSubdir, config.DocsSubdir, config.StdxSubdir} {
		info, err := root.Lstat(subdir)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("cannot unlink SDK through non-directory or linked parent %s", subdir)
		}
	}
	name := filepath.Base(roots.TcDir)
	for _, subdir := range []string{config.DocsSubdir, config.StdxSubdir} {
		path := filepath.Join(subdir, name)
		if _, err := root.Lstat(path); err == nil {
			return fmt.Errorf("cannot unlink SDK %s while component root %s exists", name, path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	// Hold the toolchains directory itself so the final name remains scoped
	// even if an unrelated process renames a parent directory.
	links, err := root.OpenRoot(config.ToolchainsSubdir)
	if err != nil {
		return err
	}
	defer links.Close() //nolint:errcheck
	info, err := links.Lstat(name)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("SDK entry %s is no longer a symbolic link", name)
	}
	target, err := links.Readlink(name)
	if err != nil {
		return err
	}
	stillLinked := func() error {
		if err := sameToolchainLink(links, name, info, target); err != nil {
			return err
		}
		// A pinned directory can remain usable after its canonical name is
		// moved or replaced. Settings refer to the canonical name, so verify
		// it too before deleting the entry or restoring those references.
		parent, err := root.Lstat(config.ToolchainsSubdir)
		if err != nil {
			return err
		}
		if !parent.IsDir() || parent.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("SDK parent %s changed during removal", config.ToolchainsSubdir)
		}
		return sameToolchainLink(root, filepath.Join(config.ToolchainsSubdir, name), info, target)
	}
	settingsAttempted, removed := false, false
	defer func() {
		if !settingsAttempted || removed {
			return
		}
		// A failed or interrupted unlink must never restore references to an
		// absent or replaced entry. Keep them cleared when state is uncertain.
		if err := stillLinked(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("toolchain settings left cleared: %w", err))
			return
		}
		if err := sf.Save(before); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("restore toolchain settings: %w", err))
		}
	}()
	settingsAttempted = true
	if _, err := sf.Update(update); err != nil {
		return err
	}
	if err := stillLinked(); err != nil {
		return err
	}
	if err := unlink(links, name); err != nil {
		return err
	}
	removed = true
	return nil
}

func sameToolchainLink(root *os.Root, name string, before os.FileInfo, target string) error {
	info, err := root.Lstat(name)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink == 0 || !os.SameFile(before, info) {
		return fmt.Errorf("SDK link %s changed during removal", name)
	}
	currentTarget, err := root.Readlink(name)
	if err != nil {
		return err
	}
	if currentTarget != target {
		return fmt.Errorf("SDK link %s target changed during removal", name)
	}
	return nil
}
