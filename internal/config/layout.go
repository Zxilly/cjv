package config

import (
	"path/filepath"
	"strings"
)

// CJV_HOME 的目录布局。这是唯一拼写 CJV_HOME 下各子目录名，以及安装过程留在
// 工具链旁的临时树（staging 树、旧式备份、fstx 事务目录）命名规则的地方；
// fstx、toolchain、component、lifecycle 都从这里派生这些路径，而不是自己拼后缀。
const (
	// ToolchainsSubdir holds one SDK tree per installed toolchain.
	ToolchainsSubdir = "toolchains"
	// DocsSubdir holds one documentation tree per toolchain.
	DocsSubdir = "docs"
	// StdxSubdir holds one stdx tree per toolchain.
	StdxSubdir      = "stdx"
	binSubdir       = "bin"
	downloadsSubdir = "downloads"
)

const (
	// StagingSuffix marks a toolchain tree that is still being materialized
	// beside its destination. Callers derive the path with StagingDir.
	StagingSuffix = ".staging"
	// BackupSuffix marks legacy backups taken by force-installs before
	// transactions existed. They carry no commit marker.
	BackupSuffix = ".old"
	// TxTempPrefix marks fstx transaction directories, which hold the journal
	// and backups of one in-progress or interrupted change.
	TxTempPrefix = ".fstx-"
)

// StagingDir returns the staging tree that an install materializes before
// swapping it into dest. It is a sibling of dest, so a transaction on dest
// also owns it.
func StagingDir(dest string) string { return dest + StagingSuffix }

// IsScratchName reports whether a directory name under toolchains/ is
// install residue (staging, legacy backup or transaction directory) rather
// than an installed toolchain.
func IsScratchName(name string) bool {
	return strings.HasSuffix(name, StagingSuffix) ||
		strings.HasSuffix(name, BackupSuffix) ||
		strings.HasPrefix(name, TxTempPrefix)
}

func ToolchainsDir() (string, error) {
	h, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ToolchainsSubdir), nil
}

// ToolchainDirFor returns the SDK directory for a specific toolchain.
func ToolchainDirFor(tcName string) (string, error) {
	root, err := ToolchainsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, tcName), nil
}

func BinDir() (string, error) {
	h, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, binSubdir), nil
}

func DownloadsDir() (string, error) {
	h, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, downloadsSubdir), nil
}

// DocsDir returns the root directory holding per-toolchain documentation
// trees: <CJV_HOME>/docs/. Each toolchain's docs live in a subdirectory named
// after the toolchain (e.g. <CJV_HOME>/docs/lts-1.0.5/).
func DocsDir() (string, error) {
	h, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, DocsSubdir), nil
}

// DocsDirFor returns the documentation directory for a specific toolchain.
func DocsDirFor(tcName string) (string, error) {
	root, err := DocsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, tcName), nil
}

// StdxDir returns the root directory holding per-toolchain stdx trees:
// <CJV_HOME>/stdx/. Each toolchain's stdx files (dynamic/, static/) live
// directly under the toolchain-named subdir below.
func StdxDir() (string, error) {
	h, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, StdxSubdir), nil
}

// StdxDirFor returns the stdx directory for a specific toolchain.
func StdxDirFor(tcName string) (string, error) {
	root, err := StdxDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, tcName), nil
}
