package resolve

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// AutoInstallFunc is wired by main to avoid importing cli from lower-level packages.
var AutoInstallFunc func(ctx context.Context, input string, targets []string) error

// AutoInstallComponentsFunc auto-installs missing components on demand. Wired
// by main; must be safe to leave nil in tests where component installation
// is not exercised.
var AutoInstallComponentsFunc func(ctx context.Context, input string, components []string) error

type ActiveToolchain struct {
	Dir        string
	Name       string
	Source     config.OverrideSource
	Targets    []string
	Components []string
}

func Active(ctx context.Context, tcOverride string) (ActiveToolchain, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	toolchain.CleanupStagingDirs()

	settings, settingsErr := loadSettings()
	tcName, source, targets, components, err := resolveName(settings, settingsErr, tcOverride)
	if err != nil {
		return ActiveToolchain{}, err
	}
	if settingsErr != nil {
		slog.Warn("failed to load settings", "error", settingsErr)
	}

	parsed, err := toolchain.ParseToolchainName(tcName)
	if err != nil {
		return ActiveToolchain{}, err
	}
	if parsed.PlatformKey != "" {
		hostName := toolchain.ToolchainName{
			Channel: parsed.Channel,
			Version: parsed.Version,
		}.String()
		return ActiveToolchain{}, fmt.Errorf("target variant %q cannot be used as the active toolchain; use host toolchain %q and configure targets instead", tcName, hostName)
	}

	tcDir, findErr := toolchain.FindInstalled(parsed)
	if findErr != nil {
		if !parsed.IsCustom() && shouldAutoInstall(settings) && AutoInstallFunc != nil {
			fmt.Fprintln(os.Stderr, i18n.T("AutoInstalling", i18n.MsgData{"Name": tcName}))
			if installErr := AutoInstallFunc(ctx, tcName, targets); installErr != nil {
				fmt.Fprintf(os.Stderr, "%s\n", i18n.T("AutoInstallFailed", i18n.MsgData{
					"Name": tcName,
					"Err":  installErr.Error(),
				}))
				return ActiveToolchain{}, &cjverr.ToolchainNotInstalledError{Name: tcName}
			}
			tcDir, findErr = toolchain.FindInstalled(parsed)
		}
		if findErr != nil {
			if !errors.Is(findErr, os.ErrNotExist) {
				return ActiveToolchain{}, findErr
			}
			return ActiveToolchain{}, &cjverr.ToolchainNotInstalledError{Name: tcName}
		}
	}

	if err := ensureTargets(ctx, filepath.Base(tcDir), tcDir, settings, targets); err != nil {
		return ActiveToolchain{}, err
	}

	if err := ensureComponents(ctx, filepath.Base(tcDir), tcDir, settings, components); err != nil {
		return ActiveToolchain{}, err
	}

	return ActiveToolchain{
		Dir:        tcDir,
		Name:       filepath.Base(tcDir),
		Source:     source,
		Targets:    targets,
		Components: components,
	}, nil
}

func loadSettings() (*config.Settings, error) {
	sf, err := config.DefaultSettingsFile()
	if err != nil {
		return nil, err
	}
	return sf.Load()
}

func resolveName(settings *config.Settings, settingsErr error, tcOverride string) (string, config.OverrideSource, []string, []string, error) {
	if tcOverride != "" {
		return tcOverride, config.SourceUnknown, nil, nil, nil
	}
	if envTC := os.Getenv(config.EnvToolchain); envTC != "" {
		return envTC, config.SourceEnv, nil, nil, nil
	}
	if settingsErr != nil {
		return "", config.SourceUnknown, nil, nil, settingsErr
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", config.SourceUnknown, nil, nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	resolved, err := config.ResolveToolchainConfig(settings, cwd)
	if err != nil {
		return "", config.SourceUnknown, nil, nil, err
	}
	return resolved.Name, resolved.Source, resolved.Targets, resolved.Components, nil
}

func ensureTargets(ctx context.Context, tcInput, tcDir string, settings *config.Settings, targets []string) error {
	if len(targets) == 0 {
		return nil
	}

	host, err := toolchain.ParseToolchainName(filepath.Base(tcDir))
	if err != nil {
		return err
	}
	if host.IsCustom() || host.Channel == toolchain.UnknownChannel || host.Version == "" {
		return nil
	}

	var missingTargets []string
	var missingNames []string
	for _, target := range targets {
		platformKey, err := targetPlatformKey(settings, target)
		if err != nil {
			return err
		}
		name := toolchain.ToolchainName{
			Channel:     host.Channel,
			Version:     host.Version,
			PlatformKey: platformKey,
		}
		if _, err := toolchain.FindInstalled(name); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			missingTargets = append(missingTargets, target)
			missingNames = append(missingNames, name.String())
		}
	}
	if len(missingTargets) == 0 {
		return nil
	}

	if !shouldAutoInstall(settings) || AutoInstallFunc == nil {
		return &cjverr.ToolchainNotInstalledError{Name: missingNames[0]}
	}

	fmt.Fprintln(os.Stderr, i18n.T("AutoInstalling", i18n.MsgData{"Name": strings.Join(missingNames, ", ")}))
	if installErr := AutoInstallFunc(ctx, tcInput, missingTargets); installErr != nil {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("AutoInstallFailed", i18n.MsgData{
			"Name": strings.Join(missingNames, ", "),
			"Err":  installErr.Error(),
		}))
		return &cjverr.ToolchainNotInstalledError{Name: missingNames[0]}
	}

	for _, missingName := range missingNames {
		parsed, err := toolchain.ParseToolchainName(missingName)
		if err != nil {
			return err
		}
		if _, err := toolchain.FindInstalled(parsed); err != nil {
			return &cjverr.ToolchainNotInstalledError{Name: missingName}
		}
	}
	return nil
}

func shouldAutoInstall(settings *config.Settings) bool {
	return settings != nil && settings.AutoInstall
}

func ensureComponents(ctx context.Context, tcInput, tcDir string, settings *config.Settings, components []string) error {
	if len(components) == 0 {
		return nil
	}

	parsedNames, err := component.NormalizeList(components)
	if err != nil {
		return err
	}

	var missingNames []component.Name
	for _, n := range parsedNames {
		if !component.IsInstalled(tcDir, n) {
			missingNames = append(missingNames, n)
		}
	}
	if len(missingNames) == 0 {
		return nil
	}

	asStrings := make([]string, len(missingNames))
	for i, n := range missingNames {
		asStrings[i] = string(n)
	}

	if !shouldAutoInstall(settings) || AutoInstallComponentsFunc == nil {
		return &cjverr.ComponentNotInstalledError{
			Toolchain: filepath.Base(tcDir),
			Component: asStrings[0],
		}
	}

	fmt.Fprintln(os.Stderr, i18n.T("AutoInstalling", i18n.MsgData{
		"Name": strings.Join(asStrings, ", "),
	}))
	if err := AutoInstallComponentsFunc(ctx, tcInput, asStrings); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("AutoInstallFailed", i18n.MsgData{
			"Name": strings.Join(asStrings, ", "),
			"Err":  err.Error(),
		}))
		return &cjverr.ComponentNotInstalledError{
			Toolchain: filepath.Base(tcDir),
			Component: asStrings[0],
		}
	}
	for _, n := range missingNames {
		if !component.IsInstalled(tcDir, n) {
			return &cjverr.ComponentNotInstalledError{
				Toolchain: filepath.Base(tcDir),
				Component: string(n),
			}
		}
	}
	return nil
}

func targetPlatformKey(settings *config.Settings, target string) (string, error) {
	defaultHost := ""
	if settings != nil {
		defaultHost = settings.DefaultHost
	}
	return dist.CurrentPlatformKeyWithTarget(defaultHost, target)
}
