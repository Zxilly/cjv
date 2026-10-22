package clictx

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/resolve"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

// ConfigureRoot makes root this invocation's root command and adds the
// global output flags every command inherits.
func (c *Context) ConfigureRoot(root *cobra.Command) {
	c.Root = root
	root.PersistentFlags().BoolVar(&c.Output.JSON, "json", false, i18n.T("RootFlagJSON", nil))
	root.PersistentFlags().BoolVarP(&c.Quiet, "quiet", "q", false, "Disable progress output")
	root.PersistentFlags().BoolVar(&c.Verbose, "verbose", false, "Enable debug logging")
	root.PersistentFlags().BoolVar(&c.Offline, "offline", false, "Use only local files and cached downloads; disable network access")
	root.PersistentFlags().StringVar(&c.ConfigFile, "config", "", "Use the specified user settings file (or CJV_CONFIG_FILE)")
	root.PersistentFlags().BoolVar(&c.NoAutoInstall, "no-auto-install", false, "Disable automatic installation for this invocation")
	root.MarkFlagsMutuallyExclusive("quiet", "verbose")
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// run/exec parse a global prefix themselves and must finish parsing
		// before applying policies or validating environment overrides.
		if cmd.DisableFlagParsing {
			return nil
		}
		return c.ApplyInvocation(cmd)
	}
}

// ApplyInvocation runs after all global flags have been consumed. The
// scoped environment carries these policies to nested proxy invocations.
func (c *Context) ApplyInvocation(cmd *cobra.Command) error {
	if c.Quiet && c.Verbose {
		return fmt.Errorf("--quiet and --verbose cannot be used together")
	}
	c.Output.Quiet = c.Quiet
	if os.Getenv(config.EnvLog) == "" && c.Verbose {
		slog.SetDefault(slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{Level: slog.LevelDebug})))
	}
	if c.ConfigFile != "" {
		path, err := filepath.Abs(c.ConfigFile)
		if err != nil {
			return err
		}
		if err := c.setEnvironment(config.EnvConfigFile, path); err != nil {
			return err
		}
	} else if flag := c.Root.PersistentFlags().Lookup("config"); flag != nil && flag.Changed {
		return fmt.Errorf("--config requires a non-empty path")
	}
	if c.NoAutoInstall {
		if err := c.setEnvironment(config.EnvAutoInstall, "0"); err != nil {
			return err
		}
	}
	if c.Offline {
		if err := c.setEnvironment("CJV_OFFLINE", "1"); err != nil {
			return err
		}
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = dist.WithOffline(ctx, c.Offline)
	cmd.SetContext(resolve.WithProgress(ctx, c.Output.ProgressTo(cmd.ErrOrStderr(), cmd.ErrOrStderr())))
	return nil
}

func (c *Context) setEnvironment(key, value string) error {
	if c.environment == nil {
		c.environment = make(map[string]*string)
	}
	if _, saved := c.environment[key]; !saved {
		if old, present := os.LookupEnv(key); present {
			c.environment[key] = &old
		} else {
			c.environment[key] = nil
		}
	}
	return os.Setenv(key, value)
}

// RestoreEnvironment releases invocation overrides, including after errors.
func (c *Context) RestoreEnvironment() {
	for key, value := range c.environment {
		if value == nil {
			_ = os.Unsetenv(key)
		} else {
			_ = os.Setenv(key, *value)
		}
	}
	clear(c.environment)
}

// Execute runs the root command with args and renders its error once.
func (c *Context) Execute(args []string) error {
	previousLogger := slog.Default()
	defer slog.SetDefault(previousLogger)
	defer c.RestoreEnvironment()
	for len(args) > 0 {
		if selector, rest, present := toolchain.SplitPlusSelector(args); present {
			if c.Selector != "" {
				return c.Output.RenderErrorTo(c.Root.OutOrStdout(), c.Root.ErrOrStderr(), fmt.Errorf("only one +toolchain selector is allowed"))
			}
			if selector == "" {
				return c.Output.RenderErrorTo(c.Root.OutOrStdout(), c.Root.ErrOrStderr(), fmt.Errorf("toolchain name cannot be empty after '+'"))
			}
			c.Selector = selector
			args = rest
			continue
		}
		// Unknown commands fail before Cobra parses flags. Consume only the
		// root prefix; run/exec stop parsing at their own positional boundary.
		consumed, err := c.ApplyGlobalFlag(args)
		if err != nil {
			return c.Output.RenderErrorTo(c.Root.OutOrStdout(), c.Root.ErrOrStderr(), err)
		}
		if consumed == 0 {
			break
		}
		args = args[consumed:]
	}
	c.Root.SetArgs(args)
	err := c.Root.Execute()
	return c.Output.RenderErrorTo(c.Root.OutOrStdout(), c.Root.ErrOrStderr(), err)
}
