package ports

import (
	"context"
	"time"

	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
)

type ReserveManager interface {
	ReserveOrQueue(
		ctx context.Context,
		shipment *entities.Shipment,
		sourceRoutePoint *entities.RoutePoint,
		targetRoutePoints []*entities.RoutePoint,
	) (*entities.Reserve, error)
	CancelReserve(
		ctx context.Context,
		reserve *entities.Reserve,
		pointID values.PointID,
	) error
	ApproveReserveTask(
		ctx context.Context,
		task values.ReserveTask,
	) (time.Time, error)
}
