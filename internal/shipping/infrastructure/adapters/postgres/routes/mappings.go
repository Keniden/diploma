package routes

import (
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
)

func routeFromRecord(rec routeRecord) *entities.Route {
	return &entities.Route{
		ID:         rec.ID,
		ShipmentID: rec.ShipmentID,
		TemplateID: values.RouteTemplateID(rec.TemplateID),
		Distance:   rec.Distance,
		Duration:   rec.Duration,
		Points:     make([]*entities.RoutePoint, 0),
	}
}

func routePointFromRecord(rec routePointRecord) *entities.RoutePoint {
	return &entities.RoutePoint{
		ID:       rec.ID,
		PointID:  values.PointID(rec.PointID),
		SeqNum:   rec.SeqNum,
		Distance: rec.Distance,
		Status:   values.RoutePointStatus(rec.Status),
	}
}
