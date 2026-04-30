package cli

import (
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

func runWhich(cmd *cobra.Command, args []string) error {
	active, err := resolve.Active(cmd.Context(), "")
	if err != nil {
		return err
	}

	if len(args) == 0 {
		cmd.Println(active.Dir)
		return nil
	}

	toolPath, err := proxy.ResolveInstalledToolBinary(active.Dir, args[0])
	if err != nil {
		return err
	}

	cmd.Println(toolPath)
	return nil
}

func init() {
	rootCmd.AddCommand(whichCmd)
}
