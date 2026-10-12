// Package postgrestest provides a PostgreSQL-backed harness for the shipping
// persistence integration tests.
//
// It runs against a dedicated test database (see TestDSN) and applies the real
// goose migrations from migrations/ before the tests execute, so the adapters
// are exercised against exactly the schema the production code targets. No
// production database is ever touched: never point the DSN at last_mile,
// last_mile_pgdriver or last_mile_e2e.
package postgrestest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const defaultDSN = "postgres://postgres:postgres@localhost:5432/last_mile_agent_e1?sslmode=disable"

// advisory lock key serializing concurrent migration runs (test binaries run
// packages in parallel and share the test database).
const migrateLockKey = 72710

// TestDSN returns the DSN of the dedicated test database. Override with
// LMT_TEST_DB_URL when the default postgres container setup is not used.
func TestDSN() string {
	if dsn := os.Getenv("LMT_TEST_DB_URL"); dsn != "" {
		return dsn
	}
	return defaultDSN
}

// OpenPool connects to the test database.
func OpenPool(ctx context.Context) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, TestDSN())
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping test database %s: %w", TestDSN(), err)
	}
	return pool, nil
}

// Migrate applies all goose migrations to the test database. It is
// idempotent and serialized across concurrent runs with a session-level
// advisory lock taken on a dedicated connection.
func Migrate(ctx context.Context) error {
	dsn := TestDSN()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open migration db: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping migration db %s: %w", dsn, err)
	}

	lockConn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration lock conn: %w", err)
	}
	defer func() { _ = lockConn.Close() }()

	if _, err := lockConn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", migrateLockKey); err != nil {
		return fmt.Errorf("acquire migration advisory lock: %w", err)
	}
	defer func() { _, _ = lockConn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", migrateLockKey) }()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS(migrationsDir()))
	if err != nil {
		return fmt.Errorf("goose provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}

// migrationsDir resolves the module-root migrations directory by walking up
// from the current working directory (go test runs from the package dir).
func migrationsDir() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("postgrestest: getwd: %v", err))
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "migrations")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic(fmt.Sprintf("postgrestest: go.mod not found above %s", dir))
		}
		dir = parent
	}
}

var _ = stdlib.GetDefaultDriver // ensure pgx stdlib driver registered
