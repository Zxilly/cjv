package runcmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/resolve"
	"github.com/spf13/cobra"
)

func (c *commands) runRun(cmd *cobra.Command, args []string) error {
	defer c.ctx.RestoreEnvironment()

	// Manual flag parsing since DisableFlagParsing is true.
	// Consume all flags before the positional <toolchain> argument.
	install := false
	installSpecified := false
flagLoop:
	for len(args) > 0 {
		consumed, err := c.ctx.ApplyGlobalFlag(args)
		if err != nil {
			return err
		}
		if consumed > 0 {
			args = args[consumed:]
			continue
		}
		if value, found := strings.CutPrefix(args[0], "--install="); found {
			enabled, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf(i18n.T("RootInvalidFlagValue", nil), "--install", value)
			}
			install = enabled
			installSpecified = true
			args = args[1:]
			continue
		}
		switch args[0] {
		case "--install":
			install = true
			installSpecified = true
			args = args[1:]
		case "--help", "-h":
			return cmd.Help()
		case "--":
			args = args[1:]
			break flagLoop
		default:
			break flagLoop
		}
	}
	if install && c.ctx.NoAutoInstall {
		return errors.New(i18n.T("RunInstallConflict", nil))
	}
	if err := c.ctx.ApplyInvocation(cmd); err != nil {
		return err
	}
	ctx := cmd.Context()
	if installSpecified {
		ctx = resolve.WithAutoInstall(ctx, install)
	}
	if c.ctx.Output.IsJSON() {
		return &cjverr.UnsupportedForJSONError{Command: "run"}
	}

	if len(args) < 2 {
		_ = cmd.Help()
		return errors.New(i18n.T("RunMissingCommand", nil))
	}

	return runRuntimeCommand(ctx, args[0], args[1], args[2:])
}

// resolveToolchainToolPath resolves command to an absolute path within tcDir,
// first consulting the known proxy-tool table, then scanning bin/ and
// tools/bin/. The bool reports whether the tool was found inside the toolchain;
// when false the returned path is the bare command name.
func resolveToolchainToolPath(tcDir, command string) (string, bool) {
	return env.ResolveToolPath(tcDir, command)
}
