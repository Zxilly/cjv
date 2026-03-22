package logging

import (
	"log/slog"
	"testing"
)

func TestInit_DefaultLevel(t *testing.T) {
	t.Setenv("CJV_LOG", "")
	Init()
	if !slog.Default().Enabled(nil, slog.LevelWarn) {
		t.Error("default level should enable Warn")
	}
	if slog.Default().Enabled(nil, slog.LevelInfo) {
		t.Error("default level should not enable Info")
	}
}

func TestInit_DebugLevel(t *testing.T) {
	t.Setenv("CJV_LOG", "debug")
	Init()
	if !slog.Default().Enabled(nil, slog.LevelDebug) {
		t.Error("debug level should enable Debug")
	}
}

func TestInit_InvalidLevel(t *testing.T) {
	t.Setenv("CJV_LOG", "invalid")
	Init() // should not panic, fall back to warn
	if !slog.Default().Enabled(nil, slog.LevelWarn) {
		t.Error("invalid level should fall back to Warn")
	}
}
