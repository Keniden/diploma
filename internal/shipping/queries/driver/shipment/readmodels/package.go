package readmodels

import (
	"time"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/values"
)

type Package struct {
	ID         uuid.UUID
	ShipmentID uuid.UUID
	TareID     values.TareID
	TareType   values.TareType
	IsReturn   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

type PackageAction struct {
	PackageID       uuid.UUID
	RoutePointID    uuid.UUID
	ShipmentID      uuid.UUID
	Action          int
	Status          int
	ActionTimestamp *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
