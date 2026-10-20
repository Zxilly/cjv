// Package componentcmd implements `cjv component`.
package componentcmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Zxilly/cjv/internal/cli/clictx"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/output"
	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/progress"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// commands holds the flags of `cjv component` for one invocation.
type commands struct {
	ctx               *clictx.Context
	toolchain         string
	target            string
	addForce          bool
	linkForce         bool
	listInstalledOnly bool
	listQuiet         bool
	link              func(componentlib.Roots, componentlib.Name, string, bool) (string, error)
}

func newCommands(ctx *clictx.Context) *commands {
	return &commands{ctx: ctx, link: componentlib.Link}
}

// Register adds `component` and its subcommands to root.
func Register(root *cobra.Command, ctx *clictx.Context) {
	root.AddCommand(newCommands(ctx).command())
}

func (c *commands) command() *cobra.Command {
	component := &cobra.Command{
		Use:   "component",
		Short: i18n.T("ComponentSubcmdShort", nil),
		Args:  cobra.NoArgs,
	}
	add := &cobra.Command{
		Use:   "add <name>...",
		Short: i18n.T("ComponentAddShort", nil),
		Args:  cobra.MinimumNArgs(1),
		RunE:  c.runAdd,
	}
	remove := &cobra.Command{
		Use:     "remove <name>...",
		Aliases: []string{"uninstall", "rm", "delete", "del"},
		Short:   i18n.T("ComponentRemoveShort", nil),
		Args:    cobra.MinimumNArgs(1),
		RunE:    c.runRemove,
	}
	list := &cobra.Command{
		Use:   "list",
		Short: i18n.T("ComponentListShort", nil),
		Args:  cobra.NoArgs,
		RunE:  c.runList,
	}
	link := &cobra.Command{
		Use:   "link <name> <path>",
		Short: i18n.T("ComponentLinkShort", nil),
		Args:  cobra.ExactArgs(2),
		RunE:  c.runLink,
	}

	component.PersistentFlags().StringVar(&c.toolchain, "toolchain", "", i18n.T("ComponentFlagToolchain", nil))
	component.PersistentFlags().StringVar(&c.target, "target", "", "Cross SDK environment whose components to manage")
	add.Flags().BoolVar(&c.addForce, "force", false, i18n.T("InstallFlagForce", nil))
	list.Flags().BoolVar(&c.listInstalledOnly, "installed", false, i18n.T("ComponentFlagInstalled", nil))
	list.Flags().BoolVarP(&c.listQuiet, "quiet", "q", false, i18n.T("ComponentFlagQuiet", nil))
	link.Flags().BoolVar(&c.linkForce, "force", false, i18n.T("ComponentLinkFlagForce", nil))

	component.AddCommand(add, remove, list, link)
	return component
}

type componentEntry struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
}

type componentListResult struct {
	Toolchain  string           `json:"toolchain"`
	Components []componentEntry `json:"components"`
}

// componentListView attaches presentation flags to a componentListResult so
// they can affect Text() rendering without polluting the JSON data shape.
type componentListView struct {
	componentListResult
	quiet         bool
	installedOnly bool
}

func (v componentListView) Text() string {
	if v.quiet {
		var b strings.Builder
		for _, c := range v.Components {
			b.WriteString(c.Name)
			b.WriteByte('\n')
		}
		return b.String()
	}
	if len(v.Components) == 0 && v.installedOnly {
		return i18n.T("NoComponentsInstalled", nil)
	}
	var b strings.Builder
	for _, c := range v.Components {
		label := i18n.T("AvailableComponents", nil)
		if c.Installed {
			label = i18n.T("StatusInstalled", nil)
		}
		fmt.Fprintf(&b, "%s (%s)\n", c.Name, label)
	}
	return b.String()
}

func (v componentListView) JSONValue() any {
	return v.componentListResult
}

type componentRemovedEntry struct {
	Toolchain string `json:"toolchain"`
	Component string `json:"component"`
}

type componentRemoveResult struct {
	Removed []componentRemovedEntry `json:"removed"`
}

type componentAddResult struct {
	output.ProgressDriven
	Toolchain  string              `json:"toolchain"`
	Components []componentlib.Name `json:"components"`
	Forced     bool                `json:"forced"`
}

func (r componentRemoveResult) Text() string {
	var b strings.Builder
	for _, e := range r.Removed {
		b.WriteString(color.GreenString(i18n.T("ComponentRemoved", i18n.MsgData{
			"Toolchain": e.Toolchain,
			"Component": e.Component,
		})))
		b.WriteByte('\n')
	}
	return b.String()
}

type componentLinkResult struct {
	Toolchain string `json:"toolchain"`
	Component string `json:"component"`
	Path      string `json:"path"`
}

func (r componentLinkResult) Text() string {
	return color.GreenString(i18n.T("ComponentLinked", i18n.MsgData{
		"Component": r.Component,
		"Path":      r.Path,
		"Toolchain": r.Toolchain,
	}))
}

func (c *commands) resolveToolchain() (string, toolchain.ToolchainName, error) {
	dir, release, err := clictx.ResolveToolchainArg(c.ctx.Toolchain(c.toolchain))
	if err == nil {
		managed, rootErr := config.ToolchainDirFor(filepath.Base(dir))
		if rootErr != nil {
			return "", release, rootErr
		}
		if config.NormalizePath(managed) != config.NormalizePath(dir) {
			return "", release, fmt.Errorf("components of external toolchain %s must be managed by its owner", dir)
		}
	}
	if err != nil || c.target == "" {
		return dir, release, err
	}
	host, err := toolchain.ReadHostTargets(dir)
	if err != nil {
		return "", release, err
	}
	return host.FindTarget(c.target)
}

func (c *commands) runAdd(cmd *cobra.Command, args []string) error {
	tcDir, tcName, err := c.resolveToolchain()
	if err != nil {
		return err
	}
	if tcName.IsCustom() {
		return &cjverr.ComponentRequiresHostError{Component: args[0]}
	}
	if err := lifecycle.InstallComponents(cmd.Context(), filepath.Base(tcDir), args, c.addForce, c.ctx.LifecycleOptions()); err != nil {
		return err
	}
	components, err := componentlib.NormalizeList(args)
	if err != nil {
		return err
	}
	return c.ctx.Output.RenderTo(clictx.Out(cmd), componentAddResult{
		Toolchain:  filepath.Base(tcDir),
		Components: components,
		Forced:     c.addForce,
	})
}

func (c *commands) runRemove(cmd *cobra.Command, args []string) error {
	parsed, parseErrs := parseComponentRemoveArgs(args)
	if len(parsed) == 0 {
		return errors.Join(parseErrs...)
	}

	tcDir, _, err := c.resolveToolchain()
	if err != nil {
		return err
	}
	toolchainName := filepath.Base(tcDir)

	lock, err := toolchain.LockHome(cmd.Context())
	if err != nil {
		return err
	}
	defer lock.Close() //nolint:errcheck
	if err := lock.Recover(); err != nil {
		return err
	}

	roots, err := componentlib.RootsFor(toolchainName)
	if err != nil {
		return err
	}

	var removeErrs []error
	removeErrs = append(removeErrs, parseErrs...)
	result := componentRemoveResult{}
	sink := c.ctx.Progress()
	for _, c := range parsed {
		if !componentlib.IsInstalled(tcDir, c) {
			removeErrs = append(removeErrs, &cjverr.ComponentNotInstalledError{
				Toolchain: toolchainName,
				Component: string(c),
			})
			continue
		}
		sink.Report(progress.Event{Kind: progress.RemovingComponent, Component: string(c)})
		if err := componentlib.Remove(roots, c); err != nil {
			removeErrs = append(removeErrs, err)
			continue
		}
		result.Removed = append(result.Removed, componentRemovedEntry{
			Toolchain: toolchainName,
			Component: string(c),
		})
	}
	// Emit what was actually removed before surfacing any error so a partial
	// removal is visible to the caller.
	return c.ctx.Output.RenderOutcome(clictx.Out(cmd), result, errors.Join(removeErrs...))
}

func (c *commands) runLink(cmd *cobra.Command, args []string) error {
	name, err := componentlib.ParseName(args[0])
	if err != nil {
		return err
	}

	tcDir, _, err := c.resolveToolchain()
	if err != nil {
		return err
	}
	toolchainName := filepath.Base(tcDir)

	lock, err := toolchain.LockHome(cmd.Context())
	if err != nil {
		return err
	}
	defer lock.Close() //nolint:errcheck
	if err := lock.Recover(); err != nil {
		return err
	}

	roots, err := componentlib.RootsFor(toolchainName)
	if err != nil {
		return err
	}

	c.ctx.Progress().Report(progress.Event{Kind: progress.InstallingComponent, Component: string(name)})

	absPath, err := c.link(roots, name, args[1], c.linkForce)
	if err != nil {
		return err
	}

	return c.ctx.Output.RenderTo(clictx.Out(cmd), componentLinkResult{
		Toolchain: toolchainName,
		Component: string(name),
		Path:      absPath,
	})
}

func (c *commands) runList(cmd *cobra.Command, args []string) error {
	tcDir, tcName, err := c.resolveToolchain()
	if err != nil {
		return err
	}

	installed, err := componentlib.ListInstalled(tcDir)
	if err != nil {
		return err
	}

	var available []componentlib.Name
	if !tcName.IsCustom() {
		tuple := tcName.Target
		if tuple == "" {
			record, err := toolchain.ReadInstallation(tcDir)
			if err != nil {
				return err
			}
			tuple = record.Tuple
		}
		if tuple == "" {
			_, settings, err := config.LoadDefaultSettings()
			if err != nil {
				return err
			}
			tuple, err = sdktarget.CurrentHostTuple(settings.DefaultHost)
			if err != nil {
				return err
			}
		}
		available = componentlib.AvailableComponents(tcName, tuple)
	}

	// Initialize Components to a non-nil empty slice so `--json` emits [] rather
	// than null for the empty state, keeping the array field's shape stable.
	result := componentListResult{Toolchain: filepath.Base(tcDir), Components: []componentEntry{}}
	for _, n := range installed {
		result.Components = append(result.Components, componentEntry{Name: string(n), Installed: true})
	}
	if !c.listInstalledOnly {
		for _, n := range available {
			if slices.Contains(installed, n) {
				continue
			}
			result.Components = append(result.Components, componentEntry{Name: string(n), Installed: false})
		}
	}
	return c.ctx.Output.RenderTo(clictx.Out(cmd), componentListView{
		componentListResult: result,
		quiet:               c.listQuiet,
		installedOnly:       c.listInstalledOnly,
	})
}

func parseComponentRemoveArgs(args []string) ([]componentlib.Name, []error) {
	var out []componentlib.Name
	var errs []error
	seen := make(map[componentlib.Name]bool)
	seenUnknown := make(map[string]bool)
	for _, raw := range args {
		for part := range strings.SplitSeq(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			n, err := componentlib.ParseName(part)
			if err != nil {
				if !seenUnknown[part] {
					seenUnknown[part] = true
					errs = append(errs, err)
				}
				continue
			}
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	return out, errs
}
