package publicdriver

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/values"
)

// CancelReserve cancels a not-yet-confirmed reserve of a shipment and returns
// the shipment to its previous stage.
func (a *Application) CancelReserve(
	ctx context.Context,
	shipmentID uuid.UUID,
) error {
	if err := a.reserveService.CancelReserveByShipmentID(
		ctx,
		values.ShipmentID(shipmentID),
	); err != nil {
		return fmt.Errorf("cancel reserve by shipment id: %w", err)
	}

	return nil
}