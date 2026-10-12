package reserves

import (
	"time"

	"github.com/google/uuid"
)

type reserveRecord struct {
	ID         uuid.UUID `db:"reserve_id"`
	RoutePoint uuid.UUID `db:"route_point_id"`
	ShipmentID uuid.UUID `db:"shipment_id"`

	Status int16 `db:"status"`

	TaskID    string     `db:"task_id"`
	ScannedAt *time.Time `db:"scanned_at"`
	GateID    int64      `db:"gate_id"`
	GateName  string     `db:"gate_name"`
	ExpiresAt *time.Time `db:"expires_at"`
}
