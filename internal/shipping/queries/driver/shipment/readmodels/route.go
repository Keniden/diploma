package readmodels

import (
	"time"

	"github.com/google/uuid"
)

type Route struct {
	ID         uuid.UUID
	ShipmentID uuid.UUID
	TemplateID int64
	Distance   int64
	Duration   int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type RoutePoint struct {
	ID         uuid.UUID
	ShipmentID uuid.UUID
	RouteID    uuid.UUID
	PointID    int64
	Address    string
	SeqNum     int32
	Distance   int64
	Status     int
}
