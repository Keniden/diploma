package infrastructure

import (
	"context"
	"log/slog"
	"time"

	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
)

// ReserveManager is the slice-scoped stand-in for the gate-reserve service
// (internal/clients/reserve → ReserveOrQueue, DriverArrived, DriverDeparted).
//
// TODO(LASTMILE-099): replace with the client-backed adapter once the
// gate-reserve client is ported. The stub allocates a deterministic gate and
// task so the reserve flow is reproducible in the slice; CancelReserve is a
// no-op and ApproveReserveTask returns a fixed approval window.
type ReserveManager struct{}

// NewReserveManager builds the stub gate-reserve manager.
func NewReserveManager() ports.ReserveManager { return &ReserveManager{} }

// ReserveOrQueue allocates a deterministic gate/task for the reserve.
func (m *ReserveManager) ReserveOrQueue(
	_ context.Context,
	shipment *entities.Shipment,
	sourceRoutePoint *entities.RoutePoint,
	_ []*entities.RoutePoint,
) (*entities.Reserve, error) {
	expiresAt := values.Now().Add(20 * time.Minute)

	return entities.NewReserveForApproval(
		shipment.ID,
		sourceRoutePoint.ID,
		values.ReserveGate{ID: 7, Name: "Slice Gate"},
		values.ReserveTask{ID: values.ReserveTaskID("slice-task-" + shipment.ID.String()[:8])},
		expiresAt,
	), nil
}

// CancelReserve is a no-op until the gate-reserve client is ported.
func (m *ReserveManager) CancelReserve(_ context.Context, reserve *entities.Reserve, pointID values.PointID) error {
	slog.Debug("stub gate-reserve: CancelReserve",
		"reserve_id", reserve.ID.String(),
		"point_id", pointID,
	)

	return nil
}

// ApproveReserveTask confirms the gate task and returns the gate approval
// deadline.
func (m *ReserveManager) ApproveReserveTask(_ context.Context, task values.ReserveTask) (time.Time, error) {
	slog.Debug("stub gate-reserve: ApproveReserveTask", "task_id", task.ID)

	return values.Now().Add(20 * time.Minute), nil
}