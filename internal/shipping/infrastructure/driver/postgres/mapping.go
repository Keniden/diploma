package pgdriver

import (
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/queries/driver/shipment/readmodels"
	"last-mile-task/internal/utils"
)

func mapRecordToShipment(shipmentRec shipmentRecord) readmodels.Shipment {
	return readmodels.Shipment{
		ID:           shipmentRec.ShipmentID,
		ShortID:      shipmentRec.ShipmentShortID,
		Status:       shipmentRec.Status,
		DriverID:     shipmentRec.DriverID,
		SupplierID:   shipmentRec.SupplierID,
		LicensePlate: utils.Unref(shipmentRec.CarLicensePlate),
		ClosedAt:     shipmentRec.ClosedAt,
		CreatedAt:    shipmentRec.CreatedAt,
		UpdatedAt:    shipmentRec.UpdatedAt,
		DeletedAt:    shipmentRec.DeletedAt,
	}
}

func mapRecordToRoute(record routeRecord) readmodels.Route {
	return readmodels.Route{
		ID:         record.RouteID,
		ShipmentID: record.ShipmentID,
		TemplateID: record.TemplateID,
		Distance:   record.Distance,
		Duration:   record.Duration,
	}
}

func mapRecordToRoutePoint(record routePointRecord) readmodels.RoutePoint {
	return readmodels.RoutePoint{
		ID:         record.RoutePointID,
		ShipmentID: record.ShipmentID,
		RouteID:    record.RouteID,
		PointID:    record.PointID,
		SeqNum:     record.SeqNum,
		Status:     record.Status,
		Distance:   record.Distance,
	}
}

func mapRecordToPackageAction(record packageActionRecord) readmodels.PackageAction {
	return readmodels.PackageAction{
		PackageID:       record.PackageID,
		RoutePointID:    record.RoutePointID,
		ShipmentID:      record.ShipmentID,
		Action:          record.Action,
		Status:          record.Status,
		ActionTimestamp: record.ActionTimestamp,
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
	}
}

func mapRecordToPackage(record packageRecord) readmodels.Package {
	return readmodels.Package{
		ID:         record.PackageID,
		ShipmentID: record.ShipmentID,
		IsReturn:   record.IsReturn,
		TareID:     values.TareID(record.TareID),
		TareType:   values.TareType(record.TareType),
		CreatedAt:  record.CreatedAt,
		UpdatedAt:  record.UpdatedAt,
		DeletedAt:  record.DeletedAt,
	}
}
