package lifecycle

import (
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/fsops"
	"github.com/Zxilly/cjv/internal/fstx"
	"github.com/Zxilly/cjv/internal/progress"
)

// publishComponentBatch runs under the home lock after installation state has
// been revalidated. Content roots and the component index share one durable
// decision; a killed process is recovered by the ordinary RecoverHome path.
func publishComponentBatch(home, identity string, prepared component.Roots, names []component.Name, force bool, opts Options) (retErr error) {
	roots, err := component.RootsFor(identity)
	if err != nil {
		return err
	}
	staging := component.Roots{
		TcDir: config.StagingDir(roots.TcDir), DocsDir: config.StagingDir(roots.DocsDir), StdxDir: config.StagingDir(roots.StdxDir),
	}
	stageDirs := []string{staging.TcDir, staging.DocsDir, staging.StdxDir}
	defer func() {
		var recovery *fstx.RecoveryError
		if !errors.As(retErr, &recovery) {
			for _, dir := range stageDirs {
				_ = fsops.RemoveAllRetry(dir)
			}
		}
	}()
	for _, dir := range stageDirs {
		if err := fsops.RemoveAllRetry(dir); err != nil {
			return err
		}
	}
	if err := component.StagePreparedBatch(roots, prepared, staging, names, force); err != nil {
		return err
	}
	var pairs [][2]string
	for _, name := range names {
		spec, err := component.SpecFor(name)
		if err != nil {
			return err
		}
		pair := [2]string{spec.InstallRoot(staging), spec.InstallRoot(roots)}
		if !slices.Contains(pairs, pair) {
			pairs = append(pairs, pair)
		}
	}
	pairs = append(pairs, [2]string{filepath.Join(staging.TcDir, component.MetaDir), filepath.Join(roots.TcDir, component.MetaDir)})
	tx, err := fstx.NewToolchainTransaction(home, identity)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			retErr = errors.Join(retErr, tx.Rollback())
		}
	}()
	for _, pair := range pairs {
		if _, err := os.Lstat(pair[1]); err == nil {
			if err := tx.RemoveDir(pair[1]); err != nil {
				return err
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := tx.RenameFile(pair[0], pair[1]); err != nil {
			return err
		}
		if afterComponentRootHook != nil {
			if err := afterComponentRootHook(pair[1]); err != nil {
				return err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	for _, name := range names {
		opts.emit(progress.Event{Kind: progress.ComponentInstalled, Toolchain: identity, Component: string(name)})
	}
	return nil
}

// Tests interrupt or obstruct publication after one durable root placement.
var afterComponentRootHook func(string) error
