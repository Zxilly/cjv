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

