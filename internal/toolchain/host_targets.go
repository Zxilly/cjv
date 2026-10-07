package toolchain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/target"
)

// HostTargets describes the cross SDKs owned by an installed host. Identity
// (tracking or pinned), artifact release and platform stay together so callers
// need not reconstruct their relationship. It neither recovers nor installs.
type HostTargets struct {
	Dir      string
	Release  ToolchainName
	Tuple    string
	identity ToolchainName
}

// ReadHostTargets reads the installed host's identity and release. Old records
// without a tuple use their explicit host name, then the configured host.
func ReadHostTargets(dir string) (HostTargets, error) {
	identity, err := ParseToolchainName(filepath.Base(dir))
	if err != nil {
		return HostTargets{}, err
	}
	if identity.IsCustom() || identity.Target != "" || identity.Channel == UnknownChannel {
		return HostTargets{}, fmt.Errorf("toolchain %s is not a published host", filepath.Base(dir))
	}
	record, err := ReadInstallation(dir)
	if err != nil {
		return HostTargets{}, err
	}
	release, err := ParseToolchainName(record.Release)
	if err != nil {
		return HostTargets{}, err
	}
	tuple := record.Tuple
	if tuple == "" {
		tuple = identity.Host
	}
	if tuple == "" {
		_, settings, err := config.LoadDefaultSettings()
		if err != nil {
			return HostTargets{}, err
		}
		tuple, err = target.CurrentHostTuple(settings.DefaultHost)
		if err != nil {
			return HostTargets{}, err
		}
	}
	id, err := target.ParseIdentity(tuple)
	if err != nil {
		return HostTargets{}, err
	}
	if id.IsTargetVariant() {
		return HostTargets{}, fmt.Errorf("toolchain %s has a cross SDK platform instead of a host", filepath.Base(dir))
	}
	return HostTargets{Dir: dir, Release: release, Tuple: tuple, identity: identity}, nil
}

// TargetName keeps the host's tracking/pinned identity, including when its
// selector was an alias, and uses the installed artifact's platform.
func (h HostTargets) TargetName(environment string) (ToolchainName, error) {
	id, err := target.ParseIdentity(h.Tuple)
	if err != nil {
		return ToolchainName{}, err
	}
	cross, err := id.WithEnvironment(environment)
	if err != nil {
		return ToolchainName{}, err
	}
	if !cross.IsTargetVariant() {
		return ToolchainName{}, fmt.Errorf("cross SDK environment must not be empty")
	}
	name := h.identity
	name.Host = ""
	name.Target = cross.Tuple()
	return name, nil
}

// FindTarget requires both an owned identity and the host's current release.
// A stale target is reported as missing so the caller can choose to install it.
func (h HostTargets) FindTarget(environment string) (string, ToolchainName, error) {
	name, err := h.TargetName(environment)
	if err != nil {
		return "", ToolchainName{}, err
	}
	dir, err := FindInstalled(name)
	if errors.Is(err, os.ErrNotExist) {
		return "", ToolchainName{}, &cjverr.ToolchainNotInstalledError{Name: name.String()}
	}
	if err != nil {
		return "", ToolchainName{}, err
	}
	release, err := InstalledRelease(dir)
	if err != nil {
		return "", ToolchainName{}, err
	}
	if release.Channel != h.Release.Channel || release.Version != h.Release.Version {
		return "", ToolchainName{}, &cjverr.ToolchainNotInstalledError{Name: name.String()}
	}
	return dir, release, nil
}

// Installed lists only targets usable with this host's current release.
// Broken records remain errors; targets from older releases are simply absent.
func (h HostTargets) Installed() ([]string, error) {
	installed, err := ListInstalled()
	if err != nil {
		return nil, err
	}
	var environments []string
	for _, raw := range installed {
		name, err := ParseToolchainName(raw)
		if err != nil || name.Channel != h.identity.Channel || name.Version != h.identity.Version || !strings.HasPrefix(name.Target, h.Tuple+"-") {
			continue
		}
		environment := strings.TrimPrefix(name.Target, h.Tuple+"-")
		if _, _, err := h.FindTarget(environment); err != nil {
			var missing *cjverr.ToolchainNotInstalledError
			if errors.As(err, &missing) {
				continue
			}
			return nil, err
		}
		environments = append(environments, environment)
	}
	return environments, nil
}
