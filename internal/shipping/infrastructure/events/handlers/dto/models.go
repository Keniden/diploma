package dto

import (
	"time"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/values"
)

func Username(firstName, middleName, lastName string) string {
	name := firstName
	if middleName != "" {
		name += " " + middleName
	}

	if lastName != "" {
		name += " " + lastName
	}

	return name
}

type Shipment struct {
	ID              values.ShipmentID
	DriverID        values.DriverID
	CarLicensePlate values.LicensePlate
}

type RoutePoint struct {
	ID        values.RoutePointID
	PointID   values.PointID
	PointUUID uuid.UUID
}

type Package struct {
	ID          values.PackageID
	TareID      values.TareID
	TareType    values.TareType
	TareSticker string
	ActionType  values.PackageActionType
	ScannedAt   time.Time
}
