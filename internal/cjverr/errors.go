package cjverr

import (
	"fmt"
	"strconv"

	"github.com/Zxilly/cjv/internal/i18n"
)

// ErrorCode is a stable machine-readable error identifier.
type ErrorCode string

const (
	ErrorCodeUnknown                         ErrorCode = "UNKNOWN"
	ErrorCodeToolchainNotInstalled           ErrorCode = "TOOLCHAIN_NOT_INSTALLED"
	ErrorCodeToolchainAlreadyInstalled       ErrorCode = "TOOLCHAIN_ALREADY_INSTALLED"
	ErrorCodeVersionNotFound                 ErrorCode = "VERSION_NOT_FOUND"
	ErrorCodeVersionNotAvailable             ErrorCode = "VERSION_NOT_AVAILABLE"
	ErrorCodeNoToolchainConfigured           ErrorCode = "NO_TOOLCHAIN_CONFIGURED"
	ErrorCodeUnknownTool                     ErrorCode = "UNKNOWN_TOOL"
	ErrorCodeToolNotInToolchain              ErrorCode = "TOOL_NOT_IN_TOOLCHAIN"
	ErrorCodeChecksumMismatch                ErrorCode = "CHECKSUM_MISMATCH"
	ErrorCodeUnsupportedPlatform             ErrorCode = "UNSUPPORTED_PLATFORM"
	ErrorCodeRecursionLimitExceeded          ErrorCode = "RECURSION_LIMIT_EXCEEDED"
	ErrorCodeUnknownChannel                  ErrorCode = "UNKNOWN_CHANNEL"
	ErrorCodeGitCodeAPIKeyRequired           ErrorCode = "GITCODE_API_KEY_REQUIRED"
	ErrorCodeUnsupportedForJSON              ErrorCode = "UNSUPPORTED_FOR_JSON"
	ErrorCodeUnknownComponent                ErrorCode = "UNKNOWN_COMPONENT"
	ErrorCodeComponentNotInstalled           ErrorCode = "COMPONENT_NOT_INSTALLED"
	ErrorCodeComponentAlreadyInstalled       ErrorCode = "COMPONENT_ALREADY_INSTALLED"
	ErrorCodeComponentNotAvailableForChannel ErrorCode = "COMPONENT_NOT_AVAILABLE_FOR_CHANNEL"
	ErrorCodeComponentNotPublished           ErrorCode = "COMPONENT_NOT_PUBLISHED"
	ErrorCodeComponentRequiresHost           ErrorCode = "COMPONENT_REQUIRES_HOST"
	ErrorCodeComponentLinkNotSupported       ErrorCode = "COMPONENT_LINK_NOT_SUPPORTED"
	ErrorCodeComponentLinkInvalidPath        ErrorCode = "COMPONENT_LINK_INVALID_PATH"
	ErrorCodeDocsNotInstalled                ErrorCode = "DOCS_NOT_INSTALLED"
	ErrorCodeDocsTopicNotFound               ErrorCode = "DOCS_TOPIC_NOT_FOUND"
)

// Coded is implemented by errors that carry a stable machine-readable code
// and a structured details map. The output renderer uses this to build the
// JSON error envelope without knowing about specific error types.
type Coded interface {
	error
	Code() ErrorCode
	Details() map[string]any
}

// ToolchainNotInstalledError indicates the requested toolchain is not installed.
type ToolchainNotInstalledError struct {
	Name string
}

func (e *ToolchainNotInstalledError) Error() string {
	return i18n.T("ToolchainNotInstalled", i18n.MsgData{"Name": e.Name})
}
func (e *ToolchainNotInstalledError) Code() ErrorCode         { return ErrorCodeToolchainNotInstalled }
func (e *ToolchainNotInstalledError) Details() map[string]any { return map[string]any{"name": e.Name} }

// ToolchainAlreadyInstalledError indicates the toolchain is already present.
type ToolchainAlreadyInstalledError struct {
	Name string
}

func (e *ToolchainAlreadyInstalledError) Error() string {
	return i18n.T("ToolchainAlreadyInstalled", i18n.MsgData{"Name": e.Name})
}
func (e *ToolchainAlreadyInstalledError) Code() ErrorCode { return ErrorCodeToolchainAlreadyInstalled }
func (e *ToolchainAlreadyInstalledError) Details() map[string]any {
	return map[string]any{"name": e.Name}
}

// VersionNotFoundError indicates a version was not found in any channel.
type VersionNotFoundError struct {
	Version string
}

func (e *VersionNotFoundError) Error() string {
	return i18n.T("VersionNotFound", i18n.MsgData{"Version": e.Version})
}
func (e *VersionNotFoundError) Code() ErrorCode         { return ErrorCodeVersionNotFound }
func (e *VersionNotFoundError) Details() map[string]any { return map[string]any{"version": e.Version} }

// VersionNotAvailableError indicates a version is not available for the platform.
type VersionNotAvailableError struct {
	Version string
	Target  string
}

func (e *VersionNotAvailableError) Error() string {
	return i18n.T("VersionNotAvailable", i18n.MsgData{
		"Version": e.Version,
		"Target":  e.Target,
	})
}
func (e *VersionNotAvailableError) Code() ErrorCode { return ErrorCodeVersionNotAvailable }
func (e *VersionNotAvailableError) Details() map[string]any {
	return map[string]any{"version": e.Version, "target": e.Target}
}

// NoToolchainConfiguredError indicates no toolchain has been configured.
type NoToolchainConfiguredError struct{}

func (e *NoToolchainConfiguredError) Error() string {
	return i18n.T("NoToolchainConfigured", nil)
}
func (e *NoToolchainConfiguredError) Code() ErrorCode         { return ErrorCodeNoToolchainConfigured }
func (e *NoToolchainConfiguredError) Details() map[string]any { return map[string]any{} }

// UnknownToolError indicates an unrecognized tool name.
type UnknownToolError struct {
	Name string
}

func (e *UnknownToolError) Error() string {
	return i18n.T("UnknownTool", i18n.MsgData{"Name": e.Name})
}
func (e *UnknownToolError) Code() ErrorCode         { return ErrorCodeUnknownTool }
func (e *UnknownToolError) Details() map[string]any { return map[string]any{"name": e.Name} }

// ToolNotInToolchainError indicates the requested proxy tool is not present in
// the resolved toolchain installation.
type ToolNotInToolchainError struct {
	Tool string
	Path string
}

func (e *ToolNotInToolchainError) Error() string {
	return i18n.T("ToolNotInToolchain", i18n.MsgData{
		"Tool": e.Tool,
		"Path": e.Path,
	})
}
func (e *ToolNotInToolchainError) Code() ErrorCode { return ErrorCodeToolNotInToolchain }
func (e *ToolNotInToolchainError) Details() map[string]any {
	return map[string]any{"tool": e.Tool, "path": e.Path}
}

// ChecksumMismatchError indicates a download checksum verification failure.
type ChecksumMismatchError struct {
	Expected string
	Actual   string
}

func (e *ChecksumMismatchError) Error() string {
	return i18n.T("ChecksumMismatch", i18n.MsgData{
		"Expected": e.Expected,
		"Actual":   e.Actual,
	})
}
func (e *ChecksumMismatchError) Code() ErrorCode { return ErrorCodeChecksumMismatch }
func (e *ChecksumMismatchError) Details() map[string]any {
	return map[string]any{"expected": e.Expected, "actual": e.Actual}
}

// UnsupportedPlatformError indicates the platform is not supported.
type UnsupportedPlatformError struct {
	OS   string
	Arch string
}

func (e *UnsupportedPlatformError) Error() string {
	return i18n.T("UnsupportedPlatform", i18n.MsgData{"OS": e.OS, "Arch": e.Arch})
}
func (e *UnsupportedPlatformError) Code() ErrorCode { return ErrorCodeUnsupportedPlatform }
func (e *UnsupportedPlatformError) Details() map[string]any {
	return map[string]any{"os": e.OS, "arch": e.Arch}
}

// RecursionLimitError indicates proxy recursion exceeded the maximum.
type RecursionLimitError struct {
	Max int
}

func (e *RecursionLimitError) Error() string {
	return i18n.T("RecursionLimitExceeded", i18n.MsgData{"Max": strconv.Itoa(e.Max)})
}
func (e *RecursionLimitError) Code() ErrorCode         { return ErrorCodeRecursionLimitExceeded }
func (e *RecursionLimitError) Details() map[string]any { return map[string]any{"max": e.Max} }

// UnknownChannelError indicates an unrecognized channel name.
type UnknownChannelError struct {
	Channel string
}

func (e *UnknownChannelError) Error() string {
	return i18n.T("UnknownChannel", i18n.MsgData{"Channel": e.Channel})
}
func (e *UnknownChannelError) Code() ErrorCode         { return ErrorCodeUnknownChannel }
func (e *UnknownChannelError) Details() map[string]any { return map[string]any{"channel": e.Channel} }

// GitCodeAPIKeyRequiredError indicates the GitCode API key is not configured.
type GitCodeAPIKeyRequiredError struct{}

func (e *GitCodeAPIKeyRequiredError) Error() string {
	return i18n.T("GitCodeAPIKeyRequired", nil)
}
func (e *GitCodeAPIKeyRequiredError) Code() ErrorCode         { return ErrorCodeGitCodeAPIKeyRequired }
func (e *GitCodeAPIKeyRequiredError) Details() map[string]any { return map[string]any{} }

// ExitCodeError carries a process exit code so callers can propagate it
// without calling os.Exit directly (which would skip deferred cleanup).
//
// ExitCodeError intentionally does NOT implement Coded: it is a transparent
// wrapper for child-process exit codes and carries no semantic information
// for an end user. Renderers should pass it through without writing an envelope.
type ExitCodeError struct {
	Code int
}

func (e *ExitCodeError) Error() string {
	return fmt.Sprintf("process exited with code %d", e.Code)
}

// UnsupportedForJSONError indicates the requested command cannot produce
// machine-readable JSON output (e.g. shell-script emitters, interactive
// installers, or transparent process proxies).
type UnsupportedForJSONError struct {
	Command string
}

func (e *UnsupportedForJSONError) Error() string {
	return i18n.T("UnsupportedForJSON", i18n.MsgData{"Command": e.Command})
}
func (e *UnsupportedForJSONError) Code() ErrorCode { return ErrorCodeUnsupportedForJSON }
func (e *UnsupportedForJSONError) Details() map[string]any {
	return map[string]any{"command": e.Command}
}

// UnknownComponentError indicates an unrecognized component name.
type UnknownComponentError struct {
	Name string
}

func (e *UnknownComponentError) Error() string {
	return i18n.T("UnknownComponent", i18n.MsgData{"Name": e.Name})
}
func (e *UnknownComponentError) Code() ErrorCode         { return ErrorCodeUnknownComponent }
func (e *UnknownComponentError) Details() map[string]any { return map[string]any{"name": e.Name} }

// ComponentNotInstalledError indicates the requested component is missing
// from a toolchain.
type ComponentNotInstalledError struct {
	Toolchain string
	Component string
}

func (e *ComponentNotInstalledError) Error() string {
	return i18n.T("ComponentNotInstalled", i18n.MsgData{
		"Toolchain": e.Toolchain,
		"Component": e.Component,
	})
}
func (e *ComponentNotInstalledError) Code() ErrorCode { return ErrorCodeComponentNotInstalled }
func (e *ComponentNotInstalledError) Details() map[string]any {
	return map[string]any{"toolchain": e.Toolchain, "component": e.Component}
}

// ComponentAlreadyInstalledError indicates the component is already present
// on the toolchain and --force was not supplied.
type ComponentAlreadyInstalledError struct {
	Toolchain string
	Component string
}

func (e *ComponentAlreadyInstalledError) Error() string {
	return i18n.T("ComponentAlreadyInstalled", i18n.MsgData{
		"Toolchain": e.Toolchain,
		"Component": e.Component,
	})
}
func (e *ComponentAlreadyInstalledError) Code() ErrorCode { return ErrorCodeComponentAlreadyInstalled }
func (e *ComponentAlreadyInstalledError) Details() map[string]any {
	return map[string]any{"toolchain": e.Toolchain, "component": e.Component}
}

// ComponentNotAvailableForChannelError indicates the component cannot be
// installed offline on the given channel (e.g. docs / stdx-docs on lts/sts).
type ComponentNotAvailableForChannelError struct {
	Component string
	Channel   string
}

func (e *ComponentNotAvailableForChannelError) Error() string {
	return i18n.T("ComponentNotAvailableForChannel", i18n.MsgData{
		"Component": e.Component,
		"Channel":   e.Channel,
	})
}
func (e *ComponentNotAvailableForChannelError) Code() ErrorCode {
	return ErrorCodeComponentNotAvailableForChannel
}
func (e *ComponentNotAvailableForChannelError) Details() map[string]any {
	return map[string]any{"component": e.Component, "channel": e.Channel}
}

// ComponentNotPublishedError indicates the version manifest carries no download
// link for the requested component — either the version ships no components at
// all, or (for stdx) not for the requested platform.
type ComponentNotPublishedError struct {
	Component string
	Version   string
	Target    string // stdx archive platform token; empty for docs / stdx-docs
}

func (e *ComponentNotPublishedError) Error() string {
	if e.Target != "" {
		return i18n.T("ComponentNotPublishedForTarget", i18n.MsgData{
			"Component": e.Component,
			"Version":   e.Version,
			"Target":    e.Target,
		})
	}
	return i18n.T("ComponentNotPublished", i18n.MsgData{
		"Component": e.Component,
		"Version":   e.Version,
	})
}
func (e *ComponentNotPublishedError) Code() ErrorCode { return ErrorCodeComponentNotPublished }
func (e *ComponentNotPublishedError) Details() map[string]any {
	return map[string]any{"component": e.Component, "version": e.Version, "target": e.Target}
}

// ComponentRequiresHostError indicates a component cannot be installed on a
// custom / linked toolchain.
type ComponentRequiresHostError struct {
	Component string
}

func (e *ComponentRequiresHostError) Error() string {
	return i18n.T("ComponentRequiresHost", i18n.MsgData{"Component": e.Component})
}
func (e *ComponentRequiresHostError) Code() ErrorCode { return ErrorCodeComponentRequiresHost }
func (e *ComponentRequiresHostError) Details() map[string]any {
	return map[string]any{"component": e.Component}
}

// ComponentLinkNotSupportedError indicates the user invoked `cjv component
// link` on a component that does not support filesystem linking. Only stdx
// currently supports link; docs / stdx-docs must use `cjv component add`.
type ComponentLinkNotSupportedError struct {
	Component string
}

func (e *ComponentLinkNotSupportedError) Error() string {
	return i18n.T("ComponentLinkNotSupported", i18n.MsgData{"Component": e.Component})
}
func (e *ComponentLinkNotSupportedError) Code() ErrorCode {
	return ErrorCodeComponentLinkNotSupported
}
func (e *ComponentLinkNotSupportedError) Details() map[string]any {
	return map[string]any{"component": e.Component}
}

// ComponentLinkInvalidPathError indicates the user-supplied stdx source path
// failed structural validation (missing, not a directory, or missing the
// required dynamic / static subdirectories).
type ComponentLinkInvalidPathError struct {
	Reason string
}

func (e *ComponentLinkInvalidPathError) Error() string {
	return i18n.T("ComponentLinkInvalidPath", i18n.MsgData{"Reason": e.Reason})
}
func (e *ComponentLinkInvalidPathError) Code() ErrorCode {
	return ErrorCodeComponentLinkInvalidPath
}
func (e *ComponentLinkInvalidPathError) Details() map[string]any {
	return map[string]any{"reason": e.Reason}
}

// DocsNotInstalledError indicates `cjv doc` was invoked on a toolchain that
// has neither docs nor stdx-docs installed (only meaningful for nightly).
type DocsNotInstalledError struct {
	Toolchain string
}

func (e *DocsNotInstalledError) Error() string {
	return i18n.T("DocsNotInstalled", i18n.MsgData{"Toolchain": e.Toolchain})
}
func (e *DocsNotInstalledError) Code() ErrorCode { return ErrorCodeDocsNotInstalled }
func (e *DocsNotInstalledError) Details() map[string]any {
	return map[string]any{"toolchain": e.Toolchain}
}

// DocsTopicNotFoundError indicates `cjv doc <topic>` could not resolve the
// topic to a file under the toolchain's docs root.
type DocsTopicNotFoundError struct {
	Toolchain        string
	Topic            string
	MissingComponent string
}

func (e *DocsTopicNotFoundError) Error() string {
	if e.MissingComponent == "" {
		return i18n.T("DocsTopicNotFoundNoHint", i18n.MsgData{
			"Toolchain": e.Toolchain,
			"Topic":     e.Topic,
		})
	}
	return i18n.T("DocsTopicNotFound", i18n.MsgData{
		"Toolchain": e.Toolchain,
		"Topic":     e.Topic,
		"Component": e.MissingComponent,
	})
}
func (e *DocsTopicNotFoundError) Code() ErrorCode { return ErrorCodeDocsTopicNotFound }
func (e *DocsTopicNotFoundError) Details() map[string]any {
	return map[string]any{
		"toolchain":         e.Toolchain,
		"topic":             e.Topic,
		"missing_component": e.MissingComponent,
	}
}
