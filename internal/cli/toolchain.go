package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/selfupdate"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/Zxilly/cjv/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// isURLPath reports whether the link target is an http(s) URL (vs a local path).
func isURLPath(s string) bool {
	l := strings.ToLower(s)
	return strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://")
}

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

func (app *application) initToolchainCommands() {
	app.toolchainCmd = &cobra.Command{
		Use:   "toolchain",
		Short: i18n.T("ToolchainCmdShort", nil),
	}
	app.toolchainListCmd = &cobra.Command{
		Use:   "list",
		Short: i18n.T("ToolchainListShort", nil),
		RunE:  app.runShowInstalled,
	}
	app.toolchainLinkCmd = &cobra.Command{
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

			// URL target: download and materialize a cjv-owned toolchain.
			if isURLPath(targetPath) {
				if err := lifecycle.InstallToolchainFromURL(cmd.Context(), name, targetPath, app.linkSHA256, app.linkForce, app.linkNoStdx, app.lifecycleOptions()); err != nil {
					return err
				}
				if !app.output.IsJSON() {
					return nil
				}
				tcDir, err := config.ToolchainsDir()
				if err != nil {
					return err
				}
				return app.output.RenderTo(cmdOutput(cmd), toolchainLinkResult{Name: name, Path: filepath.Join(tcDir, name)})
			}

			absPath, err := filepath.Abs(targetPath)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("LinkInvalidPath", nil), err)
			}
			info, err := os.Stat(absPath)
			if err != nil {
				return errors.New(i18n.T("LinkPathNotExist", i18n.MsgData{"Path": absPath}))
			}

			// A local archive file (zip/tar.gz) is extracted and materialized as a
			// cjv-owned toolchain, exactly like the URL path; --sha256/--force/--no-stdx
			// apply here too. A directory is referenced in place via a symlink, and
			// those flags do not apply to it.
			if !info.IsDir() {
				if err := lifecycle.InstallToolchainFromZip(cmd.Context(), name, absPath, app.linkSHA256, app.linkForce, app.linkNoStdx, app.lifecycleOptions()); err != nil {
					return err
				}
				if !app.output.IsJSON() {
					return nil
				}
				tcDir, err := config.ToolchainsDir()
				if err != nil {
					return err
				}
				return app.output.RenderTo(cmdOutput(cmd), toolchainLinkResult{Name: name, Path: filepath.Join(tcDir, name)})
			}

			// Directory link: the archive/URL-only flags must not be used here. Reject
			// explicit use rather than silently ignoring them.
			for _, f := range []string{"sha256", "force", "no-stdx"} {
				if cmd.Flags().Changed(f) {
					return errors.New(i18n.T("LinkFlagURLOnly", i18n.MsgData{"Flag": f}))
				}
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

			return app.output.RenderTo(cmdOutput(cmd), toolchainLinkResult{Name: name, Path: absPath})
		},
	}
	app.toolchainUninstallCmd = &cobra.Command{
		Use:   "uninstall <name>",
		Short: i18n.T("ToolchainUninstallShort", nil),
		Args:  cobra.ExactArgs(1),
		RunE:  app.runUninstall,
	}

	app.toolchainUninstallCmd.Flags().BoolVarP(&app.uninstallYes, "yes", "y", false, i18n.T("FlagSkipConfirm", nil))
	app.toolchainLinkCmd.Flags().StringVar(&app.linkSHA256, "sha256", "", i18n.T("LinkFlagSHA256", nil))
	app.toolchainLinkCmd.Flags().BoolVar(&app.linkForce, "force", false, i18n.T("LinkFlagForce", nil))
	app.toolchainLinkCmd.Flags().BoolVar(&app.linkNoStdx, "no-stdx", false, i18n.T("LinkFlagNoStdx", nil))
	app.toolchainCmd.AddCommand(app.toolchainListCmd)
	app.toolchainCmd.AddCommand(app.toolchainLinkCmd)
	app.toolchainCmd.AddCommand(app.toolchainUninstallCmd)
	app.rootCmd.AddCommand(app.toolchainCmd)

}
