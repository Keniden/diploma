package routes

import (
	"time"

	"github.com/google/uuid"
)

type routeRecord struct {
	ID         uuid.UUID `db:"route_id"`
	ShipmentID uuid.UUID `db:"shipment_id"`
	TemplateID int64     `db:"routecar_id"`
	Distance   int64     `db:"distance"`
	Duration   int64     `db:"duration"`
}

type routePointRecord struct {
	ID         uuid.UUID  `db:"route_point_id"`
	RouteID    uuid.UUID  `db:"route_id"`
	ShipmentID uuid.UUID  `db:"shipment_id"`
	PointID    int64      `db:"routecar_point_id"`
	SeqNum     int32      `db:"seq_num"`
	Distance   int64      `db:"distance"`
	Status     int        `db:"route_point_status"`
	DeletedAt  *time.Time `db:"deleted_at"`
}
