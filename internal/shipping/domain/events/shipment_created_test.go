package events_test

import (
	"testing"

	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/values"
)

func TestNewShipmentCreatedEvent(t *testing.T) {
	shipment := entities.NewShipment()
	route := &entities.Route{
		ID: values.NewID(),
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 10, SeqNum: entities.SourceSeqNum},
			{ID: values.NewID(), PointID: 20, SeqNum: entities.SourceSeqNum + 1},
			{ID: values.NewID(), PointID: 30, SeqNum: entities.SourceSeqNum + 2},
		},
	}

	evt, err := events.NewShipmentCreatedEvent(shipment, route)
	if err != nil {
		t.Fatal(err)
	}

	if evt.ShipmentID != shipment.ID {
		t.Fatalf("expected shipment ID %v, got %v", shipment.ID, evt.ShipmentID)
	}
	if evt.RouteID != route.ID {
		t.Fatalf("expected route ID %v, got %v", route.ID, evt.RouteID)
	}
	if evt.SourceRoutePointID != route.Points[0].ID {
		t.Fatal("source route point ID mismatch")
	}
	if evt.SourcePointID != 10 {
		t.Fatalf("expected source point ID 10, got %d", evt.SourcePointID)
	}
}

func TestNewShipmentCreatedEvent_RouteTooShort(t *testing.T) {
	shipment := entities.NewShipment()
	route := &entities.Route{
		ID: values.NewID(),
		Points: []*entities.RoutePoint{
			{ID: values.NewID(), PointID: 10},
		},
	}

	_, err := events.NewShipmentCreatedEvent(shipment, route)
	if err == nil {
		t.Fatal("expected error for too-short route")
	}
}

func TestShipmentCreatedEvent_Type(t *testing.T) {
	evt := events.ShipmentCreatedEvent{
		ShipmentID: values.NewID(),
		RouteID:    values.NewID(),
	}

	if evt.Type() != events.ShipmentCreated {
		t.Fatalf("expected type %q, got %q", events.ShipmentCreated, evt.Type())
	}
}
