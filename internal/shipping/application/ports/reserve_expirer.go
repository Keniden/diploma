package ports

import "context"

// ReserveExpirer expires due reserves on a schedule. The concrete
// implementation (shipping/application.ReserveService) is idempotent: already
// expired reserves are not reprocessed.
type ReserveExpirer interface {
	ExpireDueWaitingApprovalReserves(ctx context.Context) error
	ExpireDueActiveReserves(ctx context.Context) error
}