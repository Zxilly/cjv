package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Zxilly/cjv/internal/cli/selfmgmt"
	clisettings "github.com/Zxilly/cjv/internal/cli/settings"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/selfupdate"
	"github.com/charmbracelet/huh"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	initYes              bool
	initDefaultToolchain string
	initNoModifyPath     bool
	initMirror           bool
)

func init() {
	initCmd.Flags().BoolVarP(&initYes, "yes", "y", false, "Skip confirmation prompt")
	initCmd.Flags().StringVar(&initDefaultToolchain, "default-toolchain", "lts", "Default toolchain to install (use 'none' to skip)")
	initCmd.Flags().BoolVar(&initNoModifyPath, "no-modify-path", false, "Do not modify PATH")
	initCmd.Flags().BoolVar(&initMirror, "mirror", false, "Use mirror for toolchain downloads")
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Install cjv and configure the environment",
	Long:  "Set up cjv for first use: copy binary, create proxy links, configure PATH, and optionally install a default toolchain.",
	RunE:  runInit,
}

const (
	menuProceed   = "proceed"
	menuCustomize = "customize"
	menuCancel    = "cancel"
)

func yesNoStr(b bool) string {
	if b {
		return i18n.T("Yes", nil)
	}
	return i18n.T("No", nil)
}

func runInit(cmd *cobra.Command, _ []string) error {
	selfmgmt.CheckSudoSafety()

	home, err := config.Home()
	if err != nil {
		return err
	}
	binDir, err := config.BinDir()
	if err != nil {
		return err
	}

	managedPath := filepath.Join(binDir, proxy.CjvBinaryName())
	if _, err := os.Stat(managedPath); err == nil {
		fmt.Println(i18n.T("InitAlreadyInstalled", i18n.MsgData{"Path": managedPath}))

		if !initYes {
			confirm := false
			if err := huh.NewConfirm().
				Title(i18n.T("InitReinstallConfirm", nil)).
				Value(&confirm).
				Run(); err != nil {
				return err
			}
			if !confirm {
				return nil
			}
		}
	}

	// Effective options — initialized from CLI flags, may be modified by interactive menu
	toolchain := initDefaultToolchain
	modifyPath := !initNoModifyPath
	useMirror := initMirror

	fmt.Println()
	color.Cyan(i18n.T("InitWelcome", nil))
	fmt.Println()
	fmt.Println(i18n.T("InitDescription", nil))
	fmt.Println()

	fmt.Println(i18n.T("InitDataDir", nil))
	fmt.Println()
	fmt.Printf("    %s\n", home)
	fmt.Println()
	fmt.Println(i18n.T("InitDataDirEnvHint", nil))
	fmt.Println()

	fmt.Println(i18n.T("InitCommandsAvailable", nil))
	fmt.Println()
	fmt.Printf("    %s\n", binDir)
	fmt.Println()

	if !modifyPath {
		fmt.Println(i18n.T("InitPathNeedManual", nil))
	} else if runtime.GOOS == "windows" {
		fmt.Println(i18n.T("InitRegistryPath", nil))
	} else {
		fmt.Println(i18n.T("InitShellConfigs", nil))
		fmt.Println()
		posix, fish := env.ShellConfigPaths()
		for _, rc := range posix {
			fmt.Printf("    %s\n", rc)
		}
		if fish != "" {
			fmt.Printf("    %s\n", fish)
		}
	}
	fmt.Println()

	fmt.Println(i18n.T("InitUninstallHint", nil))

	if !initYes {
		customized := false
	menuLoop:
		for {
			fmt.Println()
			fmt.Println(i18n.T("InitCurrentOptions", nil))
			fmt.Println()
			fmt.Printf("   %s %s\n", i18n.T("InitOptToolchain", nil), toolchain)
			fmt.Printf("   %s %s\n", i18n.T("InitOptModifyPath", nil), yesNoStr(modifyPath))
			fmt.Printf("   %s %s\n", i18n.T("InitOptMirror", nil), yesNoStr(useMirror))
			fmt.Println()

			proceedLabel := i18n.T("InitProceedStandard", nil)
			if customized {
				proceedLabel = i18n.T("InitProceedSelected", nil)
			}

			var choice string
			if err := huh.NewSelect[string]().
				Options(
					huh.NewOption(proceedLabel, menuProceed),
					huh.NewOption(i18n.T("InitCustomize", nil), menuCustomize),
					huh.NewOption(i18n.T("InitCancelInstall", nil), menuCancel),
				).
				Value(&choice).
				Run(); err != nil {
				return err
			}

			switch choice {
			case menuCancel:
				return nil
			case menuProceed:
				break menuLoop
			case menuCustomize:
				customized = true

				if err := huh.NewSelect[string]().
					Title(i18n.T("InitToolchainQuestion", nil)).
					Options(
						huh.NewOption("lts", "lts"),
						huh.NewOption("sts", "sts"),
						huh.NewOption("nightly", "nightly"),
						huh.NewOption(i18n.T("InitToolchainNone", nil), "none"),
					).
					Value(&toolchain).
					Run(); err != nil {
					return err
				}

				if err := huh.NewConfirm().
					Title(i18n.T("InitModifyPathQuestion", nil)).
					Value(&modifyPath).
					Run(); err != nil {
					return err
				}

				if err := huh.NewConfirm().
					Title(i18n.T("InitMirrorQuestion", nil)).
					Value(&useMirror).
					Run(); err != nil {
					return err
				}
			}
		}
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	if err := config.EnsureDirs(); err != nil {
		return err
	}
	if _, err := selfupdate.ForceUpdateManagedExecutable(); err != nil {
		return err
	}
	if err := proxy.CreateAllProxyLinks(); err != nil {
		return err
	}
	if modifyPath {
		ensurePathConfiguredFn()
	}
	if err := env.WriteEnvScripts(home, binDir); err != nil {
		slog.Warn("failed to write env scripts", "error", err)
	}

	if useMirror {
		sf, settings, err := clisettings.LoadSettings()
		if err != nil {
			return err
		}
		settings.ManifestURL = config.MirrorManifestURL
		if err := sf.Save(settings); err != nil {
			return err
		}
	}

	if toolchain != "none" {
		// Prevent install from re-configuring PATH — init already did it
		if err := os.Setenv(config.EnvNoPathSetup, "1"); err != nil {
			return err
		}
		defer os.Unsetenv(config.EnvNoPathSetup) //nolint:errcheck // best-effort cleanup
		if err := InstallToolchainWithOptions(ctx, toolchain, false); err != nil {
			fmt.Fprintf(os.Stderr, "\n%s\n", i18n.T("InitToolchainFailed", i18n.MsgData{
				"Name": toolchain,
				"Err":  err.Error(),
			}))
		}
	}

	fmt.Println()
	color.Green(i18n.T("InitComplete", nil))
	fmt.Println()
	fmt.Println(i18n.T("InitSourceHint", nil))
	fmt.Println()
	fmt.Println(i18n.T("InitSourceHintRun", nil))
	fmt.Println()
	if runtime.GOOS != "windows" {
		envPath := filepath.Join(home, "env")
		fmt.Printf("    source \"%s\"\n", envPath)
	} else {
		ps1Path := filepath.Join(home, "env.ps1")
		batPath := filepath.Join(home, "env.bat")
		fmt.Printf("    PowerShell: . \"%s\"\n", ps1Path)
		fmt.Printf("    CMD:        \"%s\"\n", batPath)
	}
	fmt.Println()

	if toolchain == "none" {
		fmt.Println(i18n.T("InitInstallHint", nil))
		fmt.Println()
		fmt.Println("    cjv install <toolchain>")
		fmt.Println()
	}
	if !modifyPath {
		fmt.Println(i18n.T("InitNoModifyPath", i18n.MsgData{"BinDir": binDir}))
	}

	return nil
}
