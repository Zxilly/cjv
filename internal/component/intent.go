package component

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Intent records how an installed component should follow a toolchain upgrade.
// An empty Source means that the new version needs its own published archive.
// A Source names the user's directory, whose contents remain user-owned.
type Intent struct {
	Name   Name
	Source string
}

// InstalledIntents inspects existing installations, including links created by
// older cjv versions which did not store a separate source record.
func InstalledIntents(roots Roots) ([]Intent, error) {
	names, err := ListInstalled(roots.TcDir)
	if err != nil {
		return nil, err
	}
	intents := make([]Intent, 0, len(names))
	for _, name := range names {
		spec, err := SpecFor(name)
		if err != nil {
			return nil, err
		}
		source, err := linkedSource(roots, spec)
		if err != nil {
			return nil, err
		}
		intents = append(intents, Intent{Name: name, Source: source})
	}
	return intents, nil
}

func linkedSource(roots Roots, spec Spec) (string, error) {
	if !spec.Linkable {
		return "", nil
	}
	var source string
	links := 0
	for _, child := range spec.LinkChildren {
		path := filepath.Join(spec.InstallRoot(roots), child)
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		target, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		target = filepath.Clean(target)
		parent := filepath.Dir(target)
		if filepath.Base(target) != child || (source != "" && source != parent) {
			return "", fmt.Errorf("component %s has inconsistent linked sources", spec.Name)
		}
		source = parent
		links++
	}
	if links != 0 && links != len(spec.LinkChildren) {
		return "", fmt.Errorf("component %s has an incomplete linked source", spec.Name)
	}
	return source, nil
}
