package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli"
	"github.com/Zxilly/cjv/internal/cli/output"
	"github.com/Zxilly/cjv/internal/logging"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/utils"
	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	updateURL string
)

func main() {
	if code := run(); code != 0 {
		os.Exit(code)
	}
}

func run() int {
	logging.Init()

	utils.AppVersion = version

	toolName := proxy.ExtractToolName(os.Args[0])

	if proxy.IsProxyTool(toolName) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		if err := proxy.Run(ctx, toolName, os.Args[1:]); err != nil {
			if exitErr, ok := errors.AsType[*cjverr.ExitCodeError](err); ok {
				return exitErr.Code
			}
			fmt.Fprintln(os.Stderr, "cjv:", err)
			return 1
		}
		return 0
	}

	if isInitInvocation(toolName) {
		// cjv-init is designed to be double-clicked from Explorer, so disable
		// cobra's mousetrap that would otherwise abort with a "use cmd.exe" notice.
		cobra.MousetrapHelpText = ""
		os.Args = append([]string{os.Args[0], "init"}, os.Args[1:]...)
		defer utils.PauseIfStandaloneConsole()
	}

	if err := cli.Execute(version, updateURL); err != nil {
		if exitErr, ok := errors.AsType[*cjverr.ExitCodeError](err); ok {
			return exitErr.Code
		}
		// In JSON mode the envelope was already written to stdout by
		// cli.Execute; keep stderr clean so consumers see only JSON on stdout.
		if !output.IsJSON() {
			fmt.Fprintln(os.Stderr, "cjv:", err)
		}
		return 1
	}
	return 0
}

// isInitInvocation makes the binary double as an installer when launched by name.
// Prefix match tolerates browser-renamed duplicates such as "cjv-init(1)" or "cjv-init-2".
func isInitInvocation(toolName string) bool {
	return strings.HasPrefix(toolName, "cjv-init") || strings.HasPrefix(toolName, "cjv-setup")
}
