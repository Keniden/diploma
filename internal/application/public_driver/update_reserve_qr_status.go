package publicdriver

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/values"
)

// UpdateReserveQRStatusRequest carries the shipment and route point whose
// reserve QR status is being confirmed by the TSD/wb driver.
type UpdateReserveQRStatusRequest struct {
	ShipmentID   uuid.UUID
	RoutePointID uuid.UUID
}

// UpdateReserveQRStatus confirms the reserve QR handshake: it approves the
// gate task, marks the reserve scanned and moves the shipment into
// LOADING/UNLOADING.
func (a *Application) UpdateReserveQRStatus(
	ctx context.Context,
	req UpdateReserveQRStatusRequest,
) error {
	if err := a.reserveService.ApproveReserveByRoutePointID(
		ctx,
		values.ShipmentID(req.ShipmentID),
		values.RoutePointID(req.RoutePointID),
	); err != nil {
		return fmt.Errorf("approve reserve by route point id: %w", err)
	}

	return nil
}