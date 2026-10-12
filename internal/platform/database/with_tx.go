package database

import "context"

func (r *Repository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	txCtx, ownsTx, err := r.beginTx(ctx)
	if err != nil {
		return err
	}
	if ownsTx {
		defer func() { _ = r.Rollback(txCtx) }()
	}

	if err := fn(txCtx); err != nil {
		return err
	}

	if ownsTx {
		return r.Commit(txCtx)
	}

	return nil
}
