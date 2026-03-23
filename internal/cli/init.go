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
	alreadyInstalled := false
	if _, err := os.Stat(managedPath); err == nil {
		alreadyInstalled = true
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

	fmt.Println()
	color.Cyan(i18n.T("InitWelcome", nil))
	fmt.Println()
	fmt.Println(i18n.T("InitSummary", nil))
	fmt.Println()
	fmt.Println(i18n.T("InitCjvHome", i18n.MsgData{"Path": home}))
	fmt.Println(i18n.T("InitBinDir", i18n.MsgData{"Path": binDir}))
	if initDefaultToolchain != "none" {
		fmt.Println(i18n.T("InitDefaultToolchain", i18n.MsgData{"Name": initDefaultToolchain}))
	}
	fmt.Println()

	if !initNoModifyPath {
		if runtime.GOOS == "windows" {
			fmt.Println(i18n.T("InitRegistryPath", nil))
		} else {
			fmt.Println(i18n.T("InitShellConfigs", nil))
			posix, fish := env.ShellConfigPaths()
			for _, rc := range posix {
				fmt.Printf("  %s\n", rc)
			}
			if fish != "" {
				fmt.Printf("  %s\n", fish)
			}
		}
		fmt.Println()
	}

	if !initYes && !alreadyInstalled {
		confirm := true
		if err := huh.NewConfirm().
			Title(i18n.T("InitConfirm", nil)).
			Value(&confirm).
			Run(); err != nil {
			return err
		}
		if !confirm {
			return nil
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
	if !initNoModifyPath {
		ensurePathConfiguredFn()
	}
	if err := env.WriteEnvScripts(home, binDir); err != nil {
		slog.Warn("failed to write env scripts", "error", err)
	}

	if initMirror {
		sf, settings, err := clisettings.LoadSettings()
		if err != nil {
			return err
		}
		settings.ManifestURL = config.MirrorManifestURL
		if err := sf.Save(settings); err != nil {
			return err
		}
	}

	if initDefaultToolchain != "none" {
		// Prevent install from re-configuring PATH — init already did it
		os.Setenv("CJV_NO_PATH_SETUP", "1")
		defer os.Unsetenv("CJV_NO_PATH_SETUP")
		if err := InstallToolchainWithOptions(ctx, initDefaultToolchain, false); err != nil {
			fmt.Fprintf(os.Stderr, "\n%s\n", i18n.T("InitToolchainFailed", i18n.MsgData{
				"Name": initDefaultToolchain,
				"Err":  err.Error(),
			}))
			// Don't return error — cjv itself is installed successfully
		}
	}

	fmt.Println()
	color.Green(i18n.T("InitComplete", nil))
	fmt.Println()
	if runtime.GOOS != "windows" {
		fmt.Println(i18n.T("InitSourceHint", nil))
		envPath := filepath.Join(home, "env")
		fmt.Printf("\n  source %s\n\n", envPath)
	} else {
		fmt.Println(i18n.T("InitSourceHint", nil))
		fmt.Println()
	}
	if initDefaultToolchain == "none" {
		fmt.Println(i18n.T("InitInstallHint", nil))
		fmt.Println()
		fmt.Println("  cjv install <toolchain>")
		fmt.Println()
	}
	if initNoModifyPath {
		fmt.Println(i18n.T("InitNoModifyPath", i18n.MsgData{"BinDir": binDir}))
	}

	return nil
}
