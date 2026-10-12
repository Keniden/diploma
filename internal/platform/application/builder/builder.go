// Package builder boots the long-lived dependencies of the service: config,
// logging, database pool, repository and outbox adapter.
package builder

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"last-mile-task/internal/config"
	"last-mile-task/internal/platform/database"
	"last-mile-task/internal/platform/observability/logging"
	"last-mile-task/pkg/outbox"
)

// Builder is the dependency container of the service.
type Builder struct {
	ctx     context.Context
	cancel  context.CancelFunc
	conf    *config.Config
	version string

	pool *pgxpool.Pool

	mu         sync.Mutex
	repo       *database.Repository
	outbox     outbox.Querier
	shutdownFn []func()
	closeOnce  sync.Once
}

// New loads the configuration, initializes the logger and creates the
// builder context.
func New(ctx context.Context, version string) (*Builder, error) {
	conf, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("builder: load config: %w", err)
	}

	level, err := logging.ParseLevel(conf.Log.Level)
	if err != nil {
		return nil, fmt.Errorf("builder: parse log level: %w", err)
	}

	logging.Init(logging.Config{
		Environment: conf.EnvName,
		Level:       level,
		AppVersion:  version,
	})

	ctx, cancel := context.WithCancel(ctx)

	return &Builder{
		ctx:     ctx,
		cancel:  cancel,
		conf:    &conf,
		version: version,
	}, nil
}

// Config returns the service configuration.
func (b *Builder) Config() *config.Config {
	return b.conf
}

// Context returns the builder context, canceled on Shutdown.
func (b *Builder) Context() context.Context {
	return b.ctx
}

// Version returns the build version the service was started with.
func (b *Builder) Version() string {
	return b.version
}

// StartBackground runs fn with the builder context.
func (b *Builder) StartBackground(fn func(ctx context.Context)) {
	go fn(b.ctx)
}

// OnShutdown registers a hook called once from Shutdown before the context
// is canceled (e.g. closing the mock server / the db pool).
func (b *Builder) OnShutdown(fn func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.shutdownFn = append(b.shutdownFn, fn)
}

// Ping verifies database connectivity (used by the health/readiness probes).
func (b *Builder) Ping(ctx context.Context) error {
	db, err := b.DB()
	if err != nil {
		return err
	}

	return db.Ping(ctx)
}

// Shutdown runs the registered hooks in reverse order, closes the db pool
// and cancels the builder context. Safe to call multiple times.
func (b *Builder) Shutdown() {
	b.closeOnce.Do(func() {
		b.mu.Lock()
		fns := b.shutdownFn
		b.shutdownFn = nil
		b.mu.Unlock()

		for i := len(fns) - 1; i >= 0; i-- {
			fns[i]()
		}

		if b.pool != nil {
			b.pool.Close()
		}

		b.cancel()
	})
}
