package publicdriver

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/values"
)

// GetActiveShipmentID returns the ID of the driver's current active shipment,
// or uuid.Nil when the driver has none.
func (a *Application) GetActiveShipmentID(ctx context.Context) (uuid.UUID, error) {
	driver, err := a.driversProvider.GetDriverFromContext(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get driver from context: %w", err)
	}

	shipments, err := a.shippingQueries.GetShipmentByDriverInStatuses(
		ctx,
		driver.ID,
		values.ActiveShipmentStatuses()...,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get driver shipments by statuses: %w", err)
	}

	if len(shipments) == 0 {
		return uuid.Nil, nil
	}

	return shipments[0].ID, nil
}

// GetActiveShipmentSourceRoutePointID returns the route point ID of the
// driver's active shipment source point, or uuid.Nil when the driver has no
// active shipment.
func (a *Application) GetActiveShipmentSourceRoutePointID(ctx context.Context) (uuid.UUID, error) {
	activeID, err := a.GetActiveShipmentID(ctx)
	if err != nil {
		return uuid.Nil, err
	}

	if activeID == uuid.Nil {
		return uuid.Nil, nil
	}

	detailed, err := a.shippingQueries.GetDetailedShipmentByID(ctx, activeID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get detailed shipment by id: %w", err)
	}

	if len(detailed.RoutePoints) == 0 {
		return uuid.Nil, errors.New("shipment has no route points")
	}

	return detailed.RoutePoints[0].ID, nil
}

// GetActiveShipmentDriverArrivedRoutePointID returns the ID of the active
// shipment route point the driver is currently at (DRIVER_ARRIVED), or
// uuid.Nil when the driver has no active shipment or no such point.
func (a *Application) GetActiveShipmentDriverArrivedRoutePointID(ctx context.Context) (uuid.UUID, error) {
	activeID, err := a.GetActiveShipmentID(ctx)
	if err != nil {
		return uuid.Nil, err
	}

	if activeID == uuid.Nil {
		return uuid.Nil, nil
	}

	detailed, err := a.shippingQueries.GetDetailedShipmentByID(ctx, activeID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get detailed shipment by id: %w", err)
	}

	for _, rp := range detailed.RoutePoints {
		if values.RoutePointStatus(rp.Status) == values.RoutePointStatusDriverArrived {
			return rp.ID, nil
		}
	}

	return uuid.Nil, nil
}
