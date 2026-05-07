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
	"github.com/Zxilly/cjv/internal/logging"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/resolve"
	"github.com/Zxilly/cjv/internal/utils"
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

	// Break circular import: proxy cannot import cli, so we wire the callback here.
	resolve.AutoInstallFunc = func(ctx context.Context, input string, targets []string) error {
		return cli.InstallToolchainWithTargets(ctx, input, targets, false)
	}
	resolve.AutoInstallComponentsFunc = func(ctx context.Context, input string, components []string) error {
		return cli.InstallComponentsForToolchain(ctx, input, components)
	}

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
		os.Args = append([]string{os.Args[0], "init"}, os.Args[1:]...)
		defer utils.PauseIfStandaloneConsole()
	}

	if err := cli.Execute(version, updateURL); err != nil {
		if exitErr, ok := errors.AsType[*cjverr.ExitCodeError](err); ok {
			return exitErr.Code
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
