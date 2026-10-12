package database

import (
	"context"

	"last-mile-task/pkg/outbox"
)

// OutboxAdapter adapts the shared repository to the pkg/outbox Querier
// contract so the outbox bus can run queries and transactions through the
// same pool/tx handling as the rest of the persistence layer.
type OutboxAdapter struct {
	repo *Repository
}

func NewOutboxAdapter(repo *Repository) *OutboxAdapter {
	return &OutboxAdapter{repo: repo}
}

func (o *OutboxAdapter) Exec(ctx context.Context, query string, args ...any) error {
	_, err := o.repo.Querier(ctx).Exec(ctx, query, args...)
	return err
}

func (o *OutboxAdapter) Query(ctx context.Context, query string, args ...any) (outbox.Rows, error) {
	rows, err := o.repo.Querier(ctx).Query(ctx, query, args...)
	return rows, err
}

func (o *OutboxAdapter) BeginTx(ctx context.Context) (context.Context, error) {
	return o.repo.BeginTx(ctx)
}

func (o *OutboxAdapter) Commit(ctx context.Context) error {
	return o.repo.Commit(ctx)
}

func (o *OutboxAdapter) Rollback(ctx context.Context) error {
	return o.repo.Rollback(ctx)
}
