package shipments

import (
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
)

func shipmentFromRecord(rec shipmentRecord) *entities.Shipment {
	var licensePlate values.LicensePlate
	if rec.LicensePlate != nil {
		licensePlate = values.LicensePlate(*rec.LicensePlate)
	}

	return &entities.Shipment{
		ID:           rec.ID,
		ShortID:      values.ShipmentShortID(rec.ShortID),
		Status:       values.ShipmentStatus(rec.Status),
		DriverID:     rec.DriverID,
		SupplierID:   values.SupplierID(rec.SupplierID),
		LicensePlate: licensePlate,
	}
}
