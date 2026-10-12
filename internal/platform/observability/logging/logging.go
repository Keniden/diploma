// Package logging configures the process-wide slog logger.
package logging

import (
	"log/slog"
)

// Config holds logging settings shared with the rest of the platform.
type Config struct {
	Environment string
	AppVersion  string
	Level       slog.Level
}

// Init installs the default slog logger.
func Init(conf Config) {
	handler := NewSlogHandler(conf.Environment, conf.AppVersion, conf.Level)
	slog.SetDefault(slog.New(handler))
}