package toolchaincmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/cli/output"
	"github.com/Zxilly/cjv/internal/cli/showcmd"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/toolchain"
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

// toolchainMaterializedResult is the URL and archive form of a link: the
// toolchain was downloaded or unpacked into CJV_HOME, and the install
// progress already told the user so.
type toolchainMaterializedResult struct {
	output.ProgressDriven
	Name string `json:"name"`
	Path string `json:"path"`
}

func (r toolchainLinkResult) Text() string {
	return color.GreenString(i18n.T("ToolchainLinked", i18n.MsgData{
		"Name": r.Name,
		"Path": r.Path,
	}))
}

// link holds the flags of `cjv toolchain link` for one invocation.
type link struct {
	ctx    *clictx.Context
	sha256 string
	force  bool
	noStdx bool
}

// renderMaterialized renders the result of a URL or archive link, whose
// toolchain now lives under the toolchains directory.
func (c *link) renderMaterialized(cmd *cobra.Command, name string) error {
	tcDir, err := config.ToolchainsDir()
	if err != nil {
		return err
	}
	return c.ctx.Output.RenderTo(clictx.Out(cmd), toolchainMaterializedResult{Name: name, Path: filepath.Join(tcDir, name)})
}

func (c *link) run(cmd *cobra.Command, args []string) error {
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

	name = parsed.String()

	// URL target: download and materialize a cjv-owned toolchain.
	if isURLPath(targetPath) {
		if err := lifecycle.InstallToolchainFromURL(cmd.Context(), name, targetPath, c.sha256, c.force, c.noStdx, c.ctx.LifecycleOptions()); err != nil {
			return err
		}
		return c.renderMaterialized(cmd, name)
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
		if err := lifecycle.InstallToolchainFromZip(cmd.Context(), name, absPath, c.sha256, c.force, c.noStdx, c.ctx.LifecycleOptions()); err != nil {
			return err
		}
		return c.renderMaterialized(cmd, name)
	}

	// Directory link: the archive/URL-only flags must not be used here. Reject
	// explicit use rather than silently ignoring them.
	for _, f := range []string{"sha256", "force", "no-stdx"} {
		if cmd.Flags().Changed(f) {
			return errors.New(i18n.T("LinkFlagURLOnly", i18n.MsgData{"Flag": f}))
		}
	}

	if err := lifecycle.LinkToolchainDir(name, absPath); err != nil {
		return err
	}
	return c.ctx.Output.RenderTo(clictx.Out(cmd), toolchainLinkResult{Name: name, Path: absPath})
}

func (c *link) command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link <name> <path>",
		Short: i18n.T("ToolchainLinkShort", nil),
		Args:  cobra.ExactArgs(2),
		RunE:  c.run,
	}
	cmd.Flags().StringVar(&c.sha256, "sha256", "", i18n.T("LinkFlagSHA256", nil))
	cmd.Flags().BoolVar(&c.force, "force", false, i18n.T("LinkFlagForce", nil))
	cmd.Flags().BoolVar(&c.noStdx, "no-stdx", false, i18n.T("LinkFlagNoStdx", nil))
	return cmd
}

func toolchainCommand(ctx *clictx.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "toolchain",
		Short: i18n.T("ToolchainCmdShort", nil),
	}
	list := &cobra.Command{
		Use:   "list",
		Short: i18n.T("ToolchainListShort", nil),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showcmd.RunShowInstalled(ctx, cmd)
		},
	}
	cmd.AddCommand(
		list,
		(&link{ctx: ctx}).command(),
		newUninstall(ctx).command("uninstall <name>...", i18n.T("ToolchainUninstallShort", nil)),
		(&listRemote{ctx: ctx}).command(),
	)
	return cmd
}
