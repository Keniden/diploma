// Package cleanup reaps finished outbox rows (COMPLETED/DISCARDED) older than
// the retention window so the lmt.outbox table does not grow unbounded.
package cleanup

import (
	"context"
	"fmt"
)

// Worker deletes stale outbox messages in batches.
//
// The real repository builds the worker from *database.Repository and
// config.OutboxCleanup and deletes rows via `DELETE ... WHERE type = ANY($1)
// AND status = ANY($2) AND scheduled_at < NOW() - $3::interval ...`,
// stopping when a batch is not full (uses pgconn.CommandTag.RowsAffected).
// The skeleton has neither the repository/batch-count accessor nor the
// config block, so the worker is a stub for now. TODO(LASTMILE-056):
// implement Run once the database layer and config.OutboxCleanup are ported.
type Worker struct {
	messageTypes  []string
	retentionDays int
	batchSize     int
}

func NewWorker(messageTypes []string, retentionDays, batchSize int) *Worker {
	return &Worker{
		messageTypes:  messageTypes,
		retentionDays: retentionDays,
		batchSize:     batchSize,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	return fmt.Errorf("outbox cleanup: TODO(LASTMILE-056): not implemented in the skeleton")
}
