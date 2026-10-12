// Package config loads the service configuration from environment variables
// with kelseyhightower/envconfig.
//
// TODO(LASTMILE-019): mirror the full real Config blocks (Cloud, Handshake,
// Encryption, TSP, Auth, S3Export, ...) as those integrations are added.
package config

import (
	"github.com/kelseyhightower/envconfig"
)

// Config is the root configuration of last-mile-task.
type Config struct {
	AppName string `envconfig:"APP_NAME" default:"logistics-last-mile-task"`
	EnvName string `envconfig:"ENV_NAME" default:"local"`

	HTTP      HTTP
	GRPC      GRPC
	Lifecycle Lifecycle
	Log       Log
	Postgres  Postgres
	WMS       WMS
	Outbox    Outbox
	Mock      Mock
}

// HTTP holds the gateway binding.
type HTTP struct {
	Addr string `envconfig:"HTTP_ADDR" default:":8080"`
}

// GRPC holds the gRPC server binding.
type GRPC struct {
	Addr string `envconfig:"GRPC_ADDR" default:":8001"`
}

// Lifecycle holds the health probes binding.
type Lifecycle struct {
	Addr string `envconfig:"LIFECYCLE_ADDR" default:":8081"`
}

// Log holds logging settings.
type Log struct {
	Level string `envconfig:"LOG_LEVEL" default:"info"`
}

// Postgres holds the lmt database settings.
type Postgres struct {
	DSN string `envconfig:"POSTGRES_DSN" default:"postgres://postgres:postgres@localhost:5432/last_mile?sslmode=disable"`
}

// WMS holds the WMS exporter settings.
type WMS struct {
	BaseURL  string `envconfig:"WMS_BASE_URL" default:"http://127.0.0.1:9090"`
	Username string `envconfig:"WMS_USERNAME" default:""`
	Password string `envconfig:"WMS_PASSWORD" default:""`
}

// Outbox holds the outbox worker settings.
type Outbox struct {
	BatchSize    int           `envconfig:"OUTBOX_BATCH_SIZE" default:"20"`
	PollInterval string        `envconfig:"OUTBOX_POLL_INTERVAL" default:"1s"`
	MaxAttempts  int           `envconfig:"OUTBOX_MAX_ATTEMPTS" default:"5"`
	Enabled      bool          `envconfig:"OUTBOX_ENABLED" default:"true"`
}

// Mock enables the in-process mock mode (no external services).
type Mock struct {
	Enabled bool `envconfig:"MOCK_ENABLED" default:"true"`
}

// Load reads the configuration from environment variables.
func Load() (Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}