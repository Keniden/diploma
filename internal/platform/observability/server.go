// Package observability exposes the lifecycle HTTP server of the service:
// /healthz (liveness) and /readyz (readiness) probes.
package observability

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

// Config configures the lifecycle server.
type Config struct {
	Addr    string
	Version string
	Logger  *slog.Logger
}

func (c *Config) defaults() {
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
}

// Server is the HTTP server serving the health probes. The real repository
// also exposes Prometheus metrics and OpenTelemetry tracing here; those
// parts are not ported into the skeleton. TODO(LASTMILE-092): add
// /metrics and trace propagation once metrics/tracing deps are allowed.
type Server struct {
	version string
	mux     *http.ServeMux
	server  *http.Server
	logger  *slog.Logger
	started time.Time

	ctx    context.Context
	cancel context.CancelFunc

	healthChecks    []healthCheck
	readinessChecks []readinessCheck
}

// NewServer builds a lifecycle HTTP server.
func NewServer(cfg Config) *Server {
	cfg.defaults()

	mux := http.NewServeMux()
	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		version: cfg.Version,
		mux:     mux,
		ctx:     ctx,
		cancel:  cancel,
		logger:  cfg.Logger,

		server: &http.Server{
			Addr:         cfg.Addr,
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  30 * time.Second,
		},

		healthChecks:    make([]healthCheck, 0),
		readinessChecks: make([]readinessCheck, 0),
	}
}

// ListenAndServe registers the probe routes and starts serving. It blocks
// until the server fails or Shutdown is called.
func (s *Server) ListenAndServe() error {
	s.started = time.Now()

	// The skeleton serves the k8s-conventional paths (/healthz, /readyz) and
	// the real repo paths (/health, /ready) on the same handlers.
	s.mux.HandleFunc("/healthz", s.handleHealth)
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/readyz", s.handleReady)
	s.mux.HandleFunc("/ready", s.handleReady)

	s.logger.Debug(
		"observability: http server started",
		"addr", s.server.Addr,
		"version", s.version,
	)

	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Debug(
		"observability: http server shutting down",
		"addr", s.server.Addr,
		"version", s.version,
	)

	s.cancel()
	return s.server.Shutdown(ctx)
}

// AddHealthCheck registers a synchronous liveness probe.
func (s *Server) AddHealthCheck(name string, check func() error) {
	s.healthChecks = append(s.healthChecks, healthCheck{
		name:  name,
		check: check,
	})
}

// AddReadinessCheck registers a background readiness probe refreshed every
// interval; nil probes are ignored.
func (s *Server) AddReadinessCheck(
	name string,
	interval,
	timeout time.Duration,
	ready func(ctx context.Context) error,
) {
	if ready == nil {
		return
	}

	rd := readinessCheck{
		name:   name,
		status: &atomic.Bool{},
	}

	s.readinessChecks = append(s.readinessChecks, rd)

	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				readyCtx, readyCancel := context.WithTimeout(context.Background(), timeout)
				rd.status.Store(ready(readyCtx) == nil)
				readyCancel()
			case <-s.ctx.Done():
				return
			}
		}
	}()
}

type response struct {
	Success bool           `json:"success"`
	Data    map[string]any `json:"data,omitempty"`
	Message string         `json:"message"`
}

func (s *Server) handleHealth(wr http.ResponseWriter, _ *http.Request) {
	var (
		status = true
		checks = make(map[string]any, len(s.healthChecks))
	)

	for _, hc := range s.healthChecks {
		checks[hc.name] = "OK"

		if err := hc.do(); err != nil {
			status = false
			checks[hc.name] = err.Error()
		}
	}

	code := http.StatusOK
	if !status {
		code = http.StatusInternalServerError
	}

	s.writeJSON(wr, code, response{
		Success: status,
		Data: map[string]any{
			"status":  boolString(status, "OK", "ERROR"),
			"version": s.version,
			"checks":  checks,
			"uptime":  time.Since(s.started).Seconds(),
		},
		Message: boolString(status, "service is healthy", "service is unhealthy"),
	})
}

func (s *Server) handleReady(wr http.ResponseWriter, _ *http.Request) {
	var (
		ready      = true
		readieries = make(map[string]any, len(s.readinessChecks))
	)

	for _, rd := range s.readinessChecks {
		readieries[rd.name] = "READY"

		if !rd.status.Load() {
			ready = false
			readieries[rd.name] = "NOT READY"
		}
	}

	code := http.StatusOK
	if !ready {
		code = http.StatusServiceUnavailable
	}

	s.writeJSON(wr, code, response{
		Success: ready,
		Data: map[string]any{
			"ready":      ready,
			"readieries": readieries,
		},
		Message: boolString(ready, "service is ready", "service is unavailable"),
	})
}

func (s *Server) writeJSON(wr http.ResponseWriter, code int, res response) {
	wr.Header().Set("Content-Type", "application/json")
	wr.WriteHeader(code)

	if err := json.NewEncoder(wr).Encode(res); err != nil {
		http.Error(wr, err.Error(), http.StatusInternalServerError)
	}
}

func boolString(v bool, ifTrue, ifFalse string) string {
	if v {
		return ifTrue
	}
	return ifFalse
}