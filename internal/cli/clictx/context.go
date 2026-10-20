// Package clictx holds the per-invocation state shared by cjv's command
// groups: the output renderer, the +toolchain selector, and the root command
// whose writer progress reports to. Each cli.Execute creates one Context;
// command packages keep their own flags and read shared state from here.
package clictx

import (
	"io"
	"os"

	"github.com/Zxilly/cjv/internal/cli/output"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

// Context is the state one CLI invocation shares across its commands.
type Context struct {
	Output    *output.Renderer
	Root      *cobra.Command
	Version   string
	UpdateURL string
	// Selector is the toolchain named by a leading +toolchain argument.
	Selector string
}

// New returns a Context with a fresh renderer and a placeholder root
// command; cli replaces Root with the real command tree.
func New(version, updateURL string) *Context {
	return &Context{
		Output:    &output.Renderer{},
		Root:      &cobra.Command{},
		Version:   version,
		UpdateURL: updateURL,
	}
}

// Progress is the adapter this invocation's operations report to: the
// renderer picks text on the root command writer or nothing in JSON mode.
func (c *Context) Progress() progress.Sink {
	return c.Output.Progress(c.Root.OutOrStdout())
}

// LifecycleOptions returns the lifecycle adapters for this invocation.
// Commands with --force or --allow-downgrade set those fields themselves.
func (c *Context) LifecycleOptions() lifecycle.Options {
	return lifecycle.Options{
		Progress:      c.Progress(),
		ConfigurePath: true,
	}
}

// Toolchain returns flag when set, otherwise the +toolchain selector.
func (c *Context) Toolchain(flag string) string {
	if flag != "" {
		return flag
	}
	return c.Selector
}

// SelectedToolchain names the toolchain a command should act on: flag, the
// +toolchain selector, or the active toolchain. A configured but missing
// toolchain is still returned, because it is a valid install selection.
func (c *Context) SelectedToolchain(flag string) (string, error) {
	if err := toolchain.RecoverHome(); err != nil {
		return "", err
	}
	if name := c.Toolchain(flag); name != "" {
		return name, nil
	}
	_, name, _, err := toolchain.ResolveActiveToolchain()
	if name != "" {
		return name, nil
	}
	return "", err
}

// ResolveToolchainArg locates an installed toolchain, falling back to the
// active toolchain when flagValue is empty.
func ResolveToolchainArg(flagValue string) (string, toolchain.ToolchainName, error) {
	if err := toolchain.RecoverHome(); err != nil {
		return "", toolchain.ToolchainName{}, err
	}
	if flagValue != "" {
		parsed, err := toolchain.ParseToolchainName(flagValue)
		if err != nil {
			return "", toolchain.ToolchainName{}, err
		}
		dir, err := toolchain.FindInstalled(parsed)
		if err != nil {
			return "", toolchain.ToolchainName{}, err
		}
		actual, err := toolchain.InstalledRelease(dir)
		if err != nil {
			return "", toolchain.ToolchainName{}, err
		}
		return dir, actual, nil
	}
	dir, _, _, err := toolchain.ResolveActiveToolchain()
	if err != nil {
		return "", toolchain.ToolchainName{}, err
	}
	parsed, err := toolchain.InstalledRelease(dir)
	if err != nil {
		return "", toolchain.ToolchainName{}, err
	}
	return dir, parsed, nil
}

// Out returns the writer a command renders its result to.
func Out(cmd *cobra.Command) io.Writer {
	if cmd == nil {
		return os.Stdout
	}
	return cmd.OutOrStdout()
}
