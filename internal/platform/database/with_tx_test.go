package database

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type spyTx struct {
	commits   int
	rollbacks int
}

func (t *spyTx) Begin(context.Context) (pgx.Tx, error) { return t, nil }

func (t *spyTx) Commit(context.Context) error {
	t.commits++
	return nil
}

func (t *spyTx) Rollback(context.Context) error {
	t.rollbacks++
	return nil
}

func (t *spyTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (t *spyTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (t *spyTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, nil
}

func (t *spyTx) QueryRow(context.Context, string, ...any) pgx.Row {
	return nil
}

func (t *spyTx) Conn() *pgx.Conn { return nil }

func (t *spyTx) LargeObjects() pgx.LargeObjects { return pgx.LargeObjects{} }

func (t *spyTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func (t *spyTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	return nil
}

func TestWithTx_ReusesExistingTransaction(t *testing.T) {
	t.Parallel()

	tx := &spyTx{}
	repo := &Repository{}
	ctx := context.WithValue(context.Background(), txKey{}, tx)

	err := repo.WithTx(ctx, func(innerCtx context.Context) error {
		return repo.WithTx(innerCtx, func(context.Context) error {
			return nil
		})
	})
	require.NoError(t, err)
	require.Equal(t, 0, tx.commits)
	require.Equal(t, 0, tx.rollbacks)
}

func TestWithTx_ReusesExistingTransaction_PropagatesError(t *testing.T) {
	t.Parallel()

	tx := &spyTx{}
	repo := &Repository{}
	ctx := context.WithValue(context.Background(), txKey{}, tx)
	expectedErr := errors.New("boom")

	err := repo.WithTx(ctx, func(innerCtx context.Context) error {
		return repo.WithTx(innerCtx, func(context.Context) error {
			return expectedErr
		})
	})
	require.ErrorIs(t, err, expectedErr)
	require.Equal(t, 0, tx.commits)
	require.Equal(t, 0, tx.rollbacks)
}
