package packages

import (
	"time"

	"github.com/google/uuid"
)

type packageRecord struct {
	ID         uuid.UUID  `db:"package_id"`
	ShipmentID uuid.UUID  `db:"shipment_id"`
	Status     int        `db:"package_status"`
	IsReturn   bool       `db:"is_return"`
	TareID     int64      `db:"tare_id"`
	TareType   string     `db:"tare_type"`
	DeletedAt  *time.Time `db:"deleted_at"`
}

type packageRoutePointRecord struct {
	PackageID             uuid.UUID  `db:"package_id"`
	RoutePointID          uuid.UUID  `db:"route_point_id"`
	ShipmentID            uuid.UUID  `db:"shipment_id"`
	Action                int        `db:"action"`
	ActionStatus          int        `db:"action_status"`
	ActionStatusUpdatedAt *time.Time `db:"action_status_updated_at"`
	DeletedAt             *time.Time `db:"deleted_at"`
}
