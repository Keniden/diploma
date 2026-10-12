package outbox

import (
	"context"
)

type Rows interface {
	Scan(dest ...any) error
	Next() bool
	Close()
	Err() error
}

type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) error
	Query(ctx context.Context, sql string, args ...any) (Rows, error)

	Txer
}

type OutboxWriter interface {
	WriteMessage(ctx context.Context, msg *Message) error
}

type Txer interface {
	BeginTx(ctx context.Context) (context.Context, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type OutboxReader interface {
	ReadMessages(ctx context.Context, messageType string, batchSize int) ([]*Message, error)
	ScheduleMessage(ctx context.Context, message *Message) error
	CompleteMessages(ctx context.Context, messages []*Message) error
	DiscardMessages(ctx context.Context, messages []*Message) error

	Txer
}
