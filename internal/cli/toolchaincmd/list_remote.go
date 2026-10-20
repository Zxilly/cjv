package toolchaincmd

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
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

func (c *listRemote) run(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	channel, allChannels, err := parseListRemoteChannel(c.channel)
	if err != nil {
		return err
	}

	d, err := lifecycle.OpenDistribution(c.ctx.LifecycleOptions())
	if err != nil {
		return err
	}

	if c.allPlatforms {
		return c.runAllPlatforms(ctx, cmd, d.Source, channel, allChannels)
	}
	return c.runSingle(ctx, cmd, d, channel, allChannels)
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

func (c *listRemote) runSingle(ctx context.Context, cmd *cobra.Command, d *lifecycle.Distribution, channel toolchain.Channel, allChannels bool) error {
	// --target mirrors install's semantics: an empty environment yields the
	// current host tuple, otherwise <host>-<environment>. Validation
	// (rejecting host tuples passed as environments, etc.) is the
	// Distribution's.
	tuple, err := d.TargetTuple(c.target)
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

	source := d.Source
	if needLTS {
		result.Channels = append(result.Channels, c.channelEntry(ctx, source, toolchain.LTS, tuple))
	}
	if needSTS {
		result.Channels = append(result.Channels, c.channelEntry(ctx, source, toolchain.STS, tuple))
	}
	if needNightly {
		entry := c.channelEntry(ctx, source, toolchain.Nightly, tuple)
		// When the user explicitly asks for only the nightly channel and it
		// fails, propagate the error so CI gets a non-zero exit code.
		if !allChannels && entry.Error != "" {
			return errors.New(entry.Error)
		}
		result.Channels = append(result.Channels, entry)
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	return c.ctx.Output.RenderTo(clictx.Out(cmd), result)
}

func (c *listRemote) channelEntry(ctx context.Context, source *dist.Source, ch toolchain.Channel, tuple string) toolchainListRemoteEntry {
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
	if c.limit > 0 && len(versions) > c.limit {
		versions = versions[:c.limit]
	}
	entry.Latest = latest
	entry.Versions = versions
	return entry
}

func (c *listRemote) runAllPlatforms(ctx context.Context, cmd *cobra.Command, source *dist.Source, channel toolchain.Channel, allChannels bool) error {
	needLTS := allChannels || channel == toolchain.LTS
	needSTS := allChannels || channel == toolchain.STS
	needNightly := allChannels || channel == toolchain.Nightly

	result := toolchainListRemoteAllPlatformsResult{
		AllPlatforms: true,
		Channels:     []toolchainListRemoteAllPlatformsEntry{},
	}

	if needLTS {
		result.Channels = append(result.Channels, c.allPlatformsEntry(ctx, source, toolchain.LTS))
	}
	if needSTS {
		result.Channels = append(result.Channels, c.allPlatformsEntry(ctx, source, toolchain.STS))
	}
	if needNightly {
		entry := c.allPlatformsEntry(ctx, source, toolchain.Nightly)
		if !allChannels && entry.Error != "" {
			return errors.New(entry.Error)
		}
		result.Channels = append(result.Channels, entry)
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	return c.ctx.Output.RenderTo(clictx.Out(cmd), result)
}

func (c *listRemote) allPlatformsEntry(ctx context.Context, source *dist.Source, ch toolchain.Channel) toolchainListRemoteAllPlatformsEntry {
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
		if c.limit > 0 && len(versions) > c.limit {
			versions = versions[:c.limit]
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

// listRemote holds the flags of `cjv toolchain list-remote` for one invocation.
type listRemote struct {
	ctx          *clictx.Context
	channel      string
	target       string
	allPlatforms bool
	limit        int
}

func (c *listRemote) command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-remote",
		Short: i18n.T("ToolchainListRemoteShort", nil),
		Long:  i18n.T("ToolchainListRemoteLong", nil),
		Args:  cobra.NoArgs,
		RunE:  c.run,
	}
	cmd.Flags().StringVar(&c.channel, "channel", "all", "Channel to list (all|lts|sts|nightly)")
	cmd.Flags().StringVarP(&c.target, "target", "t", "", "Cross-compilation target suffix (e.g. ohos)")
	cmd.Flags().BoolVar(&c.allPlatforms, "all-platforms", false, "List versions grouped by every target tuple")
	cmd.Flags().IntVar(&c.limit, "limit", 0, "Show at most N versions per channel/platform (0 = no limit)")
	cmd.MarkFlagsMutuallyExclusive("target", "all-platforms")
	return cmd
}
