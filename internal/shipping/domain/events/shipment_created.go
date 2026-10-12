package events

import (
	"fmt"
	"log/slog"

	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
)

type ShipmentCreatedEvent struct {
	ShipmentID         values.ShipmentID
	RouteID            values.RouteID
	SourceRoutePointID values.RoutePointID
	SourcePointID      values.PointID
}

func NewShipmentCreatedEvent(shipment *entities.Shipment, route *entities.Route) (ShipmentCreatedEvent, error) {
	source, err := route.Source()
	if err != nil {
		return ShipmentCreatedEvent{}, fmt.Errorf("route source: %w", err)
	}

	return ShipmentCreatedEvent{
		ShipmentID:         shipment.ID,
		RouteID:            route.ID,
		SourceRoutePointID: source.ID,
		SourcePointID:      source.PointID,
	}, nil
}

func (e ShipmentCreatedEvent) Type() ShipmentEventType {
	return ShipmentCreated
}

func (e ShipmentCreatedEvent) LogValue() slog.Value {
	return slog.GroupValue(slog.String("shipment_id", e.ShipmentID.String()))
}
