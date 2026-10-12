package publicdriver

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/values"
)

// UpdateRoutePointDriverDepartedRequest carries the shipment and the route
// point from which the driver has departed.
type UpdateRoutePointDriverDepartedRequest struct {
	ShipmentID   uuid.UUID
	RoutePointID uuid.UUID
}

// UpdateRoutePointDriverDeparted completes the route point from which the
// driver has departed (COMPLETED / PACKAGES_NOT_UNLOADED / PACKAGES_NOT_LOADED)
// and, when the point is the last one of the route, completes the shipment too.
func (a *Application) UpdateRoutePointDriverDeparted(
	ctx context.Context,
	req UpdateRoutePointDriverDepartedRequest,
) error {
	if _, err := a.routePointService.DepartFromRoutePoint(
		ctx,
		values.ShipmentID(req.ShipmentID),
		values.RoutePointID(req.RoutePointID),
	); err != nil {
		return fmt.Errorf("depart from route point: %w", err)
	}

	return nil
}