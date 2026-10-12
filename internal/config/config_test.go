package config_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"last-mile-task/internal/config"
)

// helperProcessEnv marks the re-executed test binary as the defaults helper.
// envconfig reads the real process environment, so the defaults are asserted
// in a child process whose environment is scrubbed of every config variable.
const helperProcessEnv = "LMT_CONFIG_DEFAULTS_HELPER"

func TestLoad_Defaults(t *testing.T) {
	if os.Getenv(helperProcessEnv) == "1" {
		cfg, err := config.Load()
		require.NoError(t, err)

		require.Equal(t, "logistics-last-mile-task", cfg.AppName)
		require.Equal(t, "local", cfg.EnvName)
		require.Equal(t, ":8080", cfg.HTTP.Addr)
		require.Equal(t, ":8001", cfg.GRPC.Addr)
		require.Equal(t, ":8081", cfg.Lifecycle.Addr)
		require.Equal(t, "info", cfg.Log.Level)
		require.Equal(t, "postgres://postgres:postgres@localhost:5432/last_mile?sslmode=disable", cfg.Postgres.DSN)
		require.Equal(t, "http://127.0.0.1:9090", cfg.WMS.BaseURL)
		require.Equal(t, "", cfg.WMS.Username)
		require.Equal(t, "", cfg.WMS.Password)
		require.Equal(t, 20, cfg.Outbox.BatchSize)
		require.Equal(t, "1s", cfg.Outbox.PollInterval)
		require.Equal(t, 5, cfg.Outbox.MaxAttempts)
		require.True(t, cfg.Outbox.Enabled)
		require.True(t, cfg.Mock.Enabled)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestLoad_Defaults$")
	cmd.Env = []string{helperProcessEnv + "=1"}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("defaults helper failed: %v\n%s", err, out)
	}
}

func TestLoad_Overrides(t *testing.T) {
	t.Setenv("APP_NAME", "custom-app")
	t.Setenv("GRPC_ADDR", ":9999")
	t.Setenv("MOCK_ENABLED", "false")
	t.Setenv("OUTBOX_MAX_ATTEMPTS", "7")

	cfg, err := config.Load()
	require.NoError(t, err)

	require.Equal(t, "custom-app", cfg.AppName)
	require.Equal(t, ":9999", cfg.GRPC.Addr)
	require.False(t, cfg.Mock.Enabled)
	require.Equal(t, 7, cfg.Outbox.MaxAttempts)
}