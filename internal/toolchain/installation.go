package toolchain

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/Zxilly/cjv/internal/fsops"
)

const installationFile = ".cjv/toolchain.toml"

// Installation describes the release inside an independent installation.
// The directory name is its identity; Release is the artifact selection.
type Installation struct {
	Release string `toml:"release"`
	Tuple   string `toml:"tuple,omitempty"`
	SHA256  string `toml:"sha256,omitempty"`
}

func WriteInstallation(dir string, installed Installation) error {
	if _, err := ParseToolchainName(installed.Release); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(installed); err != nil {
		return err
	}
	path := filepath.Join(dir, installationFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return fsops.WriteFileAtomic(path, buf.Bytes(), 0o644)
}

// ReadInstallation accepts old version directories without metadata. Tracking
// installations always need a record: their names do not identify a release.
func ReadInstallation(dir string) (Installation, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return Installation{}, err
	}
	if !info.IsDir() {
		return Installation{}, fmt.Errorf("toolchain %s is not a directory", filepath.Base(dir))
	}
	identity, err := ParseToolchainName(filepath.Base(dir))
	if err != nil {
		return Installation{}, err
	}
	var installed Installation
	_, err = toml.DecodeFile(filepath.Join(dir, installationFile), &installed)
	if errors.Is(err, os.ErrNotExist) && (identity.Version != "" || identity.IsCustom()) {
		tuple := identity.Target
		if tuple == "" {
			tuple = identity.Host
		}
		return Installation{Release: identity.String(), Tuple: tuple}, nil
	}
	if err != nil {
		return Installation{}, err
	}
	release, err := ParseToolchainName(installed.Release)
	if err != nil || release.IsCustom() || release.Version == "" || release.Channel != identity.Channel || release.Target != identity.Target || release.Host != identity.Host || identity.Version != "" && identity.Version != release.Version {
		return Installation{}, fmt.Errorf("invalid release record for toolchain %s", identity.String())
	}
	return installed, nil
}

func InstalledRelease(dir string) (ToolchainName, error) {
	installed, err := ReadInstallation(dir)
	if err != nil {
		return ToolchainName{}, err
	}
	return ParseToolchainName(installed.Release)
}
