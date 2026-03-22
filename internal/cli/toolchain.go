package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/selfupdate"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/Zxilly/cjv/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var toolchainCmd = &cobra.Command{
	Use:   "toolchain",
	Short: "Manage installed toolchains",
}

var toolchainListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed toolchains",
	RunE:  runShowInstalled,
}

var toolchainLinkCmd = &cobra.Command{
	Use:   "link <name> <path>",
	Short: "Link a custom toolchain to a local directory",
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
			return fmt.Errorf("'%s' is a reserved channel name; use a unique custom name for linked toolchains", name)
		}

		absPath, err := filepath.Abs(targetPath)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}
		if _, err := os.Stat(absPath); err != nil {
			return fmt.Errorf("path does not exist: %s", absPath)
		}

		// Validate the directory contains a Cangjie SDK (bin/cjc must exist)
		if _, err := proxy.ResolveInstalledToolBinary(absPath, "cjc"); err != nil {
			return fmt.Errorf("directory does not appear to contain a Cangjie SDK: %w", err)
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
			return fmt.Errorf("failed to create link: %w", err)
		}

		// Ensure proxy links exist in bin directory
		if err := proxy.CreateAllProxyLinks(); err != nil {
			return err
		}

		color.Green(i18n.T("ToolchainLinked", i18n.MsgData{
			"Name": name,
			"Path": absPath,
		}))
		return nil
	},
}

var toolchainUninstallCmd = &cobra.Command{
	Use:   "uninstall <name>",
	Short: "Uninstall a toolchain",
	Args:  cobra.ExactArgs(1),
	RunE:  runUninstall,
}

func init() {
	toolchainCmd.AddCommand(toolchainListCmd)
	toolchainCmd.AddCommand(toolchainLinkCmd)
	toolchainCmd.AddCommand(toolchainUninstallCmd)
	rootCmd.AddCommand(toolchainCmd)
}
