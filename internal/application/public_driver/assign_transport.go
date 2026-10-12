package publicdriver

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	lmterrors "last-mile-task/internal/shipping/domain/errors"
	"last-mile-task/internal/shipping/domain/values"
)

// AssignTransport sets the car license plate on the shipment (the
// UpdateShipmentTransport RPC).
func (a *Application) AssignTransport(
	ctx context.Context,
	shipmentID uuid.UUID,
	licensePlate string,
) error {
	plate, err := parseLicensePlate(licensePlate)
	if err != nil {
		return err
	}

	vehicle := values.Vehicle{
		ID:           0, // TODO(LASTMILE-113): integrate with vehicles-core.
		LicensePlate: plate,
	}

	if err := a.progressService.AssignTransportByShipmentID(
		ctx,
		values.ShipmentID(shipmentID),
		vehicle,
	); err != nil {
		return fmt.Errorf("assign transport by shipment id: %w", err)
	}

	return nil
}

// parseLicensePlate normalizes the plate and rejects empty values.
func parseLicensePlate(raw string) (values.LicensePlate, error) {
	plate := values.NewLicensePlate(raw)
	if plate == "" {
		return "", lmterrors.NewInvalidShipmentCarLicensePlateError("empty license plate")
	}
	return plate, nil
}