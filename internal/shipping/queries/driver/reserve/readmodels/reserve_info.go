package readmodels

import (
	"time"

	"github.com/google/uuid"
)

type ReserveInfo struct {
	ReserveID    uuid.UUID
	ShipmentID   uuid.UUID
	RoutePointID uuid.UUID
	TaskID       string
	Gate         Gate
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ExpiresAt    time.Time
	Status       int
}

type Gate struct {
	ID   int32
	Name string
}
