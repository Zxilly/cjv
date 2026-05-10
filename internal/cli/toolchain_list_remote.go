package cli

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Zxilly/cjv/internal/cli/output"
	clisettings "github.com/Zxilly/cjv/internal/cli/settings"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/i18n"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	toolchainListRemoteChannel      string
	toolchainListRemoteTarget       string
	toolchainListRemoteAllPlatforms bool
	toolchainListRemoteLimit        int
)

var toolchainListRemoteCmd = &cobra.Command{
	Use:   "list-remote",
	Short: "List all toolchain versions available remotely",
	Long: `List Cangjie toolchain versions available from the remote SDK manifest.

By default lists versions for the current host target tuple across all
channels (lts, sts, nightly). Pass --target <suffix> to query a cross-compile
build (matching ` + "`cjv install --target`" + `), or --all-platforms to enumerate
every target tuple present in the manifest.`,
	Args: cobra.NoArgs,
	RunE: runToolchainListRemote,
}

type toolchainListRemoteEntry struct {
	Channel  string   `json:"channel"`
	Latest   string   `json:"latest,omitempty"`
	Versions []string `json:"versions"`
	Error    string   `json:"error,omitempty"`
}

type toolchainListRemoteResult struct {
	Target string                     `json:"target"`
	Channels    []toolchainListRemoteEntry `json:"channels"`
}

type platformVersionsEntry struct {
	Target string   `json:"target"`
	Versions    []string `json:"versions"`
}

// LTS/STS populate Platforms; nightly populates Versions (single tag) — the
// nightly tag is platform-orthogonal and has no per-platform breakdown.
type toolchainListRemoteAllPlatformsEntry struct {
	Channel   string                  `json:"channel"`
	Latest    string                  `json:"latest,omitempty"`
	Platforms []platformVersionsEntry `json:"platforms,omitempty"`
	Versions  []string                `json:"versions,omitempty"`
	Error     string                  `json:"error,omitempty"`
}

type toolchainListRemoteAllPlatformsResult struct {
	AllPlatforms bool                                   `json:"all_platforms"`
	Channels     []toolchainListRemoteAllPlatformsEntry `json:"channels"`
}

func runToolchainListRemote(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	channel, allChannels, err := parseListRemoteChannel(toolchainListRemoteChannel)
	if err != nil {
		return err
	}

	_, settings, err := clisettings.LoadSettings()
	if err != nil {
		return err
	}

	if toolchainListRemoteAllPlatforms {
		return runToolchainListRemoteAllPlatforms(ctx, cmd, settings, channel, allChannels)
	}
	return runToolchainListRemoteSingle(ctx, cmd, settings, channel, allChannels)
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

func runToolchainListRemoteSingle(ctx context.Context, cmd *cobra.Command, settings *config.Settings, channel toolchain.Channel, allChannels bool) error {
	tuple, err := resolveListRemoteTuple(settings)
	if err != nil {
		return err
	}

	result := toolchainListRemoteResult{
		Target: tuple,
		Channels:    []toolchainListRemoteEntry{},
	}

	needLTS := allChannels || channel == toolchain.LTS
	needSTS := allChannels || channel == toolchain.STS
	needNightly := allChannels || channel == toolchain.Nightly

	var manifest *dist.Manifest
	if needLTS || needSTS {
		manifest, err = fetchManifest(ctx, settings.ManifestURL)
		if err != nil {
			return err
		}
	}
	if needLTS {
		result.Channels = append(result.Channels, buildSingleChannelEntry(manifest, toolchain.LTS, tuple))
	}
	if needSTS {
		result.Channels = append(result.Channels, buildSingleChannelEntry(manifest, toolchain.STS, tuple))
	}
	if needNightly {
		entry := buildNightlyEntry(ctx, settings)
		// When the user explicitly asks for only the nightly channel and it
		// fails, propagate the error so CI gets a non-zero exit code.
		if !allChannels && entry.Error != "" {
			return errors.New(entry.Error)
		}
		result.Channels = append(result.Channels, entry)
	}

	return output.RenderTo(cmdOutput(cmd), result)
}

// resolveListRemoteTuple mirrors install's --target semantics: an empty
// environment yields the current host tuple, otherwise it composes
// <host>-<environment>. Validation (rejecting host tuples passed as
// environments, etc.) is delegated to dist.CurrentTargetTuple.
func resolveListRemoteTuple(settings *config.Settings) (string, error) {
	target, err := sdktarget.Normalize(toolchainListRemoteTarget)
	if err != nil {
		return "", err
	}
	return dist.CurrentTargetTuple(settings.DefaultHost, target)
}

func buildSingleChannelEntry(m *dist.Manifest, ch toolchain.Channel, tuple string) toolchainListRemoteEntry {
	entry := toolchainListRemoteEntry{
		Channel:  ch.String(),
		Versions: []string{},
	}
	versions, err := m.ListVersions(ch, tuple)
	if err != nil {
		entry.Error = err.Error()
		return entry
	}
	if toolchainListRemoteLimit > 0 && len(versions) > toolchainListRemoteLimit {
		versions = versions[:toolchainListRemoteLimit]
	}
	entry.Versions = versions
	if latest, lErr := m.GetLatestVersion(ch); lErr == nil {
		entry.Latest = latest
	}
	return entry
}

func buildNightlyEntry(ctx context.Context, settings *config.Settings) toolchainListRemoteEntry {
	entry := toolchainListRemoteEntry{
		Channel:  toolchain.Nightly.String(),
		Versions: []string{},
	}
	tag, err := dist.FetchLatestNightly(ctx, dist.DefaultNightlyAPIURL, settings.GitCodeAPIKey)
	if err != nil {
		entry.Error = err.Error()
		return entry
	}
	entry.Latest = tag
	entry.Versions = []string{tag}
	return entry
}

func runToolchainListRemoteAllPlatforms(ctx context.Context, cmd *cobra.Command, settings *config.Settings, channel toolchain.Channel, allChannels bool) error {
	needLTS := allChannels || channel == toolchain.LTS
	needSTS := allChannels || channel == toolchain.STS
	needNightly := allChannels || channel == toolchain.Nightly

	result := toolchainListRemoteAllPlatformsResult{
		AllPlatforms: true,
		Channels:     []toolchainListRemoteAllPlatformsEntry{},
	}

	var manifest *dist.Manifest
	var err error
	if needLTS || needSTS {
		manifest, err = fetchManifest(ctx, settings.ManifestURL)
		if err != nil {
			return err
		}
	}
	if needLTS {
		result.Channels = append(result.Channels, buildAllPlatformsChannelEntry(manifest, toolchain.LTS))
	}
	if needSTS {
		result.Channels = append(result.Channels, buildAllPlatformsChannelEntry(manifest, toolchain.STS))
	}
	if needNightly {
		entry := buildNightlyAllPlatformsEntry(ctx, settings)
		if !allChannels && entry.Error != "" {
			return errors.New(entry.Error)
		}
		result.Channels = append(result.Channels, entry)
	}

	return output.RenderTo(cmdOutput(cmd), result)
}

func buildAllPlatformsChannelEntry(m *dist.Manifest, ch toolchain.Channel) toolchainListRemoteAllPlatformsEntry {
	entry := toolchainListRemoteAllPlatformsEntry{Channel: ch.String()}
	if latest, lErr := m.GetLatestVersion(ch); lErr == nil {
		entry.Latest = latest
	}
	pkVersions, err := m.VersionsByTuple(ch)
	if err != nil {
		entry.Error = err.Error()
		return entry
	}
	keys := make([]string, 0, len(pkVersions))
	for k := range pkVersions {
		keys = append(keys, k)
	}
	// Lexical sort yields host alphabetic, with bare "linux-x64" preceding
	// "linux-x64-ohos" because '-' (0x2D) sorts after end-of-string.
	sort.Strings(keys)

	platforms := make([]platformVersionsEntry, 0, len(keys))
	for _, pk := range keys {
		versions := pkVersions[pk]
		if toolchainListRemoteLimit > 0 && len(versions) > toolchainListRemoteLimit {
			versions = versions[:toolchainListRemoteLimit]
		}
		platforms = append(platforms, platformVersionsEntry{
			Target: pk,
			Versions:    versions,
		})
	}
	entry.Platforms = platforms
	return entry
}

func buildNightlyAllPlatformsEntry(ctx context.Context, settings *config.Settings) toolchainListRemoteAllPlatformsEntry {
	entry := toolchainListRemoteAllPlatformsEntry{Channel: toolchain.Nightly.String()}
	tag, err := dist.FetchLatestNightly(ctx, dist.DefaultNightlyAPIURL, settings.GitCodeAPIKey)
	if err != nil {
		entry.Error = err.Error()
		entry.Versions = []string{}
		return entry
	}
	entry.Latest = tag
	entry.Versions = []string{tag}
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
	// Nightly is platform-orthogonal — show the channel header without
	// Target to avoid implying that the tag was filtered by platform.
	withPlatform := e.Channel != toolchain.Nightly.String() && e.Latest != ""
	switch {
	case withPlatform:
		fmt.Fprintln(b, i18n.T("ListRemoteChannelHeaderTarget", i18n.MsgData{
			"Channel":     e.Channel,
			"Latest":      e.Latest,
			"Target": tuple,
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
	if e.Channel == toolchain.Nightly.String() {
		writeVersionLines(b, e.Versions, e.Latest, "  ")
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

func init() {
	toolchainListRemoteCmd.Flags().StringVar(&toolchainListRemoteChannel, "channel", "all", "Channel to list (all|lts|sts|nightly)")
	toolchainListRemoteCmd.Flags().StringVarP(&toolchainListRemoteTarget, "target", "t", "", "Cross-compilation target suffix (e.g. ohos)")
	toolchainListRemoteCmd.Flags().BoolVar(&toolchainListRemoteAllPlatforms, "all-platforms", false, "List versions grouped by every target tuple")
	toolchainListRemoteCmd.Flags().IntVar(&toolchainListRemoteLimit, "limit", 0, "Show at most N versions per channel/platform (0 = no limit)")
	toolchainListRemoteCmd.MarkFlagsMutuallyExclusive("target", "all-platforms")
	toolchainCmd.AddCommand(toolchainListRemoteCmd)
}
