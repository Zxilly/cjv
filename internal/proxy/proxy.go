package proxy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
)

const maxRecursion = 20

// AutoInstallFunc is a hook for the CLI layer to provide auto-install functionality.
// It is set by the cli package to avoid circular imports.
var AutoInstallFunc func(ctx context.Context, input string) error

// ExtractToolName extracts the tool name from argv[0], stripping directory and .exe suffix.
func ExtractToolName(argv0 string) string {
	name := filepath.Base(argv0)
	if runtime.GOOS == "windows" {
		name = strings.TrimSuffix(name, ".exe")
	}
	return name
}

func extractPlusToolchain(args []string) (string, []string, error) {
	if len(args) > 0 && strings.HasPrefix(args[0], "+") {
		tc := args[0][1:]
		if tc == "" {
			return "", args, fmt.Errorf("toolchain name cannot be empty after '+'")
		}
		return tc, args[1:], nil
	}
	return "", args, nil
}

func checkRecursion(count int) error {
	if count >= maxRecursion {
		return &cjverr.RecursionLimitError{Max: maxRecursion}
	}
	return nil
}

// GetRecursionCount reads the recursion counter from the environment.
func GetRecursionCount() int {
	s := os.Getenv(config.EnvRecursionCount)
	if s == "" {
		return 0
	}
	n, _ := strconv.Atoi(s)
	if n < 0 {
		return 0
	}
	return n
}

// ResolveToolBinary returns the full path to a tool binary within a toolchain directory.
func ResolveToolBinary(toolchainDir, toolName string) (string, error) {
	relPath := ToolRelativePath(toolName)
	if relPath == "" {
		return "", &cjverr.UnknownToolError{Name: toolName}
	}
	return PlatformBinaryName(filepath.Join(toolchainDir, relPath)), nil
}

// ResolveInstalledToolBinary returns the full path to a proxy tool and verifies
// that the binary exists in the resolved toolchain.
func ResolveInstalledToolBinary(toolchainDir, toolName string) (string, error) {
	binary, err := ResolveToolBinary(toolchainDir, toolName)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(binary); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", &cjverr.ToolNotInToolchainError{Tool: toolName, Path: binary}
		}
		return "", err
	}
	return binary, nil
}

// Run is the top-level entry point for proxy mode, called when argv[0] is a known tool name.
func Run(ctx context.Context, toolName string, args []string) error {
	count := GetRecursionCount()
	if err := checkRecursion(count); err != nil {
		return err
	}

	tcOverride, remainingArgs, err := extractPlusToolchain(args)
	if err != nil {
		return err
	}

	// Load settings once for both resolution and auto-install decisions.
	// Failure is non-fatal for env/arg overrides (auto-install just won't trigger).
	var settings *config.Settings
	var settingsErr error
	sf, sfErr := config.DefaultSettingsFile()
	if sfErr != nil {
		settingsErr = sfErr
	} else {
		settings, settingsErr = sf.Load()
	}

	var tcName string
	if tcOverride != "" {
		tcName = tcOverride
	} else if envTC := os.Getenv(config.EnvToolchain); envTC != "" {
		tcName = envTC
	} else {
		if settingsErr != nil {
			return settingsErr
		}
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		resolved, _, err := config.ResolveToolchain(settings, cwd)
		if err != nil {
			return err
		}
		tcName = resolved
	}
	// The else branch returns settingsErr directly; if we reach here via
	// an override path, surface a warning so the user knows settings are broken.
	if settingsErr != nil {
		slog.Warn("failed to load settings", "error", settingsErr)
	}

	parsed, err := toolchain.ParseToolchainName(tcName)
	if err != nil {
		return err
	}
	tcDir, findErr := toolchain.FindInstalled(parsed)
	if findErr != nil {
		// Auto-install if enabled (not for custom/linked toolchains)
		if !parsed.IsCustom() && shouldAutoInstall(settings) && AutoInstallFunc != nil {
			fmt.Fprintln(os.Stderr, i18n.T("AutoInstalling", i18n.MsgData{"Name": tcName}))
			if installErr := AutoInstallFunc(ctx, tcName); installErr != nil {
				fmt.Fprintf(os.Stderr, "%s\n", i18n.T("AutoInstallFailed", i18n.MsgData{
					"Name": tcName,
					"Err":  installErr.Error(),
				}))
				return &cjverr.ToolchainNotInstalledError{Name: tcName}
			}
			// Retry finding after install
			tcDir, findErr = toolchain.FindInstalled(parsed)
			if findErr != nil {
				return &cjverr.ToolchainNotInstalledError{Name: tcName}
			}
		} else {
			return &cjverr.ToolchainNotInstalledError{Name: tcName}
		}
	}

	binary, err := ResolveInstalledToolBinary(tcDir, toolName)
	if err != nil {
		return err
	}

	envCfg := env.LoadToolchainEnv(ctx, tcDir)

	binDir, err := config.BinDir()
	if err != nil {
		return fmt.Errorf("failed to determine bin directory: %w", err)
	}
	proxyEnv := env.BuildProxyEnv(os.Environ(), env.ProxyEnvContext{
		Cfg:             envCfg,
		CjvBinDir:       binDir,
		ToolchainBinDir: filepath.Join(tcDir, "bin"),
		Recursion:       count,
		ToolchainName:   filepath.Base(tcDir),
	})

	return execTool(ctx, binary, remainingArgs, proxyEnv)
}

func shouldAutoInstall(settings *config.Settings) bool {
	return settings != nil && settings.AutoInstall
}

