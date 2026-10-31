package clictx

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/mattn/go-isatty"
)

// ApplyGlobalFlag consumes one recognized global option and its value. It is
// shared by root-prefix parsing and commands that forward child arguments.
// A zero count means the token belongs to the command or its positional args.
func (c *Context) ApplyGlobalFlag(args []string) (int, error) {
	if len(args) == 0 {
		return 0, nil
	}
	name, value, hasValue := strings.Cut(args[0], "=")
	if name == "-q" {
		name = "--quiet"
	}
	if name == "--config" {
		count := 1
		if !hasValue {
			if len(args) < 2 {
				return 0, errors.New(i18n.T("RootConfigMissing", nil))
			}
			value, count = args[1], 2
		}
		if strings.TrimSpace(value) == "" {
			return 0, errors.New(i18n.T("RootConfigEmpty", nil))
		}
		c.ConfigFile = value
		return count, c.setGlobalFlag("config", value)
	}
	var target *bool
	switch name {
	case "--json":
		target = &c.Output.JSON
	case "--quiet":
		target = &c.Quiet
	case "--verbose":
		target = &c.Verbose
	case "--offline":
		target = &c.Offline
	case "--no-auto-install":
		target = &c.NoAutoInstall
	default:
		return 0, nil
	}
	if !hasValue {
		value = "true"
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return 0, fmt.Errorf(i18n.T("RootInvalidFlagValue", nil), name, value)
	}
	*target = enabled
	return 1, c.setGlobalFlag(strings.TrimPrefix(name, "--"), value)
}

func (c *Context) setGlobalFlag(name, value string) error {
	if c.Root.PersistentFlags().Lookup(name) == nil {
		return nil // Direct command tests may not configure a root flag set.
	}
	return c.Root.PersistentFlags().Set(name, value)
}

// StripGlobalFlagPrefix consumes leading global flags and, when
// allowPlusToolchain is set, an optional +toolchain selector. It applies the
// global options and returns the extracted toolchain (empty if none)
// together with the remaining args.
//
// A "--" terminator stops all interpretation: everything after it is returned
// verbatim. This lets a literal command that begins with "+" be run via
// `cjv exec -- +cmd` without it being mistaken for a toolchain override.
func (c *Context) StripGlobalFlagPrefix(args []string, allowPlusToolchain bool) (toolchain string, rest []string, err error) {
	for len(args) > 0 {
		arg := args[0]
		if arg == "--" {
			rest = append(rest, args[1:]...)
			return toolchain, rest, nil
		}
		consumed, ferr := c.ApplyGlobalFlag(args)
		if ferr != nil {
			return "", nil, ferr
		}
		if consumed > 0 {
			args = args[consumed:]
			continue
		}
		if allowPlusToolchain && toolchain == "" && strings.HasPrefix(arg, "+") && len(arg) > 1 {
			toolchain = arg[1:]
			args = args[1:]
			continue
		}
		// First token that is neither a recognized flag nor the +toolchain
		// selector: the rest is the command and its arguments, verbatim.
		rest = append(rest, args...)
		return toolchain, rest, nil
	}
	return toolchain, rest, nil
}

// ExtractPlusToolchain extracts an optional +toolchain prefix from args
// (used by commands that let cobra parse flags). A bare "+" is ignored.
func ExtractPlusToolchain(args []string) (string, []string) {
	if tc, rest, present := toolchain.SplitPlusSelector(args); present && tc != "" {
		return tc, rest
	}
	return "", args
}

// StdinIsTerminal reports whether stdin is interactive, so prompts can run.
func StdinIsTerminal() bool {
	fd := os.Stdin.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
