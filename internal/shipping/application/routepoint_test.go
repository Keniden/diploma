package application_test

import (
	"context"
	"testing"

	"last-mile-task/internal/shipping/application"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/values"
)

func routePoint(status values.RoutePointStatus) *entities.RoutePoint {
	return &entities.RoutePoint{
		ID:      values.NewID(),
		PointID: values.PointID(101),
		SeqNum:  1,
		Status:  status,
	}
}

func newDepartureRoute(points ...*entities.RoutePoint) *entities.Route {
	route := &entities.Route{
		ID:         values.NewID(),
		ShipmentID: values.NewID(),
		Points:     points,
	}
	route.RebuildSequence()
	return route
}

func packageWith(status values.PackageStatus, actions ...*entities.PackageAction) *entities.Package {
	return &entities.Package{
		ID:      values.NewID(),
		Status:  status,
		Actions: actions,
	}
}

func unloadPending(rp values.RoutePointID) *entities.PackageAction {
	return &entities.PackageAction{
		RoutePointID: rp,
		Type:         values.RoutePointActionTypeUnload,
		Status:       values.RoutePointPackageStatusTODO,
	}
}

func loadPending(rp values.RoutePointID) *entities.PackageAction {
	return &entities.PackageAction{
		RoutePointID: rp,
		Type:         values.RoutePointActionTypeLoad,
		Status:       values.RoutePointPackageStatusTODO,
	}
}

func newRoutePointService(
	route *entities.Route,
	packages []*entities.Package,
	routeEvents *mockRouteEventPublisher,
	completer *mockShipmentCompleter,
) *application.RoutePointService {
	return application.NewRoutePointService(
		&mockRouteRepository{
			getRouteByShipmentIDFn: func(_ context.Context, _ values.ShipmentID) (*entities.Route, error) {
				return route, nil
			},
			saveRouteFn: func(_ context.Context, _ *entities.Route) error {
				return nil
			},
		},
		&mockPackageRepository{
			getPackagesByShipmentIDFn: func(_ context.Context, _ values.ShipmentID) ([]*entities.Package, error) {
				return packages, nil
			},
		},
		routeEvents,
		completer,
		newMockTransactional(),
	)
}

func TestDepartFromRoutePoint_Target_CompletesShipment(t *testing.T) {
	source := routePoint(values.RoutePointStatusDriverArrived)
	target := routePoint(values.RoutePointStatusDriverArrived)
	route := newDepartureRoute(source, target)

	routeEvents := &mockRouteEventPublisher{}
	completer := &mockShipmentCompleter{}
	svc := newRoutePointService(route, []*entities.Package{}, routeEvents, completer)

	departedFrom, err := svc.DepartFromRoutePoint(context.Background(), route.ShipmentID, target.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if departedFrom != application.DepartedFromTarget {
		t.Fatalf("expected DepartedFromTarget, got %d", departedFrom)
	}
	if target.Status != values.RoutePointStatusCompleted {
		t.Fatalf("target must be COMPLETED, got %v", target.Status)
	}
	if len(completer.calls) != 1 || completer.calls[0] != route.ShipmentID {
		t.Fatalf("expected shipment to be completed once, got %v", completer.calls)
	}
	if !hasPointCompleted(routeEvents, target.ID) {
		t.Fatal("expected a PointCompleted event for the target")
	}
}

func TestDepartFromRoutePoint_Target_PendingUnload(t *testing.T) {
	source := routePoint(values.RoutePointStatusDriverArrived)
	target := routePoint(values.RoutePointStatusDriverArrived)
	route := newDepartureRoute(source, target)

	routeEvents := &mockRouteEventPublisher{}
	completer := &mockShipmentCompleter{}
	svc := newRoutePointService(
		route,
		[]*entities.Package{packageWith(values.PackageStatusLoaded, unloadPending(target.ID))},
		routeEvents,
		completer,
	)

	_, err := svc.DepartFromRoutePoint(context.Background(), route.ShipmentID, target.ID)
	if err == nil {
		t.Fatal("expected pending unload error")
	}
	if target.Status != values.RoutePointStatusPackagesNotUnloaded {
		t.Fatalf("target must be PACKAGES_NOT_UNLOADED, got %v", target.Status)
	}
	if len(completer.calls) != 0 {
		t.Fatalf("shipment must not be completed, got %v", completer.calls)
	}
	if !hasPointCompleted(routeEvents, target.ID) {
		t.Fatal("expected a PointCompleted event for the target")
	}
}

func TestDepartFromRoutePoint_Target_PendingLoad(t *testing.T) {
	source := routePoint(values.RoutePointStatusDriverArrived)
	target := routePoint(values.RoutePointStatusDriverArrived)
	route := newDepartureRoute(source, target)

	routeEvents := &mockRouteEventPublisher{}
	completer := &mockShipmentCompleter{}
	svc := newRoutePointService(
		route,
		[]*entities.Package{packageWith(values.PackageStatusReadyToLoad, loadPending(target.ID))},
		routeEvents,
		completer,
	)

	_, err := svc.DepartFromRoutePoint(context.Background(), route.ShipmentID, target.ID)
	if err == nil {
		t.Fatal("expected pending load error")
	}
	if target.Status != values.RoutePointStatusPackagesNotLoaded {
		t.Fatalf("target must be PACKAGES_NOT_LOADED, got %v", target.Status)
	}
	if len(completer.calls) != 0 {
		t.Fatalf("shipment must not be completed, got %v", completer.calls)
	}
	if !hasPointCompleted(routeEvents, target.ID) {
		t.Fatal("expected a PointCompleted event for the target")
	}
}

func TestDepartFromRoutePoint_Middle_NotUnloaded(t *testing.T) {
	source := routePoint(values.RoutePointStatusDriverArrived)
	middle := routePoint(values.RoutePointStatusDriverArrived)
	target := routePoint(values.RoutePointStatusDriverArrived)
	route := newDepartureRoute(source, middle, target)

	routeEvents := &mockRouteEventPublisher{}
	completer := &mockShipmentCompleter{}
	svc := newRoutePointService(
		route,
		[]*entities.Package{packageWith(values.PackageStatusLoaded, unloadPending(middle.ID))},
		routeEvents,
		completer,
	)

	departedFrom, err := svc.DepartFromRoutePoint(context.Background(), route.ShipmentID, middle.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if departedFrom != application.DepartedFromMiddlePoint {
		t.Fatalf("expected DepartedFromMiddlePoint, got %d", departedFrom)
	}
	if middle.Status != values.RoutePointStatusPackagesNotUnloaded {
		t.Fatalf("middle must be PACKAGES_NOT_UNLOADED, got %v", middle.Status)
	}
	if len(completer.calls) != 0 {
		t.Fatalf("shipment must not be completed, got %v", completer.calls)
	}
	if !hasPointCompleted(routeEvents, middle.ID) {
		t.Fatal("expected a PointCompleted event for the middle point")
	}
}

func TestDepartFromRoutePoint_Middle_Completed(t *testing.T) {
	source := routePoint(values.RoutePointStatusDriverArrived)
	middle := routePoint(values.RoutePointStatusDriverArrived)
	target := routePoint(values.RoutePointStatusDriverArrived)
	route := newDepartureRoute(source, middle, target)

	routeEvents := &mockRouteEventPublisher{}
	completer := &mockShipmentCompleter{}
	svc := newRoutePointService(route, []*entities.Package{}, routeEvents, completer)

	departedFrom, err := svc.DepartFromRoutePoint(context.Background(), route.ShipmentID, middle.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if departedFrom != application.DepartedFromMiddlePoint {
		t.Fatalf("expected DepartedFromMiddlePoint, got %d", departedFrom)
	}
	if middle.Status != values.RoutePointStatusCompleted {
		t.Fatalf("middle must be COMPLETED, got %v", middle.Status)
	}
	if len(completer.calls) != 0 {
		t.Fatalf("shipment must not be completed, got %v", completer.calls)
	}
}

func TestDepartFromRoutePoint_Source_NoLoadedPackage(t *testing.T) {
	source := routePoint(values.RoutePointStatusDriverArrived)
	target := routePoint(values.RoutePointStatusDriverArrived)
	route := newDepartureRoute(source, target)

	routeEvents := &mockRouteEventPublisher{}
	completer := &mockShipmentCompleter{}
	svc := newRoutePointService(
		route,
		[]*entities.Package{packageWith(values.PackageStatusReadyToLoad)},
		routeEvents,
		completer,
	)

	_, err := svc.DepartFromRoutePoint(context.Background(), route.ShipmentID, source.ID)
	if err == nil {
		t.Fatal("expected no loaded packages error")
	}
	if source.Status != values.RoutePointStatusDriverArrived {
		t.Fatalf("source status must not change, got %v", source.Status)
	}
}

func TestDepartFromRoutePoint_Source_Completed(t *testing.T) {
	source := routePoint(values.RoutePointStatusDriverArrived)
	target := routePoint(values.RoutePointStatusDriverArrived)
	route := newDepartureRoute(source, target)

	routeEvents := &mockRouteEventPublisher{}
	completer := &mockShipmentCompleter{}
	svc := newRoutePointService(
		route,
		[]*entities.Package{packageWith(values.PackageStatusLoaded)},
		routeEvents,
		completer,
	)

	departedFrom, err := svc.DepartFromRoutePoint(context.Background(), route.ShipmentID, source.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if departedFrom != application.DepartedFromSource {
		t.Fatalf("expected DepartedFromSource, got %d", departedFrom)
	}
	if source.Status != values.RoutePointStatusCompleted {
		t.Fatalf("source must be COMPLETED, got %v", source.Status)
	}
	if len(completer.calls) != 0 {
		t.Fatalf("shipment must not be completed on source departure, got %v", completer.calls)
	}
	if !hasPointCompleted(routeEvents, source.ID) {
		t.Fatal("expected a PointCompleted event for the source")
	}
}

func TestDepartFromRoutePoint_CompleterErrorPropagates(t *testing.T) {
	source := routePoint(values.RoutePointStatusDriverArrived)
	target := routePoint(values.RoutePointStatusDriverArrived)
	route := newDepartureRoute(source, target)

	routeEvents := &mockRouteEventPublisher{}
	completer := &mockShipmentCompleter{err: context.DeadlineExceeded}
	svc := newRoutePointService(route, []*entities.Package{}, routeEvents, completer)

	_, err := svc.DepartFromRoutePoint(context.Background(), route.ShipmentID, target.ID)
	if err == nil {
		t.Fatal("expected the completion error to propagate")
	}
}

func hasPointCompleted(publisher *mockRouteEventPublisher, routePointID values.RoutePointID) bool {
	for _, e := range publisher.routeEvents {
		if p, ok := e.(events.PointCompletedEvent); ok && p.RoutePointID == routePointID {
			return true
		}
	}
	return false
}
