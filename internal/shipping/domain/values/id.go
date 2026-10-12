package values

import (
	"strconv"

	"github.com/google/uuid"

	"last-mile-task/internal/primitives"
)

type ID = uuid.UUID

var NilID ID = uuid.Nil

func NewID() ID {
	return primitives.NewID()
}

func NewShipmentShortID() ShipmentShortID {
	return ShipmentShortID(primitives.NextSnowflake())
}

type (
	ShipmentID      = ID
	ShipmentShortID int64
	RouteID         = ID
	RouteTemplateID int64
	RoutePointID    = ID
	PointID         int64
	PackageID       = ID
	ReserveID       = ID
	GateID          int64
	ReserveTaskID   string

	TareID     int64
	TareType   string
	DriverID   = ID
	SupplierID uint64
)

func (id TareID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

func (id TareID) Sticker(tareType TareType) string {
	return string(tareType) + strconv.FormatInt(int64(id), 10)
}
