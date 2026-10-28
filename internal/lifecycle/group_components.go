package lifecycle

import (
	"path/filepath"
	"slices"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// prepareGroupComponents runs after the installed snapshot has been revalidated
// under HomeLock. It reuses the standalone component staging rules while the
// caller keeps every member's changes in the same toolchain-group transaction.
func prepareGroupComponents(preparation *dist.Preparation, member *preparedGroupMember, roots component.Roots) ([]rootChange, error) {
	if err := validateInstallation(roots.TcDir, member.release.Tuple); err != nil {
		return nil, err
	}
	base, err := preparation.TempDir(".cjv-component-group-*")
	if err != nil {
		return nil, err
	}
	staging := component.Roots{TcDir: filepath.Join(base, "sdk"), DocsDir: filepath.Join(base, "docs"), StdxDir: filepath.Join(base, "stdx")}
	var added, removed []component.Name
	for _, intent := range member.intents {
		if !slices.Contains(member.installed.intents, intent) {
			added = append(added, intent.Name)
		}
	}
	for _, intent := range member.installed.intents {
		if !slices.Contains(member.intents, intent) {
			removed = append(removed, intent.Name)
		}
	}
	if err := component.StagePreparedChanges(roots, member.prepared, staging, added, removed, false); err != nil {
		return nil, err
	}
	record := toolchain.Installation{Release: member.release.Name, Tuple: member.release.Tuple, SHA256: member.release.SHA256}
	if err := toolchain.WriteInstallation(staging.TcDir, record); err != nil {
		return nil, err
	}
	changes, err := componentRootChanges(staging, roots, append(added, removed...))
	if err != nil {
		return nil, err
	}
	return append(changes, rootChange{preparedRecordPath(staging.TcDir), preparedRecordPath(roots.TcDir)}), nil
}
