package app

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

const (
	envLogLevel = "RABBIT_CODE_LOG_LEVEL"
	envLogFile  = "RABBIT_CODE_LOG_FILE"
	envDebug    = "RABBIT_CODE_DEBUG"
)

// NewLogger builds a slog.Logger for Phase 1: stderr JSON or text, optional file sink.
func NewLogger() (*slog.Logger, func(), error) {
	level := parseLevel(os.Getenv(envLogLevel))
	if truthy(os.Getenv(envDebug)) {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{Level: level}
	handler := slog.NewTextHandler(os.Stderr, opts)

	var closers []func()
	path := strings.TrimSpace(os.Getenv(envLogFile))
	if path != "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return nil, nil, err
		}
		closers = append(closers, func() { _ = f.Close() })
		mw := io.MultiWriter(os.Stderr, f)
		handler = slog.NewTextHandler(mw, opts)
	}

	log := slog.New(handler)
	closeAll := func() {
		for i := len(closers) - 1; i >= 0; i-- {
			closers[i]()
		}
	}
	return log, closeAll, nil
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
