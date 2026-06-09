package cli

import (
	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/output"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/resolve"
	"github.com/spf13/cobra"
)

var whichCmd = &cobra.Command{
	Use:   "which [command]",
	Short: i18n.T("WhichCmdShort", nil),
	Long:  i18n.T("WhichCmdLong", nil),
	Args:  cobra.MaximumNArgs(1),
	RunE:  runWhich,
}

type whichResult struct {
	Tool      string `json:"tool,omitempty"`
	Path      string `json:"path"`
	Toolchain string `json:"toolchain"`
}

func (r whichResult) Text() string { return r.Path }

func runWhich(cmd *cobra.Command, args []string) error {
	active, err := resolve.Active(cmd.Context(), "")
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return output.RenderTo(cmdOutput(cmd), whichResult{Path: active.Dir, Toolchain: active.Name})
	}

	// Resolve through the same logic as `cjv run` so the two commands agree on
	// what is runnable: known proxy tools plus any binary under bin/ or
	// tools/bin/ (not just the fixed proxy-tool table).
	toolPath, found := resolveToolchainToolPath(active.Dir, args[0])
	if !found {
		if proxy.IsProxyTool(args[0]) {
			// A known tool that is simply absent from this toolchain — surface
			// the precise "not in toolchain" error rather than "unknown tool".
			// Use ResolveToolBinary (path only, no extra stat) since
			// resolveToolchainToolPath already confirmed it is missing.
			binary, _ := proxy.ResolveToolBinary(active.Dir, args[0])
			return &cjverr.ToolNotInToolchainError{Tool: args[0], Path: binary}
		}
		return &cjverr.UnknownToolError{Name: args[0]}
	}
	return output.RenderTo(cmdOutput(cmd), whichResult{Tool: args[0], Path: toolPath, Toolchain: active.Name})
}

func init() {
	rootCmd.AddCommand(whichCmd)
}
