package publicdriver

import (
	"context"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
)

// SyncPackagesAtRoutePoint synchronizes the packages of a route point against
// the cargo provider.
func (a *Application) SyncPackagesAtRoutePoint(
	ctx context.Context,
	shipmentID uuid.UUID,
	routePointID uuid.UUID,
) ([]*entities.Package, error) {
	return a.packageSyncService.SyncPackagesAtRoutePoint(
		ctx,
		values.ShipmentID(shipmentID),
		values.RoutePointID(routePointID),
	)
}
