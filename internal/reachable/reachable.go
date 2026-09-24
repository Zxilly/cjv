// Package reachable makes an installed cjv reachable from the user's shell.
// Four things add up to that: the managed binary under CJV_HOME/bin, the
// proxy links beside it, the env scripts under CJV_HOME, and the PATH entry
// in the shell configs or the Windows registry. Every install path
// (`cjv init`, `cjv install`, `cjv toolchain link`, `cjv self update`)
// requests the same steps through one Policy instead of assembling them by
// hand, and `cjv self uninstall` runs the inverse of the PATH step here too.
package reachable

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/Zxilly/cjv/internal/selfupdate"
)

// Policy selects how far Ensure goes. The zero value establishes a missing
// managed binary and the proxy links, which every install needs.
type Policy struct {
	// ForceManagedBinary replaces the managed binary with the running
	// executable even when one exists, so a (re)installation always leaves
	// the current cjv behind. Unset, an existing managed binary is kept.
	ForceManagedBinary bool
	// EnvScripts rewrites CJV_HOME/env (env.ps1 and env.bat on Windows).
	// The scripts are static and self-locating; a failure to write them is
	// logged and never fails the operation.
	EnvScripts bool
	// ConfigurePath adds CJV_HOME/bin to the user's persistent PATH. See
	// ConfigurePath for the policy.
	ConfigurePath bool
}

// Ensure establishes the managed binary and the proxy links, then the env
// scripts and the PATH entry when policy asks for them. It is idempotent.
func Ensure(policy Policy) error {
	if policy.ForceManagedBinary {
		if _, err := selfupdate.ForceUpdateManagedExecutable(); err != nil {
			return err
		}
	} else if _, err := selfupdate.EnsureManagedExecutable(); err != nil {
		return err
	}
	if err := sdktools.CreateAllProxyLinks(); err != nil {
		return err
	}
	if policy.EnvScripts {
		if err := writeHomeEnvScripts(); err != nil {
			slog.Warn("failed to write env scripts", "error", err)
		}
	}
	if policy.ConfigurePath {
		ConfigurePath()
	}
	return nil
}

func writeHomeEnvScripts() error {
	home, err := config.Home()
	if err != nil {
		return err
	}
	binDir, err := config.BinDir()
	if err != nil {
		return err
	}
	return writeEnvScripts(home, binDir)
}

// ConfigurePath adds CJV_HOME/bin to the user's persistent PATH so proxy
// commands are available in new shells: the user PATH in the Windows
// registry, or a marker block in each POSIX shell config (and fish when its
// config directory exists). It is idempotent.
//
// CJV_NO_PATH_SETUP=1 skips the modification (CI environments, integration
// tests). Failures are logged per file and summarized once on stderr; cjv
// stays usable through the env scripts.
func ConfigurePath() {
	if os.Getenv(config.EnvNoPathSetup) == "1" {
		return
	}

	binDir, err := config.BinDir()
	if err != nil {
		return
	}

	var pathErr error

	if runtime.GOOS == "windows" {
		if err := addPathToWindowsRegistry(binDir); err != nil {
			slog.Warn("failed to add PATH to Windows registry", "error", err)
			pathErr = err
		}
	} else {
		posix, fish := ShellConfigPaths()
		for _, rc := range posix {
			if err := addPathToShellConfig(rc, binDir); err != nil {
				slog.Warn("failed to add PATH to shell config", "file", rc, "error", err)
				pathErr = err
			}
		}
		if fish != "" {
			if err := addPathToFishConfig(fish, binDir); err != nil {
				slog.Warn("failed to add PATH to fish config", "file", fish, "error", err)
				pathErr = err
			}
		}
	}

	if pathErr != nil {
		fmt.Fprintf(os.Stderr, "\n%s\n", i18n.T("PathConfigWarning", i18n.MsgData{"BinDir": binDir}))
	}
}

// RemovePath undoes ConfigurePath: it drops the cjv marker blocks from the
// shell configs, or CJV_HOME/bin from the Windows registry PATH. Failures
// are logged; an uninstall does not stop on them.
func RemovePath() {
	if runtime.GOOS != "windows" {
		posix, fish := ShellConfigPaths()
		for _, rc := range posix {
			if err := removePathFromShellConfig(rc); err != nil {
				slog.Warn("failed to clean PATH from shell config", "path", rc, "error", err)
			}
		}
		if fish != "" {
			if err := removePathFromShellConfig(fish); err != nil {
				slog.Warn("failed to clean PATH from shell config", "path", fish, "error", err)
			}
		}
		return
	}

	binDir, err := config.BinDir()
	if err != nil {
		slog.Warn("failed to determine bin directory", "error", err)
	} else if err := removePathFromWindowsRegistry(binDir); err != nil {
		slog.Warn("failed to clean PATH from registry", "error", err)
	}
}
