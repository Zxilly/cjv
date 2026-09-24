package progress

import (
	"bytes"
	"errors"
	"testing"

	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
