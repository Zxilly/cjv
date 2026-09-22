package cli

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	clisettings "github.com/Zxilly/cjv/internal/cli/settings"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/i18n"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type toolchainListRemoteEntry struct {
	Channel       string   `json:"channel"`
	Latest        string   `json:"latest,omitempty"`
	Versions      []string `json:"versions"`
	Error         string   `json:"error,omitempty"`
	PlatformAware bool     `json:"-"`
}

type toolchainListRemoteResult struct {
	Target   string                     `json:"target"`
	Channels []toolchainListRemoteEntry `json:"channels"`
}

type platformVersionsEntry struct {
	Target   string   `json:"target"`
	Versions []string `json:"versions"`
}

type toolchainListRemoteAllPlatformsEntry struct {
	Channel   string                  `json:"channel"`
	Latest    string                  `json:"latest,omitempty"`
	Platforms []platformVersionsEntry `json:"platforms,omitempty"`
	Error     string                  `json:"error,omitempty"`
}

type toolchainListRemoteAllPlatformsResult struct {
	AllPlatforms bool                                   `json:"all_platforms"`
	Channels     []toolchainListRemoteAllPlatformsEntry `json:"channels"`
}

func (app *application) runToolchainListRemote(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	channel, allChannels, err := parseListRemoteChannel(app.toolchainListRemoteChannel)
	if err != nil {
		return err
	}

	_, settings, err := clisettings.LoadSettings()
	if err != nil {
		return err
	}

	if app.toolchainListRemoteAllPlatforms {
		return app.runToolchainListRemoteAllPlatforms(ctx, cmd, settings, channel, allChannels)
	}
	return app.runToolchainListRemoteSingle(ctx, cmd, settings, channel, allChannels)
}

func parseListRemoteChannel(raw string) (toolchain.Channel, bool, error) {
	trimmed := strings.TrimSpace(raw)
	if strings.EqualFold(trimmed, "all") {
		return toolchain.UnknownChannel, true, nil
	}
	if ch, ok := toolchain.ParseChannel(trimmed); ok {
		return ch, false, nil
	}
	return toolchain.UnknownChannel, false, errors.New(i18n.T("ListRemoteUnknownChannelFlag", i18n.MsgData{"Value": raw}))
}

func (app *application) runToolchainListRemoteSingle(ctx context.Context, cmd *cobra.Command, settings *config.Settings, channel toolchain.Channel, allChannels bool) error {
	tuple, err := app.resolveListRemoteTuple(settings)
	if err != nil {
		return err
	}

	result := toolchainListRemoteResult{
		Target:   tuple,
		Channels: []toolchainListRemoteEntry{},
	}

	needLTS := allChannels || channel == toolchain.LTS
	needSTS := allChannels || channel == toolchain.STS
	needNightly := allChannels || channel == toolchain.Nightly

	source, err := dist.NewSource(settings)
	if err != nil {
		return err
	}
	if needLTS {
		result.Channels = append(result.Channels, app.buildSourceChannelEntry(ctx, source, toolchain.LTS, tuple))
	}
	if needSTS {
		result.Channels = append(result.Channels, app.buildSourceChannelEntry(ctx, source, toolchain.STS, tuple))
	}
	if needNightly {
		entry := app.buildSourceChannelEntry(ctx, source, toolchain.Nightly, tuple)
		// When the user explicitly asks for only the nightly channel and it
		// fails, propagate the error so CI gets a non-zero exit code.
		if !allChannels && entry.Error != "" {
			return errors.New(entry.Error)
		}
		result.Channels = append(result.Channels, entry)
	}

	return app.output.RenderTo(cmdOutput(cmd), result)
}

func (app *application) buildSourceChannelEntry(ctx context.Context, source *dist.Source, ch toolchain.Channel, tuple string) toolchainListRemoteEntry {
	entry := toolchainListRemoteEntry{
		Channel:       ch.String(),
		Versions:      []string{},
		PlatformAware: true,
	}
	latest, versions, err := source.ChannelVersions(ctx, ch, tuple)
	if err != nil {
		entry.Error = err.Error()
		return entry
	}
	if app.toolchainListRemoteLimit > 0 && len(versions) > app.toolchainListRemoteLimit {
		versions = versions[:app.toolchainListRemoteLimit]
	}
	entry.Latest = latest
	entry.Versions = versions
	return entry
}

// resolveListRemoteTuple mirrors install's --target semantics: an empty
// environment yields the current host tuple, otherwise it composes
// <host>-<environment>. Validation (rejecting host tuples passed as
// environments, etc.) is delegated to dist.CurrentTargetTuple.
func (app *application) resolveListRemoteTuple(settings *config.Settings) (string, error) {
	target, err := sdktarget.Normalize(app.toolchainListRemoteTarget)
	if err != nil {
		return "", err
	}
	return dist.CurrentTargetTuple(settings.DefaultHost, target)
}

func (app *application) runToolchainListRemoteAllPlatforms(ctx context.Context, cmd *cobra.Command, settings *config.Settings, channel toolchain.Channel, allChannels bool) error {
	needLTS := allChannels || channel == toolchain.LTS
	needSTS := allChannels || channel == toolchain.STS
	needNightly := allChannels || channel == toolchain.Nightly

	result := toolchainListRemoteAllPlatformsResult{
		AllPlatforms: true,
		Channels:     []toolchainListRemoteAllPlatformsEntry{},
	}

	source, err := dist.NewSource(settings)
	if err != nil {
		return err
	}
	if needLTS {
		result.Channels = append(result.Channels, app.buildSourceAllPlatformsEntry(ctx, source, toolchain.LTS))
	}
	if needSTS {
		result.Channels = append(result.Channels, app.buildSourceAllPlatformsEntry(ctx, source, toolchain.STS))
	}
	if needNightly {
		entry := app.buildSourceAllPlatformsEntry(ctx, source, toolchain.Nightly)
		if !allChannels && entry.Error != "" {
			return errors.New(entry.Error)
		}
		result.Channels = append(result.Channels, entry)
	}

	return app.output.RenderTo(cmdOutput(cmd), result)
}

func (app *application) buildSourceAllPlatformsEntry(ctx context.Context, source *dist.Source, ch toolchain.Channel) toolchainListRemoteAllPlatformsEntry {
	entry := toolchainListRemoteAllPlatformsEntry{Channel: ch.String()}
	latest, grouped, err := source.ChannelVersionsByTuple(ctx, ch)
	if err != nil {
		entry.Error = err.Error()
		return entry
	}
	entry.Latest = latest
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		versions := grouped[key]
		if app.toolchainListRemoteLimit > 0 && len(versions) > app.toolchainListRemoteLimit {
			versions = versions[:app.toolchainListRemoteLimit]
		}
		entry.Platforms = append(entry.Platforms, platformVersionsEntry{Target: key, Versions: versions})
	}
	return entry
}

func (r toolchainListRemoteResult) Text() string {
	var b strings.Builder
	for i, e := range r.Channels {
		if i > 0 {
			b.WriteByte('\n')
		}
		writeSingleChannelHeader(&b, e, r.Target)
		writeSingleChannelBody(&b, e, r.Target)
	}
	if len(r.Channels) > 0 {
		b.WriteByte('\n')
		b.WriteString(i18n.T("ListRemoteHint", nil))
		b.WriteByte('\n')
	}
	return b.String()
}

func writeSingleChannelHeader(b *strings.Builder, e toolchainListRemoteEntry, tuple string) {
	// A channel-level source uses the compact header; a platform catalog adds
	// the selected target.
	withPlatform := e.PlatformAware && e.Latest != ""
	switch {
	case withPlatform:
		fmt.Fprintln(b, i18n.T("ListRemoteChannelHeaderTarget", i18n.MsgData{
			"Channel": e.Channel,
			"Latest":  e.Latest,
			"Target":  tuple,
		}))
	case e.Latest != "":
		fmt.Fprintln(b, i18n.T("ListRemoteChannelHeaderWithLatest", i18n.MsgData{
			"Channel": e.Channel,
			"Latest":  e.Latest,
		}))
	default:
		fmt.Fprintln(b, i18n.T("ListRemoteChannelHeader", i18n.MsgData{"Channel": e.Channel}))
	}
}

func writeSingleChannelBody(b *strings.Builder, e toolchainListRemoteEntry, tuple string) {
	if e.Error != "" {
		fmt.Fprintln(b, "  "+color.YellowString("(%s)", e.Error))
		return
	}
	if len(e.Versions) == 0 {
		fmt.Fprintln(b, "  "+color.YellowString("%s", i18n.T("ListRemoteNoVersionsForTarget", i18n.MsgData{
			"Target": tuple,
		})))
		return
	}
	writeVersionLines(b, e.Versions, e.Latest, "  ")
}

func (r toolchainListRemoteAllPlatformsResult) Text() string {
	var b strings.Builder
	for i, e := range r.Channels {
		if i > 0 {
			b.WriteByte('\n')
		}
		writeAllPlatformsChannelHeader(&b, e)
		writeAllPlatformsChannelBody(&b, e)
	}
	if len(r.Channels) > 0 {
		b.WriteByte('\n')
		b.WriteString(i18n.T("ListRemoteHintAllPlatforms", nil))
		b.WriteByte('\n')
	}
	return b.String()
}

func writeAllPlatformsChannelHeader(b *strings.Builder, e toolchainListRemoteAllPlatformsEntry) {
	if e.Latest != "" {
		fmt.Fprintln(b, i18n.T("ListRemoteChannelHeaderWithLatest", i18n.MsgData{
			"Channel": e.Channel,
			"Latest":  e.Latest,
		}))
		return
	}
	fmt.Fprintln(b, i18n.T("ListRemoteChannelHeader", i18n.MsgData{"Channel": e.Channel}))
}

func writeAllPlatformsChannelBody(b *strings.Builder, e toolchainListRemoteAllPlatformsEntry) {
	if e.Error != "" {
		fmt.Fprintln(b, "  "+color.YellowString("(%s)", e.Error))
		return
	}
	for _, p := range e.Platforms {
		fmt.Fprintf(b, "  %s\n", p.Target)
		writeVersionLines(b, p.Versions, e.Latest, "    ")
	}
}

func writeVersionLines(b *strings.Builder, versions []string, latest, indent string) {
	for _, v := range versions {
		if v == latest {
			fmt.Fprintf(b, "%s%s\n", indent, color.GreenString("%s *", v))
			continue
		}
		fmt.Fprintf(b, "%s%s\n", indent, v)
	}
}

func (app *application) initToolchainListRemoteCommands() {
	app.toolchainListRemoteCmd = &cobra.Command{
		Use:   "list-remote",
		Short: i18n.T("ToolchainListRemoteShort", nil),
		Long:  i18n.T("ToolchainListRemoteLong", nil),
		Args:  cobra.NoArgs,
		RunE:  app.runToolchainListRemote,
	}

	app.toolchainListRemoteCmd.Flags().StringVar(&app.toolchainListRemoteChannel, "channel", "all", "Channel to list (all|lts|sts|nightly)")
	app.toolchainListRemoteCmd.Flags().StringVarP(&app.toolchainListRemoteTarget, "target", "t", "", "Cross-compilation target suffix (e.g. ohos)")
	app.toolchainListRemoteCmd.Flags().BoolVar(&app.toolchainListRemoteAllPlatforms, "all-platforms", false, "List versions grouped by every target tuple")
	app.toolchainListRemoteCmd.Flags().IntVar(&app.toolchainListRemoteLimit, "limit", 0, "Show at most N versions per channel/platform (0 = no limit)")
	app.toolchainListRemoteCmd.MarkFlagsMutuallyExclusive("target", "all-platforms")
	app.toolchainCmd.AddCommand(app.toolchainListRemoteCmd)

}
