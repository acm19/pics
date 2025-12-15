package main

import (
	"log/slog"
	"os"
)

var logger *slog.Logger

func init() {
	level := slog.LevelInfo
	if os.Getenv("DEBUG") != "" {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	logger = slog.New(handler)
}
