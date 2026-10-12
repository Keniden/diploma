package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type txKey struct{}

func (r *Repository) BeginTx(ctx context.Context) (context.Context, error) {
	txCtx, _, err := r.beginTx(ctx)
	return txCtx, err
}

func (r *Repository) beginTx(ctx context.Context) (context.Context, bool, error) {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return ctx, false, nil
	}

	tx, err := r.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, false, err
	}

	return context.WithValue(ctx, txKey{}, tx), true, nil
}

func (r *Repository) Commit(ctx context.Context) error {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx.Commit(ctx)
	}
	return nil
}

func (r *Repository) Rollback(ctx context.Context) error {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		if err := tx.Rollback(ctx); err != nil {
			if errors.Is(err, pgx.ErrTxClosed) {
				return nil
			}

			return err
		}
	}
	return nil
}
