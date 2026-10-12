package builder

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"last-mile-task/internal/platform/database"
	"last-mile-task/pkg/outbox"
)

// DB lazily opens the pgx pool from config.Postgres.DSN and verifies the
// connectivity with a short ping. The real repository builds the DSN from
// individual Postgres fields and registers metrics collectors on the pool;
// the skeleton's config has a single DSN value. TODO(LASTMILE-093): add
// metrics collectors once metrics deps are allowed.
func (b *Builder) DB() (*pgxpool.Pool, error) {
	if b.pool != nil {
		return b.pool, nil
	}

	db, err := pgxpool.New(b.ctx, b.conf.Postgres.DSN)
	if err != nil {
		return nil, fmt.Errorf("builder: connect to postgres: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(b.ctx, 5*time.Second)
	defer cancel()

	if err := db.Ping(pingCtx); err != nil {
		// The pool is kept: connectivity problems surface via the readiness
		// probe / pkg/outbox errors instead of failing the boot.
		slog.Warn(
			"builder: db ping failed",
			"error", err,
		)
	} else {
		slog.Info("builder: db connected")
	}

	b.pool = db

	return db, nil
}

// Repository lazily builds the shared repository over the db pool.
func (b *Builder) Repository() (*database.Repository, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.repo != nil {
		return b.repo, nil
	}

	db, err := b.DB()
	if err != nil {
		return nil, err
	}

	b.repo = database.NewRepository(db)

	return b.repo, nil
}

// Outbox lazily builds the outbox querier over the shared repository (the
// same tx/context handling as the rest of the persistence layer).
func (b *Builder) Outbox() (outbox.Querier, error) {
	b.mu.Lock()
	if b.outbox != nil {
		defer b.mu.Unlock()
		return b.outbox, nil
	}
	b.mu.Unlock()

	// Repository() takes the same lock, so it must not be called while it is
	// held (the initial version deadlocked here).
	repo, err := b.Repository()
	if err != nil {
		return nil, err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.outbox == nil {
		b.outbox = database.NewOutboxAdapter(repo)
	}

	return b.outbox, nil
}
