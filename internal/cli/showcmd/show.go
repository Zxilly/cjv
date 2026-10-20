// Package showcmd implements the read-only inspection commands: show,
// check, and doc.
package showcmd

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/clictx"
	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

// commands holds the flags of the inspection commands for one invocation.
type commands struct {
	ctx          *clictx.Context
	docPath      bool
	docToolchain string
	openURL      func(string) error
}

func newCommands(ctx *clictx.Context) *commands {
	return &commands{ctx: ctx, openURL: openURL}
}

// Register adds show, check, and doc to root.
func Register(root *cobra.Command, ctx *clictx.Context) {
	c := newCommands(ctx)
	root.AddCommand(c.showCommand(), c.checkCommand(), c.docCommand())
}

type showHomeResult struct {
	Home   string            `json:"home"`
	Source config.HomeSource `json:"source"`
}

func (r showHomeResult) Text() string {
	return fmt.Sprintf("%s (source: %s)", r.Home, r.Source)
}

// ActiveResult describes the active toolchain and where it was selected.
type ActiveResult struct {
	Name      string `json:"name"`
	Source    string `json:"source"`
	Installed bool   `json:"installed"`
}

func (r ActiveResult) Text() string {
	name := r.Name
	if !r.Installed {
		name = name + " (not installed)"
	}
	return i18n.T("ActiveToolchain", i18n.MsgData{
		"Name":   name,
		"Source": r.Source,
	})
}

type installedEntry struct {
	Name       string   `json:"name"`
	Components []string `json:"components"`
}

// InstalledResult lists installed toolchains with their components.
type InstalledResult struct {
	Toolchains []installedEntry `json:"toolchains"`
}

func (r InstalledResult) Text() string {
	if len(r.Toolchains) == 0 {
		return i18n.T("NoToolchainsInstalled", nil)
	}
	var b strings.Builder
	b.WriteString(i18n.TP("InstalledToolchains", i18n.MsgData{
		"Count": strconv.Itoa(len(r.Toolchains)),
	}, len(r.Toolchains)))
	b.WriteByte('\n')
	for _, e := range r.Toolchains {
		fmt.Fprintf(&b, "  %s\n", e.Name)
		if len(e.Components) > 0 {
			fmt.Fprintf(&b, "    %s %s\n",
				i18n.TP("InstalledComponents",
					i18n.MsgData{"Count": strconv.Itoa(len(e.Components))},
					len(e.Components)),
				strings.Join(e.Components, ", "))
		}
	}
	return b.String()
}

type showDefaultResult struct {
	Active      *ActiveResult   `json:"active,omitempty"`
	DefaultHost *string         `json:"default_host,omitempty"`
	Installed   InstalledResult `json:"installed"`
}

func (r showDefaultResult) Text() string {
	var b strings.Builder
	if r.Active != nil {
		b.WriteString(r.Active.Text())
		b.WriteByte('\n')
	}
	if r.DefaultHost != nil {
		host := *r.DefaultHost
		if host == "" {
			host = i18n.T("ShowDefaultHostAuto", nil)
		}
		b.WriteString(i18n.T("ShowDefaultHost", i18n.MsgData{"Host": host}))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.WriteString(r.Installed.Text())
	return b.String()
}

// GatherActive reports the active toolchain, honoring the +toolchain
// selector. A selected but missing toolchain is reported as not installed.
func GatherActive(ctx context.Context, selector string) (*ActiveResult, error) {
	_, name, source, err := toolchain.InspectActive(ctx, selector)
	provenance := source.String()
	if selector != "" {
		provenance = "command line"
	}
	if err != nil {
		if notInstalled, ok := errors.AsType[*cjverr.ToolchainNotInstalledError](err); ok {
			return &ActiveResult{
				Name:   notInstalled.Name,
				Source: provenance,
			}, nil
		}
		return nil, err
	}
	return &ActiveResult{Name: name, Source: provenance, Installed: true}, nil
}

// GatherInstalled lists installed toolchains after recovering the home.
func GatherInstalled() (InstalledResult, error) {
	if err := toolchain.RecoverHome(); err != nil {
		return InstalledResult{}, err
	}
	installed, err := toolchain.ListInstalled()
	if err != nil {
		return InstalledResult{}, err
	}
	if len(installed) == 0 {
		return InstalledResult{}, nil
	}
	tcRoot, err := config.ToolchainsDir()
	if err != nil {
		return InstalledResult{}, err
	}
	entries := make([]installedEntry, 0, len(installed))
	for _, name := range installed {
		entry := installedEntry{Name: name, Components: []string{}}
		if comps, err := componentlib.ListInstalled(filepath.Join(tcRoot, name)); err == nil && len(comps) > 0 {
			parts := make([]string, len(comps))
			for i, c := range comps {
				parts[i] = string(c)
			}
			entry.Components = parts
		}
		entries = append(entries, entry)
	}
	return InstalledResult{Toolchains: entries}, nil
}

func (c *commands) runShowDefault(cmd *cobra.Command, args []string) error {
	active, activeErr := GatherActive(cmd.Context(), c.ctx.Selector)
	if activeErr != nil {
		if _, ok := errors.AsType[*cjverr.NoToolchainConfiguredError](activeErr); ok {
			// Surface the informative message on stderr for humans, but do
			// not error out — the rest of the report is still useful.
			c.ctx.Output.Note(cmd.ErrOrStderr(), activeErr.Error())
			active = nil
		} else {
			return activeErr
		}
	}

	var defaultHost *string
	if sf, err := config.DefaultSettingsFile(); err == nil {
		if settings, err := sf.Load(); err == nil && settings != nil {
			host := settings.DefaultHost
			defaultHost = &host
		}
	}

	installed, err := GatherInstalled()
	if err != nil {
		return err
	}

	return c.ctx.Output.RenderTo(clictx.Out(cmd), showDefaultResult{
		Active:      active,
		DefaultHost: defaultHost,
		Installed:   installed,
	})
}

func (c *commands) runShowActive(cmd *cobra.Command, args []string) error {
	active, err := GatherActive(cmd.Context(), c.ctx.Selector)
	if err != nil {
		return err
	}
	return c.ctx.Output.RenderTo(clictx.Out(cmd), *active)
}

// RunShowInstalled renders the installed toolchains; `toolchain list`
// shares it with `show installed`.
func RunShowInstalled(ctx *clictx.Context, cmd *cobra.Command) error {
	r, err := GatherInstalled()
	if err != nil {
		return err
	}
	return ctx.Output.RenderTo(clictx.Out(cmd), r)
}

func (c *commands) showCommand() *cobra.Command {
	show := &cobra.Command{
		Use:   "show",
		Short: i18n.T("ShowCmdShort", nil),
		RunE:  c.runShowDefault,
	}
	active := &cobra.Command{
		Use:   "active",
		Short: i18n.T("ShowActiveShort", nil),
		RunE:  c.runShowActive,
	}
	installed := &cobra.Command{
		Use:   "installed",
		Short: i18n.T("ShowInstalledShort", nil),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunShowInstalled(c.ctx, cmd)
		},
	}
	home := &cobra.Command{
		Use:   "home",
		Short: i18n.T("ShowHomeShort", nil),
		RunE: func(cmd *cobra.Command, args []string) error {
			home, src, err := config.ResolveHomeWithSource()
			if err != nil {
				return err
			}
			return c.ctx.Output.RenderTo(clictx.Out(cmd), showHomeResult{Home: home, Source: src})
		},
	}
	show.AddCommand(active, installed, home)
	return show
}
