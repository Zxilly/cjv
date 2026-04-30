package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

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
	logging.Init()

	utils.AppVersion = version

	// Break circular import: proxy cannot import cli, so we wire the callback here.
	resolve.AutoInstallFunc = func(ctx context.Context, input string, targets []string) error {
		return cli.InstallToolchainWithTargets(ctx, input, targets, false)
	}

	toolName := proxy.ExtractToolName(os.Args[0])

	if proxy.IsProxyTool(toolName) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		if err := proxy.Run(ctx, toolName, os.Args[1:]); err != nil {
			if exitErr, ok := errors.AsType[*cjverr.ExitCodeError](err); ok {
				os.Exit(exitErr.Code)
			}
			fmt.Fprintln(os.Stderr, "cjv:", err)
			os.Exit(1)
		}
		return
	}

	if err := cli.Execute(version, updateURL); err != nil {
		if exitErr, ok := errors.AsType[*cjverr.ExitCodeError](err); ok {
			os.Exit(exitErr.Code)
		}
		os.Exit(1)
	}
}
