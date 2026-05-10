package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// HomeSource indicates which mechanism resolved the active CJV_HOME.
// The zero value HomeSourceUnknown signals "not yet resolved" so an
// uninitialized value is never silently mistaken for a real source.
type HomeSource int

const (
	HomeSourceUnknown HomeSource = iota
	// HomeSourceDefault means the path fell back to <user-home>/.cjv.
	HomeSourceDefault
	// HomeSourcePersisted means the path came from settings.toml.
	HomeSourcePersisted
	// HomeSourceEnv means the CJV_HOME environment variable supplied the path.
	HomeSourceEnv
)

func (s HomeSource) String() string {
	switch s {
	case HomeSourceEnv:
		return "env"
	case HomeSourcePersisted:
		return "persisted"
	case HomeSourceDefault:
		return "default"
	default:
		return "unknown"
	}
}

// MarshalText emits the symbolic name so JSON consumers see a stable string
// even if the underlying iota order ever changes.
func (s HomeSource) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

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

// Home returns the CJV_HOME path. The resolution order is:
//  1. CJV_HOME environment variable (highest)
//  2. settings.home persisted in <user-home>/.cjv/settings.toml
//  3. <user-home>/.cjv (default)
func Home() (string, error) {
	path, _, err := ResolveHomeWithSource()
	return path, err
}

// ResolveHomeWithSource returns the CJV_HOME path along with the source that
// produced it. Useful for surfacing provenance (e.g. in `cjv show home`).
func ResolveHomeWithSource() (string, HomeSource, error) {
	if h := os.Getenv(EnvHome); h != "" {
		abs, err := filepath.Abs(h)
		if err != nil {
			return "", HomeSourceEnv, err
		}
		return abs, HomeSourceEnv, nil
	}
	if h := loadHomeFromSettings(); h != "" {
		abs, err := filepath.Abs(h)
		if err != nil {
			return "", HomeSourcePersisted, err
		}
		return abs, HomeSourcePersisted, nil
	}
	home, err := cachedUserHomeDir()
	if err != nil {
		return "", HomeSourceDefault, err
	}
	return filepath.Join(home, ".cjv"), HomeSourceDefault, nil
}

// loadHomeFromSettings reads settings.home from the persisted settings file.
// IO and parse errors are logged at warn level and reported as an empty path
// so the caller can fall back to defaults.
func loadHomeFromSettings() string {
	sf, err := DefaultSettingsFile()
	if err != nil {
		slog.Warn("failed to resolve settings file path while loading home", "error", err)
		return ""
	}
	s, err := sf.Load()
	if err != nil {
		slog.Warn("failed to load settings file while resolving home", "path", sf.Path(), "error", err)
		return ""
	}
	return s.Home
}

const (
	toolchainsSubdir = "toolchains"
	binSubdir        = "bin"
	downloadsSubdir  = "downloads"
	docsSubdir       = "docs"
	stdxSubdir       = "stdx"
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

// DocsDir returns the root directory holding per-toolchain documentation
// trees: <CJV_HOME>/docs/. Each toolchain's docs live in a subdirectory named
// after the toolchain (e.g. <CJV_HOME>/docs/lts-1.0.5/).
func DocsDir() (string, error) {
	h, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, docsSubdir), nil
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
	return filepath.Join(h, stdxSubdir), nil
}

// StdxDirFor returns the stdx directory for a specific toolchain.
func StdxDirFor(tcName string) (string, error) {
	root, err := StdxDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, tcName), nil
}

// SettingsPath returns the location of the persisted settings file. It is
// intentionally decoupled from Home(): settings.toml always lives at
// <user-home>/.cjv/settings.toml so that the home path itself can be
// persisted as a setting without creating a chicken-and-egg dependency.
func SettingsPath() (string, error) {
	h, err := cachedUserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".cjv", "settings.toml"), nil
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
