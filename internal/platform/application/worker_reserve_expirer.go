package application

import (
	"context"
	"log/slog"
	"time"

	"last-mile-task/internal/shipping/application/ports"
)

const reserveExpirerInterval = 30 * time.Second

// reserveExpirer periodically expires due reserves (ExpireDueWaitingApproval
// and ExpireDueActive) on a fixed schedule. The ReserveExpirer implementation
// is idempotent, so a crashed run is safely retried on the next tick. Errors
// are logged and do not stop the loop.
type reserveExpirer struct {
	logger   *slog.Logger
	expirer  ports.ReserveExpirer
	interval time.Duration
}

// newReserveExpirer builds the expiry worker.
func newReserveExpirer(logger *slog.Logger, expirer ports.ReserveExpirer) *reserveExpirer {
	return &reserveExpirer{
		logger:   logger,
		expirer:  expirer,
		interval: reserveExpirerInterval,
	}
}

// Run pumps an expiry pass on each tick until the context is canceled.
func (w *reserveExpirer) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.expirer.ExpireDueWaitingApprovalReserves(ctx); err != nil {
				w.logger.Error("reserve expirer: expire waiting approval reserves", "error", err)
			}
			if err := w.expirer.ExpireDueActiveReserves(ctx); err != nil {
				w.logger.Error("reserve expirer: expire active reserves", "error", err)
			}
		}
	}
}