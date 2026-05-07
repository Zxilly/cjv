package cjverr

import (
	"fmt"
	"strconv"

	"github.com/Zxilly/cjv/internal/i18n"
)

// ToolchainNotInstalledError indicates the requested toolchain is not installed.
type ToolchainNotInstalledError struct {
	Name string
}

func (e *ToolchainNotInstalledError) Error() string {
	return i18n.T("ToolchainNotInstalled", i18n.MsgData{"Name": e.Name})
}

// ToolchainAlreadyInstalledError indicates the toolchain is already present.
type ToolchainAlreadyInstalledError struct {
	Name string
}

func (e *ToolchainAlreadyInstalledError) Error() string {
	return i18n.T("ToolchainAlreadyInstalled", i18n.MsgData{"Name": e.Name})
}

// VersionNotFoundError indicates a version was not found in any channel.
type VersionNotFoundError struct {
	Version string
}

func (e *VersionNotFoundError) Error() string {
	return i18n.T("VersionNotFound", i18n.MsgData{"Version": e.Version})
}

// VersionNotAvailableError indicates a version is not available for the platform.
type VersionNotAvailableError struct {
	Version  string
	Platform string
}

func (e *VersionNotAvailableError) Error() string {
	return i18n.T("VersionNotAvailable", i18n.MsgData{
		"Version":  e.Version,
		"Platform": e.Platform,
	})
}

// NoToolchainConfiguredError indicates no toolchain has been configured.
type NoToolchainConfiguredError struct{}

func (e *NoToolchainConfiguredError) Error() string {
	return i18n.T("NoToolchainConfigured", nil)
}

// UnknownToolError indicates an unrecognized tool name.
type UnknownToolError struct {
	Name string
}

func (e *UnknownToolError) Error() string {
	return i18n.T("UnknownTool", i18n.MsgData{"Name": e.Name})
}

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

// UnsupportedPlatformError indicates the platform is not supported.
type UnsupportedPlatformError struct {
	OS   string
	Arch string
}

func (e *UnsupportedPlatformError) Error() string {
	return i18n.T("UnsupportedPlatform", i18n.MsgData{"OS": e.OS, "Arch": e.Arch})
}

// RecursionLimitError indicates proxy recursion exceeded the maximum.
type RecursionLimitError struct {
	Max int
}

func (e *RecursionLimitError) Error() string {
	return i18n.T("RecursionLimitExceeded", i18n.MsgData{"Max": strconv.Itoa(e.Max)})
}

// UnknownChannelError indicates an unrecognized channel name.
type UnknownChannelError struct {
	Channel string
}

func (e *UnknownChannelError) Error() string {
	return i18n.T("UnknownChannel", i18n.MsgData{"Channel": e.Channel})
}

// GitCodeAPIKeyRequiredError indicates the GitCode API key is not configured.
type GitCodeAPIKeyRequiredError struct{}

func (e *GitCodeAPIKeyRequiredError) Error() string {
	return i18n.T("GitCodeAPIKeyRequired", nil)
}

// ExitCodeError carries a process exit code so callers can propagate it
// without calling os.Exit directly (which would skip deferred cleanup).
type ExitCodeError struct {
	Code int
}

func (e *ExitCodeError) Error() string {
	return fmt.Sprintf("process exited with code %d", e.Code)
}

// UnknownComponentError indicates an unrecognized component name.
type UnknownComponentError struct {
	Name string
}

func (e *UnknownComponentError) Error() string {
	return i18n.T("UnknownComponent", i18n.MsgData{"Name": e.Name})
}

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

// ComponentRequiresHostError indicates a component cannot be installed on a
// custom / linked toolchain.
type ComponentRequiresHostError struct {
	Component string
}

func (e *ComponentRequiresHostError) Error() string {
	return i18n.T("ComponentRequiresHost", i18n.MsgData{"Component": e.Component})
}

// DocsNotInstalledError indicates `cjv doc` was invoked on a toolchain that
// has neither docs nor stdx-docs installed (only meaningful for nightly).
type DocsNotInstalledError struct {
	Toolchain string
}

func (e *DocsNotInstalledError) Error() string {
	return i18n.T("DocsNotInstalled", i18n.MsgData{"Toolchain": e.Toolchain})
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
