package publicdriver_test

import (
	"context"
	"testing"

	publicdriver "last-mile-task/internal/application/public_driver"
	shipping_application "last-mile-task/internal/shipping/application"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/values"
)

type mockRouteEventPublisher struct {
	routeEvents []events.RouteEvent
}

func (m *mockRouteEventPublisher) PublishRouteEvent(_ context.Context, event events.RouteEvent) error {
	m.routeEvents = append(m.routeEvents, event)
	return nil
}

func departureRoute() *entities.Route {
	source := &entities.RoutePoint{
		ID:      values.NewID(),
		PointID: values.PointID(1001),
		Status:  values.RoutePointStatusDriverArrived,
	}
	target := &entities.RoutePoint{
		ID:      values.NewID(),
		PointID: values.PointID(1002),
		Status:  values.RoutePointStatusDriverArrived,
	}
	route := &entities.Route{
		ID:         values.NewID(),
		ShipmentID: values.NewID(),
		Points:     []*entities.RoutePoint{source, target},
	}
	route.RebuildSequence()
	return route
}

func TestUpdateRoutePointDriverDeparted_LastPointCompletesShipment(t *testing.T) {
	route := departureRoute()
	target := route.Points[len(route.Points)-1]

	shipment := newInProgressShipment(t)
	shipmentEvents := &[]events.ShipmentEvent{}
	progress := newProgressService(shipment, shipmentEvents)

	routePointService := shipping_application.NewRoutePointService(
		&mockRouteRepository{
			getRouteByShipmentIDFn: func(_ context.Context, _ values.ShipmentID) (*entities.Route, error) {
				return route, nil
			},
			saveRouteFn: func(_ context.Context, _ *entities.Route) error {
				return nil
			},
		},
		&mockPackageRepository{},
		&mockRouteEventPublisher{},
		progress,
		inlineTx{},
	)

	app := publicdriver.New(publicdriver.Config{
		ProgressService:   progress,
		RoutePointService: routePointService,
	})

	err := app.UpdateRoutePointDriverDeparted(context.Background(), publicdriver.UpdateRoutePointDriverDepartedRequest{
		ShipmentID:   route.ShipmentID,
		RoutePointID: target.ID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if target.Status != values.RoutePointStatusCompleted {
		t.Fatalf("target must be COMPLETED, got %v", target.Status)
	}
	if shipment.Status != values.ShipmentStatusCompleted {
		t.Fatalf("shipment must be COMPLETED, got %v", shipment.Status)
	}
	if len(*shipmentEvents) != 1 {
		t.Fatalf("expected ShipmentCompleted event, got %d", len(*shipmentEvents))
	}
}

func TestUpdateRoutePointDriverDeparted_MiddlePointDoesNotCompleteShipment(t *testing.T) {
	route := departureRoute()
	// Insert a middle point between source and target.
	middle := &entities.RoutePoint{
		ID:      values.NewID(),
		PointID: values.PointID(1003),
		Status:  values.RoutePointStatusDriverArrived,
	}
	route.Points = []*entities.RoutePoint{route.Points[0], middle, route.Points[1]}
	route.RebuildSequence()

	shipment := newInProgressShipment(t)
	progress := newProgressService(shipment, &[]events.ShipmentEvent{})

	routePointService := shipping_application.NewRoutePointService(
		&mockRouteRepository{
			getRouteByShipmentIDFn: func(_ context.Context, _ values.ShipmentID) (*entities.Route, error) {
				return route, nil
			},
			saveRouteFn: func(_ context.Context, _ *entities.Route) error {
				return nil
			},
		},
		&mockPackageRepository{},
		&mockRouteEventPublisher{},
		progress,
		inlineTx{},
	)

	app := publicdriver.New(publicdriver.Config{
		ProgressService:   progress,
		RoutePointService: routePointService,
	})

	err := app.UpdateRoutePointDriverDeparted(context.Background(), publicdriver.UpdateRoutePointDriverDepartedRequest{
		ShipmentID:   route.ShipmentID,
		RoutePointID: middle.ID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if middle.Status != values.RoutePointStatusCompleted {
		t.Fatalf("middle must be COMPLETED, got %v", middle.Status)
	}
	if shipment.Status != values.ShipmentStatusInProgress {
		t.Fatalf("shipment must stay IN_PROGRESS, got %v", shipment.Status)
	}
}