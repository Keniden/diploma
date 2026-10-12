package publicdriver

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/values"
)

// EndShipment completes the shipment: the driver has finished the route and
// all goods are processed. The completion is atomic (Shipment → COMPLETED,
// ShipmentCompleted event, cargo-shipment FINISH and the payments accrual
// outbox message all happen inside one transaction).
func (a *Application) EndShipment(
	ctx context.Context,
	shipmentID uuid.UUID,
) error {
	if err := a.progressService.CompleteShipmentByID(
		ctx,
		values.ShipmentID(shipmentID),
	); err != nil {
		return fmt.Errorf("complete shipment by id: %w", err)
	}

	return nil
}