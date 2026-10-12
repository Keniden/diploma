package ports

import "context"

type Transactional interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
