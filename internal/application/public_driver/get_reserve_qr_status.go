package publicdriver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/values"
)

// ReserveQRStatus is the QR handshake state of a reserve.
type ReserveQRStatus struct {
	ReserveID uuid.UUID
	IsScanned bool
	Status    values.ReserveStatus
	ExpiresAt *time.Time
}

// GetReserveQRStatus returns whether the reserve QR has been scanned by the
// TSD (is_scanned). A shipment without an active reserve yields the empty
// state without an error.
func (a *Application) GetReserveQRStatus(
	ctx context.Context,
	shipmentID uuid.UUID,
) (ReserveQRStatus, error) {
	reserve, err := a.reserveService.GetReserveByShipmentID(ctx, values.ShipmentID(shipmentID))
	if err != nil {
		if errors.Is(err, ports.ErrReserveNotFound) {
			return ReserveQRStatus{}, nil
		}

		return ReserveQRStatus{}, fmt.Errorf("get reserve by shipment id: %w", err)
	}

	return ReserveQRStatus{
		ReserveID: uuid.UUID(reserve.ID),
		IsScanned: reserve.ScannedAt != nil,
		Status:    reserve.Status,
		ExpiresAt: reserve.ExpiresAt,
	}, nil
}