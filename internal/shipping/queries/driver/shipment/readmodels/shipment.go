package readmodels

import (
	"time"

	"github.com/google/uuid"
)

type Shipment struct {
	ID           uuid.UUID
	ShortID      int64
	Status       int
	DriverID     uuid.UUID
	SupplierID   uint64
	LicensePlate string
	ClosedAt     *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}
