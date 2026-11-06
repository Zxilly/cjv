package progress

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vbauerster/mpb/v8"
)

func TestDownloadBarFitsNarrowTerminals(t *testing.T) {
	for _, subject := range []string{
		"cangjie-sdk-linux-x64-1.0.5.tar.gz",
		"仓颉工具链-linux-x64-1.0.5.tar.gz",
	} {
		for _, width := range []int{20, 32, 40, 66, 80, 120} {
			for _, percent := range []int64{98, 100} {
				t.Run(fmt.Sprintf("%s/%d/%d", subject, width, percent), func(t *testing.T) {
					var out bytes.Buffer
					p := mpb.New(mpb.WithOutput(&out), mpb.WithWidth(width), mpb.WithManualRefresh(make(chan any)))
					bar := newDownloadBar(p, Event{Subject: subject, Total: 281752371})
					if percent == 100 {
						bar.SetTotal(281752371, true)
					} else {
						bar.SetCurrent(281752371 * percent / 100)
						bar.Abort(false) // Render the in-flight frame without waiting for a timer.
					}
					p.Wait()
					line := strings.TrimSpace(ansi.Strip(out.String()))
					t.Log(line)
					assert.LessOrEqual(t, ansi.StringWidth(line), width)
					assert.Contains(t, strings.ReplaceAll(line, " ", ""), fmt.Sprintf("%d%%", percent))
					parts := strings.Split(line, "|")
					require.Len(t, parts, 3, "the bar keeps both boundaries")
					assert.GreaterOrEqual(t, ansi.StringWidth(parts[1]), 8, "the bar stays readable")
					if width >= 66 {
						assert.Contains(t, line, "MiB")
					}
					if width == 120 {
						assert.Contains(t, line, subject)
					}
				})
			}
		}
	}
}

func TestTextRendersMessagesInOrder(t *testing.T) {
	var out, bar bytes.Buffer
	text := NewText(&out, &bar)

	text.Report(Event{Kind: FetchingManifest})
	text.Report(Event{Kind: ToolchainAlreadyInstalled, Toolchain: "lts-1.0.5"})
	text.Report(Event{Kind: UpdateFound, Toolchain: "lts-1.0.0", Replacement: "lts-1.0.5"})
	text.Report(Event{Kind: ComponentInstalled, Toolchain: "lts-1.0.5", Component: "docs"})
	text.Report(Event{Kind: AutoInstallFailed, Subject: "sts", Err: errors.New("boom")})

	lines := bytes.Split(bytes.TrimSuffix(out.Bytes(), []byte("\n")), []byte("\n"))
	require.Len(t, lines, 5)
	assert.Equal(t, i18n.T("FetchingManifest", nil), string(lines[0]))
	assert.Equal(t, i18n.T("ToolchainAlreadyInstalled", i18n.MsgData{"Name": "lts-1.0.5"}), string(lines[1]))
	assert.Equal(t, i18n.T("UpdateFound", i18n.MsgData{"Current": "lts-1.0.0", "Latest": "lts-1.0.5"}), string(lines[2]))
	assert.Contains(t, string(lines[3]), i18n.T("ComponentInstalled", i18n.MsgData{"Toolchain": "lts-1.0.5", "Component": "docs"}))
	assert.Equal(t, i18n.T("AutoInstallFailed", i18n.MsgData{"Name": "sts", "Err": "boom"}), string(lines[4]))
	assert.Empty(t, bar.String(), "messages never touch the bar writer")
}

func TestTextDrawsNoBarOffTerminal(t *testing.T) {
	var out, bar bytes.Buffer
	text := NewText(&out, &bar)

	text.Report(Event{Kind: DownloadStarted, Subject: "sdk.zip", Total: 100})
	text.Report(Event{Kind: DownloadAdvanced, Bytes: 60})
	text.Report(Event{Kind: DownloadFinished})
	text.Report(Event{Kind: DownloadStarted, Subject: "sdk.zip", Total: -1})
	text.Report(Event{Kind: DownloadFinished, Err: errors.New("cut")})

	assert.Empty(t, out.String())
	assert.Empty(t, bar.String())
}

func TestDiscardAndOr(t *testing.T) {
	Discard.Report(Event{Kind: ToolchainInstalled, Toolchain: "lts-1.0.5"})
	assert.Equal(t, Discard, Or(nil))
	var out bytes.Buffer
	text := NewText(&out, &out)
	assert.Equal(t, Sink(text), Or(text))
}
