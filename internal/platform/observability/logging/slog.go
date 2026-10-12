package logging

import (
	"context"
	"log/slog"
	"os"
)

var _ slog.Handler = (*slogHandler)(nil)

// slogHandler decorates a JSON handler with the request id (and, once
// tracing is ported, trace/span ids) extracted from the record context.
type slogHandler struct {
	handler slog.Handler
}

// NewSlogHandler builds a JSON slog handler targeting stderr with
// environment/version context attributes.
func NewSlogHandler(
	env, version string,
	level slog.Level,
) slog.Handler {
	return &slogHandler{
		handler: slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		}).WithAttrs(
			[]slog.Attr{
				slog.String("environment", env),
				slog.String("app_version", version),
			},
		),
	}
}

// Enabled implements [slog.Handler].
func (s *slogHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return s.handler.Enabled(ctx, l)
}

// Handle implements [slog.Handler].
func (s *slogHandler) Handle(ctx context.Context, r slog.Record) error {
	AddRequestID(ctx, &r)
	// TODO(LASTMILE-092): add trace/span ids once tracing is ported (the real
	// repo calls AddTracingInfo here).

	return s.handler.Handle(ctx, r)
}

// WithAttrs implements [slog.Handler].
func (s *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return s.handler.WithAttrs(attrs)
}

// WithGroup implements [slog.Handler].
func (s *slogHandler) WithGroup(name string) slog.Handler {
	return s.handler.WithGroup(name)
}