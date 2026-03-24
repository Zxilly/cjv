package config

import (
	"os"
	"path/filepath"
	"sync"
)

const (
	EnvHome             = "CJV_HOME"
	EnvToolchain        = "CJV_TOOLCHAIN"
	EnvRecursionCount   = "CJV_RECURSION_COUNT"
	EnvLog              = "CJV_LOG"
	EnvLang             = "CJV_LANG"
	EnvMaxRetries       = "CJV_MAX_RETRIES"
	EnvDownloadTimeout  = "CJV_DOWNLOAD_TIMEOUT"
	EnvNoPathSetup      = "CJV_NO_PATH_SETUP"
	EnvGitCodeAPIKey    = "CJV_GITCODE_API_KEY"
	EnvFallbackSettings = "CJV_FALLBACK_SETTINGS"
)

var (
	userHomeDirOnce sync.Once
	userHomeDirVal  string
	errUserHomeDir  error
)

// cachedUserHomeDir returns the user's home directory, caching the result.
// Note: if os.UserHomeDir() fails on the first call, the error is cached
// permanently for the lifetime of the process. This is acceptable for a
// short-lived CLI tool.
func cachedUserHomeDir() (string, error) {
	userHomeDirOnce.Do(func() {
		userHomeDirVal, errUserHomeDir = os.UserHomeDir()
	})
	return userHomeDirVal, errUserHomeDir
}

// Home returns the CJV_HOME path. Uses CJV_HOME env var if set, otherwise defaults to ~/.cjv.
func Home() (string, error) {
	if h := os.Getenv(EnvHome); h != "" {
		return filepath.Abs(h)
	}
	home, err := cachedUserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cjv"), nil
}

const (
	toolchainsSubdir = "toolchains"
	binSubdir        = "bin"
	downloadsSubdir  = "downloads"
)

func ToolchainsDir() (string, error) {
	h, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, toolchainsSubdir), nil
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

func SettingsPath() (string, error) {
	h, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, "settings.toml"), nil
}

var (
	defaultSettingsFileMu sync.Mutex
	defaultSettingsFiles  = make(map[string]*SettingsFile)
)

// DefaultSettingsFile returns a SettingsFile backed by the user's
// settings.toml inside CJV_HOME. Instances are cached per resolved path,
// and their internal caches avoid redundant disk reads.
func DefaultSettingsFile() (*SettingsFile, error) {
	path, err := SettingsPath()
	if err != nil {
		return nil, err
	}
	defaultSettingsFileMu.Lock()
	defer defaultSettingsFileMu.Unlock()
	sf, ok := defaultSettingsFiles[path]
	if !ok {
		sf = NewSettingsFile(path)
		defaultSettingsFiles[path] = sf
	}
	return sf, nil
}

// ResetDefaultSettingsFileCache clears the cached SettingsFile instances.
// This should only be used in tests to ensure isolation between test cases.
func ResetDefaultSettingsFileCache() {
	defaultSettingsFileMu.Lock()
	defer defaultSettingsFileMu.Unlock()
	clear(defaultSettingsFiles)
}

// ResetCachedUserHomeDir resets the cached user home directory so that
// subsequent calls to cachedUserHomeDir() will re-evaluate os.UserHomeDir().
// This should only be used in tests to ensure isolation between test cases.
func ResetCachedUserHomeDir() {
	userHomeDirOnce = sync.Once{}
	userHomeDirVal = ""
	errUserHomeDir = nil
}

func EnsureDirs() error {
	home, err := Home()
	if err != nil {
		return err
	}
	for _, sub := range []string{toolchainsSubdir, binSubdir, downloadsSubdir} {
		if err := os.MkdirAll(filepath.Join(home, sub), 0o755); err != nil {
			return err
		}
	}
	return nil
}
