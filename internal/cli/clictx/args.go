package clictx

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/mattn/go-isatty"
)

const jsonFlagName = "json"

// ParseJSONModeFlag reports whether arg is a --json flag and, if so, the
// value it sets. Commands with DisableFlagParsing use it to honor --json.
func ParseJSONModeFlag(arg string) (bool, bool, error) {
	if arg == "--"+jsonFlagName {
		return true, true, nil
	}
	prefix := "--" + jsonFlagName + "="
	if !strings.HasPrefix(arg, prefix) {
		return false, false, nil
	}
	value, err := strconv.ParseBool(strings.TrimPrefix(arg, prefix))
	if err != nil {
		return true, false, fmt.Errorf("invalid --%s value %q", jsonFlagName, strings.TrimPrefix(arg, prefix))
	}
	return true, value, nil
}

// ApplyJSONModeFlag applies arg to the renderer when it is a --json flag.
func (c *Context) ApplyJSONModeFlag(arg string) (bool, error) {
	matched, value, err := ParseJSONModeFlag(arg)
	if err != nil || !matched {
		return matched, err
	}
	c.Output.SetJSONMode(value)
	return true, nil
}

// StripJSONModeFlagPrefix consumes leading --json flags and, when
// allowPlusToolchain is set, an optional +toolchain selector. It applies JSON
// mode as a side effect and returns the extracted toolchain (empty if none)
// together with the remaining args.
//
// A "--" terminator stops all interpretation: everything after it is returned
// verbatim. This lets a literal command that begins with "+" be run via
// `cjv exec -- +cmd` without it being mistaken for a toolchain override.
func (c *Context) StripJSONModeFlagPrefix(args []string, allowPlusToolchain bool) (toolchain string, rest []string, err error) {
	for i := range args {
		arg := args[i]
		if arg == "--" {
			rest = append(rest, args[i+1:]...)
			return toolchain, rest, nil
		}
		matched, ferr := c.ApplyJSONModeFlag(arg)
		if ferr != nil {
			return "", nil, ferr
		}
		if matched {
			continue
		}
		if allowPlusToolchain && toolchain == "" && strings.HasPrefix(arg, "+") && len(arg) > 1 {
			toolchain = arg[1:]
			continue
		}
		// First token that is neither a recognized flag nor the +toolchain
		// selector: the rest is the command and its arguments, verbatim.
		rest = append(rest, args[i:]...)
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
