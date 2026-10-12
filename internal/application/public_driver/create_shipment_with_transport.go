package publicdriver

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/values"
)

// CreateShipmentWithTransportRequest carries the route template and the car
// license plate for the shipment.
type CreateShipmentWithTransportRequest struct {
	RouteTemplateID int64
	LicensePlate    string
}

// CreateShipmentWithTransport runs the whole creation flow — assembly, driver
// assignment, demand planning and transport assignment — inside a single
// transaction. Any failure cancels the created shipment.
func (a *Application) CreateShipmentWithTransport(
	ctx context.Context,
	req CreateShipmentWithTransportRequest,
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

		plate, plateErr := parseLicensePlate(req.LicensePlate)
		if plateErr != nil {
			return a.cancelCreatedShipment(txCtx, shipmentID, plateErr)
		}

		vehicle := values.Vehicle{
			ID:           0, // TODO(LASTMILE-113): integrate with vehicles-core.
			LicensePlate: plate,
		}
		if transportErr := a.progressService.AssignTransportByShipmentID(txCtx, shipmentID, vehicle); transportErr != nil {
			return a.cancelCreatedShipment(txCtx, shipmentID, fmt.Errorf("assign transport by shipment id: %w", transportErr))
		}

		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	return shipmentID, nil
}

// cancelCreatedShipment cancels the freshly created shipment after a failed
// step of the creation flow and returns the original cause (wrapped with the
// cancel error when the cancellation itself fails).
func (a *Application) cancelCreatedShipment(
	ctx context.Context,
	shipmentID values.ShipmentID,
	cause error,
) error {
	if cancelErr := a.progressService.CancelShipmentByID(ctx, shipmentID); cancelErr != nil {
		return fmt.Errorf("%w; cancel shipment: %w", cause, cancelErr)
	}

	return cause
}