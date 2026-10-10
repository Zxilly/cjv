package cli

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/charmbracelet/huh"
)

// Discover native host builds from the configured manifests, including the
// separate nightly manifest. Cross SDKs do not make a channel host-compatible.
// Failed requests are reported separately from confirmed unavailable channels.
func initAvailableChannels(ctx context.Context, source *dist.Source, tuple string) ([]string, error) {
	var channels []string
	var failures []error
	for _, channel := range []toolchain.Channel{toolchain.LTS, toolchain.STS, toolchain.Nightly} {
		_, err := source.LatestAvailableVersion(ctx, channel, tuple)
		if err == nil {
			channels = append(channels, channel.String())
			continue
		}
		var unavailable *cjverr.VersionNotAvailableError
		if errors.As(err, &unavailable) || errors.Is(err, dist.ErrManifestChannelMissing) {
			continue
		}
		failures = append(failures, fmt.Errorf("%s: %w", channel, err))
		if ctx.Err() != nil {
			break
		}
	}
	return channels, errors.Join(failures...)
}

func initPreferredToolchain(channels []string) string {
	if len(channels) > 0 {
		return channels[0]
	}
	return "none"
}

func initToolchainOptions(opts *initCustomizeOptions) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(opts.channels)+2)
	for _, channel := range opts.channels {
		options = append(options, huh.NewOption(channel, channel))
	}
	// Preserve an explicitly requested version or custom toolchain in the form.
	if opts.toolchain != "none" && !slices.Contains(opts.channels, opts.toolchain) {
		switch opts.toolchain {
		case "auto", "lts", "sts", "nightly":
			opts.toolchain = initPreferredToolchain(opts.channels)
		default:
			options = append(options, huh.NewOption(opts.toolchain, opts.toolchain))
		}
	}
	return append(options, huh.NewOption(i18n.T("InitToolchainNone", nil), "none"))
}
