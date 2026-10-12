package shipments

import (
	"time"

	"github.com/google/uuid"
)

type shipmentRecord struct {
	ID           uuid.UUID  `db:"shipment_id"`
	ShortID      int64      `db:"shipment_short_id"`
	Status       int        `db:"shipment_status"`
	DriverID     uuid.UUID  `db:"driver_id"`
	SupplierID   uint64     `db:"supplier_id"`
	LicensePlate *string    `db:"car_license_plate"`
	DeletedAt    *time.Time `db:"deleted_at"`
}
