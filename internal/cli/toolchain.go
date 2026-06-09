package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/output"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/selfupdate"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/Zxilly/cjv/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type toolchainLinkResult struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (r toolchainLinkResult) Text() string {
	return color.GreenString(i18n.T("ToolchainLinked", i18n.MsgData{
		"Name": r.Name,
		"Path": r.Path,
	}))
}

var toolchainCmd = &cobra.Command{
	Use:   "toolchain",
	Short: i18n.T("ToolchainCmdShort", nil),
}

var toolchainListCmd = &cobra.Command{
	Use:   "list",
	Short: i18n.T("ToolchainListShort", nil),
	RunE:  runShowInstalled,
}

var toolchainLinkCmd = &cobra.Command{
	Use:   "link <name> <path>",
	Short: i18n.T("ToolchainLinkShort", nil),
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		targetPath := args[1]

		// Validate name (path traversal, empty, +prefix, etc.)
		parsed, err := toolchain.ParseToolchainName(name)
		if err != nil {
			return err
		}
		// Prevent shadowing standard channel names (lts, sts, nightly)
		if !parsed.IsCustom() {
			return errors.New(i18n.T("LinkReservedName", i18n.MsgData{"Name": name}))
		}

		absPath, err := filepath.Abs(targetPath)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("LinkInvalidPath", nil), err)
		}
		if _, err := os.Stat(absPath); err != nil {
			return errors.New(i18n.T("LinkPathNotExist", i18n.MsgData{"Path": absPath}))
		}

		// Validate the directory contains a Cangjie SDK (bin/cjc must exist)
		if _, err := proxy.ResolveInstalledToolBinary(absPath, "cjc"); err != nil {
			return fmt.Errorf("%s: %w", i18n.T("LinkNotSDK", nil), err)
		}

		tcDir, err := config.ToolchainsDir()
		if err != nil {
			return err
		}
		linkPath := filepath.Join(tcDir, name)

		if _, err := os.Stat(linkPath); err == nil {
			return &cjverr.ToolchainAlreadyInstalledError{Name: name}
		}

		if err := os.MkdirAll(tcDir, 0o755); err != nil {
			return err
		}
		if _, err := selfupdate.EnsureManagedExecutable(); err != nil {
			return err
		}

		// Create symlink (with junction fallback on Windows)
		if err := utils.SymlinkOrJunction(absPath, linkPath); err != nil {
			return fmt.Errorf("%s: %w", i18n.T("LinkCreateFailed", nil), err)
		}

		// Ensure proxy links exist in bin directory
		if err := proxy.CreateAllProxyLinks(); err != nil {
			return err
		}

		return output.RenderTo(cmdOutput(cmd), toolchainLinkResult{Name: name, Path: absPath})
	},
}

var toolchainUninstallCmd = &cobra.Command{
	Use:   "uninstall <name>",
	Short: i18n.T("ToolchainUninstallShort", nil),
	Args:  cobra.ExactArgs(1),
	RunE:  runUninstall,
}

func init() {
	toolchainUninstallCmd.Flags().BoolVarP(&uninstallYes, "yes", "y", false, i18n.T("FlagSkipConfirm", nil))
	toolchainCmd.AddCommand(toolchainListCmd)
	toolchainCmd.AddCommand(toolchainLinkCmd)
	toolchainCmd.AddCommand(toolchainUninstallCmd)
	rootCmd.AddCommand(toolchainCmd)
}
