package cli

import (
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

var whichCmd = &cobra.Command{
	Use:   "which <command>",
	Short: "Show the path of an SDK tool for the active toolchain",
	Args:  cobra.ExactArgs(1),
	RunE:  runWhich,
}

func runWhich(cmd *cobra.Command, args []string) error {
	toolName := args[0]

	tcDir, _, _, err := toolchain.ResolveActiveToolchain()
	if err != nil {
		return err
	}

	toolPath, err := proxy.ResolveInstalledToolBinary(tcDir, toolName)
	if err != nil {
		return err
	}

	cmd.Println(toolPath)
	return nil
}

func init() {
	rootCmd.AddCommand(whichCmd)
}
