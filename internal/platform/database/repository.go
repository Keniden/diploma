package database

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TODO(LASTMILE-060): the real repository embeds the gitlab shared
// pgxadapter.Pool and exposes pgxadapter.Transaction as its Transactional
// implementation. The skeleton has no gitlab dependencies, so transactions
// are tracked in the context (see transaction.go) and every repository
// operation goes through Querier(ctx), which routes to the active
// transaction or the pool. Replace with pgxadapter once it is vendored.

type Repository struct {
	Pool    *pgxpool.Pool
	Builder sq.StatementBuilderType
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		Pool:    pool,
		Builder: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
}

func (r *Repository) Querier(ctx context.Context) Querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}

	return r.Pool
}
