package documentspg

import (
	"time"

	"github.com/google/uuid"
)

// documentRecord mirrors the lmt.documents row.
type documentRecord struct {
	ID               uuid.UUID  `db:"id"`
	ShipmentID       uuid.UUID  `db:"shipment_id"`
	EventUUID        string     `db:"event_uuid"`
	DocumentID       int64      `db:"document_id"`
	DataPrintTaskID  string     `db:"data_print_task_uuid"`
	GenerateTaskID   string     `db:"generate_task_uuid"`
	FromRoutePointID *uuid.UUID `db:"from_route_point_id"`
	ToRoutePointID   *uuid.UUID `db:"to_route_point_id"`
	DocumentType     int16      `db:"document_type"`
	CreatedAt        time.Time  `db:"created_at"`
}

// documentPackageRecord mirrors the lmt.document_package row.
type documentPackageRecord struct {
	DocumentID uuid.UUID `db:"document_id"`
	PackageID  uuid.UUID `db:"package_id"`
}