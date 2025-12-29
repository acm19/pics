package logger

import (
	"log/slog"
	"os"
)

var log *slog.Logger

func init() {
	level := slog.LevelInfo
	if os.Getenv("DEBUG") != "" {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	log = slog.New(handler)
}

// Info logs at info level.
func Info(msg string, args ...any) {
	log.Info(msg, args...)
}

// Error logs at error level.
func Error(msg string, args ...any) {
	log.Error(msg, args...)
}

// Debug logs at debug level.
func Debug(msg string, args ...any) {
	log.Debug(msg, args...)
}

// Warn logs at warn level.
func Warn(msg string, args ...any) {
	log.Warn(msg, args...)
}
