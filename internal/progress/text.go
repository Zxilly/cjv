package progress

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/charmbracelet/x/ansi"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

const (
	downloadBarWidth        = 24
	downloadMinBarWidth     = 10
	downloadMinNameWidth    = 8
	downloadPercentageWidth = 6
)

// Text is the human-readable adapter: each event becomes its i18n message on
// the messages writer, and a download draws an animated bar on the bar writer
// while that writer is a terminal. Completed installs are highlighted.
type Text struct {
	messages io.Writer
	barOut   io.Writer

	progress *mpb.Progress
	bar      *mpb.Bar
	total    int64
}

// NewText renders messages to messages and the download bar to bar. The bar
// is drawn only when bar is an interactive terminal, so piped and CI output
// stays clean.
func NewText(messages, bar io.Writer) *Text {
	return &Text{messages: messages, barOut: bar}
}

// Report implements Sink.
func (t *Text) Report(e Event) {
	switch e.Kind {
	case DownloadStarted:
		t.startBar(e)
	case DownloadAdvanced:
		if t.bar != nil {
			t.bar.IncrBy(int(e.Bytes))
		}
	case DownloadFinished:
		t.finishBar(e.Err)
	default:
		if text, ok := message(e); ok {
			_, _ = io.WriteString(t.messages, text+"\n")
		}
	}
}

// message renders the i18n text for a non-download event.
func message(e Event) (string, bool) {
	switch e.Kind {
	case FetchingManifest:
		return i18n.T("FetchingManifest", nil), true
	case NightlyNoChecksum:
		return i18n.T("NightlyNoChecksum", nil), true
	case Extracting:
		return i18n.T("Extracting", nil), true
	case ToolchainAlreadyInstalled:
		return i18n.T("ToolchainAlreadyInstalled", i18n.MsgData{"Name": e.Toolchain}), true
	case ToolchainInstalled:
		return color.GreenString("%s", i18n.T("ToolchainInstalled", i18n.MsgData{"Name": e.Toolchain})), true
	case ComponentAlreadyInstalled:
		return i18n.T("ComponentAlreadyInstalled", componentData(e)), true
	case ComponentInstalled:
		return color.GreenString("%s", i18n.T("ComponentInstalled", componentData(e))), true
	case FetchingComponent:
		return i18n.T("FetchingComponent", componentData(e)), true
	case InstallingComponent:
		return i18n.T("InstallingComponent", componentData(e)), true
	case RemovingComponent:
		return i18n.T("RemovingComponent", componentData(e)), true
	case AlreadyUpToDate:
		return i18n.T("AlreadyUpToDate", i18n.MsgData{"Version": e.Toolchain}), true
	case UpdateFound:
		return i18n.T("UpdateFound", i18n.MsgData{"Current": e.Toolchain, "Latest": e.Replacement}), true
	case LinkDownloadingURL:
		return i18n.T("LinkDownloadingURL", i18n.MsgData{"URL": e.Subject}), true
	case LinkUsingArchive:
		return i18n.T("LinkUsingArchive", i18n.MsgData{"Path": e.Subject}), true
	case LinkInstallingStdx:
		return i18n.T("LinkInstallingStdx", i18n.MsgData{"Name": e.Toolchain}), true
	case AutoInstalling:
		return i18n.T("AutoInstalling", i18n.MsgData{"Name": e.Subject}), true
	case AutoInstallFailed:
		errText := ""
		if e.Err != nil {
			errText = e.Err.Error()
		}
		return i18n.T("AutoInstallFailed", i18n.MsgData{"Name": e.Subject, "Err": errText}), true
	}
	return "", false
}

func componentData(e Event) i18n.MsgData {
	return i18n.MsgData{"Toolchain": e.Toolchain, "Component": e.Component}
}

func (t *Text) interactive() bool {
	f, ok := t.barOut.(*os.File)
	return ok && (isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd()))
}

func (t *Text) startBar(e Event) {
	t.finishBar(nil)
	if e.Total <= 0 || !t.interactive() {
		return
	}
	t.progress = mpb.New(
		mpb.WithOutput(t.barOut),
		mpb.WithRefreshRate(100*time.Millisecond),
	)
	t.total = e.Total
	t.bar = newDownloadBar(t.progress, e)
	if e.Bytes > 0 {
		t.bar.SetCurrent(e.Bytes)
	}
}

func newDownloadBar(p *mpb.Progress, e Event) *mpb.Bar {
	return p.New(e.Total,
		mpb.BarStyle().Lbound("|").Filler("=").Tip(">").Padding(" ").Rbound("|"),
		mpb.BarWidth(downloadBarWidth),
		mpb.PrependDecorators(
			decor.Any(func(s decor.Statistics) string {
				// Reserve a readable bar and percentage before spending columns
				// on the filename. Recompute on every frame to handle resizing.
				width := s.AvailableWidth - downloadPercentageWidth - downloadMinBarWidth - 2
				counters := downloadCounters(s)
				if width >= ansi.StringWidth(counters)+downloadMinNameWidth+1 {
					width -= ansi.StringWidth(counters)
				}
				if width <= 1 {
					return ""
				}
				return ansi.Truncate(e.Subject, width-1, "…") + " "
			}),
			decor.Percentage(decor.WC{W: downloadPercentageWidth, C: decor.DindentRight}),
		),
		mpb.AppendDecorators(
			decor.Any(func(s decor.Statistics) string {
				counters := downloadCounters(s)
				if s.AvailableWidth < downloadMinBarWidth+2+ansi.StringWidth(counters) {
					return ""
				}
				return counters
			}),
		),
	)
}

func downloadCounters(s decor.Statistics) string {
	return fmt.Sprintf("% .1f / % .1f", decor.SizeB1024(s.Current), decor.SizeB1024(s.Total))
}

func (t *Text) finishBar(err error) {
	if t.bar == nil {
		return
	}
	if err != nil {
		t.bar.Abort(false)
	} else {
		t.bar.SetTotal(t.total, true)
	}
	t.progress.Wait()
	t.bar, t.progress = nil, nil
}
