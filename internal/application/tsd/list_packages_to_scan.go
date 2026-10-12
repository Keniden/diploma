package tsd

import (
	"context"
	"fmt"

	"last-mile-task/internal/shipping/domain/values"
)

// ListPackagesToScan returns the packages awaiting a scan at a route point
// (READY_TO_LOAD / IN_TRANSIT). An empty list is returned when the point has
// no packages to scan.
func (a *Application) ListPackagesToScan(
	ctx context.Context,
	shipmentID values.ShipmentID,
	routePointID values.RoutePointID,
) ([]PackageToScan, error) {
	packages, err := a.packagesToScan.ListPackagesToScan(ctx, shipmentID, routePointID)
	if err != nil {
		return nil, fmt.Errorf("list packages to scan: %w", err)
	}

	return packages, nil
}
