package postgrestest

import (
	"context"
	"testing"

	"last-mile-task/internal/platform/database"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// Repo opens a fresh pool to the test database and returns a
// database.Repository whose schema is migrated. Close the returned pool from
// the caller (see Close).
func Repo(t *testing.T) *database.Repository {
	t.Helper()

	if err := Migrate(context.Background()); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	pool, err := OpenPool(context.Background())
	if err != nil {
		t.Fatalf("open test pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return database.NewRepository(pool)
}

// Pool is Repo without the repository wrapper.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	if err := Migrate(context.Background()); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	pool, err := OpenPool(context.Background())
	if err != nil {
		t.Fatalf("open test pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}

// CleanTables deletes all rows of the shipping test tables. It must only be
// used on a dedicated test database: it wipes every lmt table.
func CleanTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	tables := []string{
		"lmt.shipment",
		"lmt.route",
		"lmt.route_point",
		"lmt.package",
		"lmt.package_route_point",
		"lmt.reserve",
		"lmt.basket",
		"lmt.documents",
		"lmt.outbox",
	}
	for _, table := range tables {
		_, err := pool.Exec(context.Background(), "DELETE FROM "+table)
		require.NoError(t, err, "clean table %s", table)
	}
}
