package cli

import (
	"github.com/Zxilly/cjv/internal/cli/output"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/resolve"
	"github.com/spf13/cobra"
)

var whichCmd = &cobra.Command{
	Use:   "which [command]",
	Short: "Show the path of an SDK tool for the active toolchain",
	Long:  "Show the path of an SDK tool for the active toolchain.\nIf no command is given, print the toolchain root directory.",
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

	toolPath, err := proxy.ResolveInstalledToolBinary(active.Dir, args[0])
	if err != nil {
		return err
	}
	return output.RenderTo(cmdOutput(cmd), whichResult{Tool: args[0], Path: toolPath, Toolchain: active.Name})
}

func init() {
	rootCmd.AddCommand(whichCmd)
}
