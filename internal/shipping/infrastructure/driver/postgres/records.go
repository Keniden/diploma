package pgdriver

import (
	"time"

	"github.com/google/uuid"
)

type shipmentRecord struct {
	ShipmentID      uuid.UUID  `db:"shipment_id"`
	ShipmentShortID int64      `db:"shipment_short_id"`
	Status          int        `db:"shipment_status"`
	DriverID        uuid.UUID  `db:"driver_id"`
	SupplierID      uint64     `db:"supplier_id"`
	CarLicensePlate *string    `db:"car_license_plate"`
	ClosedAt        *time.Time `db:"closed_at"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
	DeletedAt       *time.Time `db:"deleted_at"`
}

type routeRecord struct {
	RouteID    uuid.UUID `db:"route_id"`
	ShipmentID uuid.UUID `db:"shipment_id"`
	TemplateID int64     `db:"routecar_id"`
	Distance   int64     `db:"distance"`
	Duration   int64     `db:"duration"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

type routePointRecord struct {
	RoutePointID uuid.UUID `db:"route_point_id"`
	RouteID      uuid.UUID `db:"route_id"`
	ShipmentID   uuid.UUID `db:"shipment_id"`
	PointID      int64     `db:"routecar_point_id"`
	SeqNum       int32     `db:"seq_num"`
	Status       int       `db:"route_point_status"`
	Distance     int64     `db:"distance"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type dstPointRecord struct {
	PackageID    uuid.UUID `db:"package_id"`
	RoutePointID uuid.UUID `db:"route_point_id"`
}

type packageActionRecord struct {
	PackageID       uuid.UUID  `db:"package_id"`
	RoutePointID    uuid.UUID  `db:"route_point_id"`
	ShipmentID      uuid.UUID  `db:"shipment_id"`
	Action          int        `db:"action"`
	Status          int        `db:"status"`
	ActionTimestamp *time.Time `db:"action_timestamp"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
}

type packageRecord struct {
	PackageID  uuid.UUID  `db:"package_id"`
	ShipmentID uuid.UUID  `db:"shipment_id"`
	IsReturn   bool       `db:"is_return"`
	TareID     int64      `db:"tare_id"`
	TareType   string     `db:"tare_type"`
	Status     int        `db:"package_status"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"`
	DeletedAt  *time.Time `db:"deleted_at"`
}
