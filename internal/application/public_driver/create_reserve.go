package publicdriver

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/pkg/qr"
)

// CreateReserveRequest carries the shipment and route point on which the door
// gate reservation is created.
type CreateReserveRequest struct {
	ShipmentID   uuid.UUID
	RoutePointID uuid.UUID
}

// CreateReserveResult carries the created reserve id and the signed QR
// handshake session handed to the driver.
type CreateReserveResult struct {
	ReserveID uuid.UUID
	Session   string
}

// CreateReserve creates a gate reserve for a route point (ARCHITECTURE.md §23
// flow 4/7), generates the QR handshake session and returns the reserve id.
func (a *Application) CreateReserve(
	ctx context.Context,
	req CreateReserveRequest,
) (CreateReserveResult, error) {
	reserve, err := a.reserveService.QueueReserveByShipmentID(
		ctx,
		values.ShipmentID(req.ShipmentID),
		values.RoutePointID(req.RoutePointID),
	)
	if err != nil {
		return CreateReserveResult{}, fmt.Errorf("queue reserve by shipment id: %w", err)
	}

	var session string
	if a.handshake != nil {
		expiresAt := values.Now().Add(20 * time.Minute)
		if reserve.ExpiresAt != nil {
			expiresAt = *reserve.ExpiresAt
		}

		session, err = a.handshake.CreateSession(qr.Session{
			ShipmentID:   req.ShipmentID.String(),
			RoutePointID: req.RoutePointID.String(),
			TaskID:       string(reserve.Task.ID),
			ExpiresAt:    expiresAt,
		})
		if err != nil {
			return CreateReserveResult{}, fmt.Errorf("create handshake session: %w", err)
		}
	}

	return CreateReserveResult{
		ReserveID: uuid.UUID(reserve.ID),
		Session:   session,
	}, nil
}