package logging

import (
	"fmt"
	"log/slog"
	"strings"
)

// ParseLevel parses a textual log level ("debug", "info", "warn", "error")
// into a slog.Level. Unknown values fall back to info.
func ParseLevel(level string) (slog.Level, error) {
	if level == "" {
		return slog.LevelInfo, nil
	}

	var parsed slog.Level
	if err := parsed.UnmarshalText([]byte(strings.ToUpper(level))); err != nil {
		return slog.LevelInfo, fmt.Errorf("parse log level %q: %w", level, err)
	}

	return parsed, nil
}