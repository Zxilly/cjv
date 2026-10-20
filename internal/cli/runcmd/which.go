package runcmd

import (
	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/resolve"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/spf13/cobra"
)

type whichResult struct {
	Tool      string `json:"tool,omitempty"`
	Path      string `json:"path"`
	Toolchain string `json:"toolchain"`
}

func (r whichResult) Text() string { return r.Path }

func (c *commands) runWhich(cmd *cobra.Command, args []string) error {
	active, err := resolve.Active(cmd.Context(), c.ctx.Toolchain(c.whichToolchain))
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return c.ctx.Output.RenderTo(clictx.Out(cmd), whichResult{Path: active.Dir, Toolchain: active.Name})
	}

	// Resolve through the same logic as `cjv run` so the two commands agree on
	// what is runnable: known proxy tools plus any binary under bin/ or
	// tools/bin/ (not just the fixed proxy-tool table).
	toolPath, found := resolveToolchainToolPath(active.Dir, args[0])
	if !found {
		if sdktools.IsProxyTool(args[0]) {
			// A known tool that is simply absent from this toolchain — surface
			// the precise "not in toolchain" error rather than "unknown tool".
			// Use ResolveToolBinary (path only, no extra stat) since
			// resolveToolchainToolPath already confirmed it is missing.
			binary, _ := sdktools.ResolveToolBinary(active.Dir, args[0])
			return &cjverr.ToolNotInToolchainError{Tool: args[0], Path: binary}
		}
		return &cjverr.UnknownToolError{Name: args[0]}
	}
	return c.ctx.Output.RenderTo(clictx.Out(cmd), whichResult{Tool: args[0], Path: toolPath, Toolchain: active.Name})
}
