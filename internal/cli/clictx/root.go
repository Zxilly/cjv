package clictx

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

// ConfigureRoot makes root this invocation's root command and adds the
// global output flags every command inherits.
func (c *Context) ConfigureRoot(root *cobra.Command) {
	c.Root = root
	var quiet, verbose bool
	root.PersistentFlags().BoolVar(&c.Output.JSON, "json", false, i18n.T("RootFlagJSON", nil))
	root.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Disable progress output")
	root.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable debug logging")
	root.MarkFlagsMutuallyExclusive("quiet", "verbose")
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		c.Output.Quiet = quiet
		if os.Getenv("CJV_LOG") == "" && verbose {
			slog.SetDefault(slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{Level: slog.LevelDebug})))
		}
		return nil
	}
}

// Execute runs the root command with args and renders its error once.
func (c *Context) Execute(args []string) error {
	previousLogger := slog.Default()
	defer slog.SetDefault(previousLogger)
	if selector, rest, present := toolchain.SplitPlusSelector(args); present {
		if selector == "" {
			return c.Output.RenderErrorTo(c.Root.OutOrStdout(), c.Root.ErrOrStderr(), fmt.Errorf("toolchain name cannot be empty after '+'"))
		}
		c.Selector = selector
		args = rest
	}
	// Unknown commands fail during Cobra's lookup, before flag parsing. Honor
	// leading output flags for those errors without interpreting child arguments.
	for _, arg := range args {
		matched, enabled, err := ParseJSONModeFlag(arg)
		if err != nil || !matched {
			break
		}
		c.Output.SetJSONMode(enabled)
	}
	c.Root.SetArgs(args)
	err := c.Root.Execute()
	return c.Output.RenderErrorTo(c.Root.OutOrStdout(), c.Root.ErrOrStderr(), err)
}
