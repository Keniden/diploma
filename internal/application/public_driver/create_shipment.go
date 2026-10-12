package publicdriver

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/values"
)

// CreateShipmentRequest carries the route template the shipment is assembled
// from.
type CreateShipmentRequest struct {
	RouteTemplateID int64
}

// CreateShipment assembles a new shipment from the route template, assigns
// the current driver and plans the package demand — the creation flow of the
// ARCHITECTURE.md §23 flow 3.
func (a *Application) CreateShipment(
	ctx context.Context,
	req CreateShipmentRequest,
) (uuid.UUID, error) {
	driver, err := a.driversProvider.GetDriverFromContext(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get driver from context: %w", err)
	}

	if err = a.assemblerService.DriverCanCreateShipment(ctx, driver); err != nil {
		return uuid.Nil, fmt.Errorf("driver can create shipment: %w", err)
	}

	template, err := a.routeTemplatesProvider.GetRouteTemplateByID(
		ctx,
		values.RouteTemplateID(req.RouteTemplateID),
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get route template by id: %w", err)
	}

	var shipmentID values.ShipmentID

	err = a.tx.WithTx(ctx, func(txCtx context.Context) error {
		id, createErr := a.assemblerService.CreateShipmentFromRouteTemplate(txCtx, template)
		if createErr != nil {
			return fmt.Errorf("create shipment from route template: %w", createErr)
		}
		shipmentID = id

		if assignErr := a.progressService.AssignDriverByShipmentID(txCtx, shipmentID, driver); assignErr != nil {
			return a.cancelCreatedShipment(txCtx, shipmentID, fmt.Errorf("assign driver by shipment id: %w", assignErr))
		}

		if planErr := a.demandService.PlanPackagesToShipmentByID(txCtx, shipmentID); planErr != nil {
			return a.cancelCreatedShipment(txCtx, shipmentID, fmt.Errorf("plan packages to shipment by id: %w", planErr))
		}

		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	return shipmentID, nil
}