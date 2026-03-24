package logging

import (
	"log/slog"
	"os"
	"strings"

	"github.com/Zxilly/cjv/internal/config"
)

// Init configures the global slog logger based on the CJV_LOG environment variable.
// Valid values: debug, info, warn, error. Default: warn.
func Init() {
	level := parseLevel(os.Getenv(config.EnvLog))
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}
