package app

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
)

const envSlowLogFile = "RABBIT_CODE_SLOW_LOG_FILE"

// NewSlowLogger builds an optional logger for SLOW_OPERATION_LOGGING (stderr and/or dedicated file).
// Returns (nil, noop, nil) when the feature is off.
func NewSlowLogger() (*slog.Logger, func(), error) {
	if !features.SlowOperationLoggingEnabled() {
		return nil, func() {}, nil
	}
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.MessageKey {
				return slog.String(a.Key, "[slow] "+a.Value.String())
			}
			return a
		},
	}
	var closers []func()
	path := strings.TrimSpace(os.Getenv(envSlowLogFile))
	handler := slog.NewTextHandler(os.Stderr, opts)
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

// LogBootstrapPrefetchSlow logs prefetch duration to the slow-op logger when enabled.
func LogBootstrapPrefetchSlow(slow *slog.Logger, d time.Duration) {
	if slow == nil {
		return
	}
	slow.Info("bootstrap parallel prefetch", "duration", d.String())
}
