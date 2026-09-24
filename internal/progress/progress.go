// Package progress is the seam between long-running operations and the
// presentation of their progress. Operations in lifecycle, resolve, dist and
// component emit typed events to one Sink; the adapter behind the Sink decides
// what, if anything, the user sees. Two adapters exist: Text renders i18n
// messages and the download bar, Discard renders nothing (JSON mode, tests).
package progress

// Kind identifies what happened. The data an adapter needs to render a kind
// travels in the Event fields documented next to it.
type Kind int

const (
	// FetchingManifest is emitted once per operation before the manifest is read.
	FetchingManifest Kind = iota + 1
	// NightlyNoChecksum: a nightly release has neither manifest nor sidecar SHA256.
	NightlyNoChecksum
	// ToolchainAlreadyInstalled: Toolchain is installed and force was not set.
	ToolchainAlreadyInstalled
	// Extracting: the downloaded archive is being unpacked.
	Extracting
	// ToolchainInstalled: Toolchain is in place.
	ToolchainInstalled
	// ComponentAlreadyInstalled: Component is installed for Toolchain and force was not set.
	ComponentAlreadyInstalled
	// ComponentInstalled: Component is in place for Toolchain.
	ComponentInstalled
	// FetchingComponent: Component for Toolchain is being downloaded.
	FetchingComponent
	// InstallingComponent: Component is being unpacked; Toolchain may be empty.
	InstallingComponent
	// RemovingComponent: Component is being removed.
	RemovingComponent
	// AlreadyUpToDate: Toolchain is the newest release of its channel.
	AlreadyUpToDate
	// UpdateFound: Toolchain is replaced by Replacement.
	UpdateFound
	// LinkDownloadingURL: a toolchain link downloads Subject (a URL).
	LinkDownloadingURL
	// LinkUsingArchive: a toolchain link unpacks Subject (a local archive path).
	LinkUsingArchive
	// LinkInstallingStdx: the stdx bundled with the linked Toolchain is installed.
	LinkInstallingStdx
	// AutoInstalling: the proxy path installs Subject (names joined with ", ").
	AutoInstalling
	// AutoInstallFailed: installing Subject failed with Err.
	AutoInstallFailed
	// DownloadStarted: a transfer labelled Subject begins; Bytes are already on
	// disk from an earlier attempt and Total is the expected size, or -1 when
	// the server did not say.
	DownloadStarted
	// DownloadAdvanced: Bytes more were received since the previous event.
	DownloadAdvanced
	// DownloadFinished: the transfer ended, with Err when it failed.
	DownloadFinished
)

// Event is one progress notification. Only the fields the Kind documents are
// set.
type Event struct {
	Kind        Kind
	Toolchain   string
	Component   string
	Replacement string
	Subject     string
	Err         error
	Bytes       int64
	Total       int64
}

// Sink receives progress events from an operation. Events arrive on the
// goroutine that runs the operation, in the order they happen.
type Sink interface {
	Report(Event)
}

// Discard drops every event. It is the JSON-mode adapter and the default for
// library callers that pass no sink.
var Discard Sink = discard{}

type discard struct{}

func (discard) Report(Event) {}

// Or returns sink, or Discard when sink is nil, so emitters never nil-check.
func Or(sink Sink) Sink {
	if sink == nil {
		return Discard
	}
	return sink
}
