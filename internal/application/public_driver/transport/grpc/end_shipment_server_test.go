package drivergrpc

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	publicdriver "last-mile-task/internal/application/public_driver"
	shipping_application "last-mile-task/internal/shipping/application"
	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/queries/driver/shipment/readmodels"
)

// ==== end-shipment / departure port mocks (kept local: the application_test
// mocks are not importable from this package) ====

type endShipmentRepo struct {
	shipment *entities.Shipment
}

func (r *endShipmentRepo) GetShipmentByID(_ context.Context, _ values.ShipmentID) (*entities.Shipment, error) {
	if r.shipment == nil {
		return nil, ports.ErrShipmentNotFound
	}
	return r.shipment, nil
}

func (r *endShipmentRepo) SaveShipment(_ context.Context, shipment *entities.Shipment) error {
	r.shipment = shipment
	return nil
}

type endPolicyRepo struct {
	shipment *entities.Shipment
	packages []*entities.Package
}

func (r *endPolicyRepo) GetShipmentByID(_ context.Context, _ values.ShipmentID) (*entities.Shipment, error) {
	if r.shipment == nil {
		return nil, ports.ErrShipmentNotFound
	}
	return r.shipment, nil
}

func (r *endPolicyRepo) GetRouteByShipmentID(context.Context, values.ShipmentID) (*entities.Route, error) {
	return nil, nil
}

func (r *endPolicyRepo) GetShipmentCountByDriverID(context.Context, values.DriverID, ...values.ShipmentStatus) (int, error) {
	return 0, nil
}

func (r *endPolicyRepo) GetPackageCountByShipmentID(context.Context, values.ShipmentID, ...values.PackageStatus) (int, error) {
	return 0, nil
}

func (r *endPolicyRepo) GetPackagesByShipmentID(context.Context, values.ShipmentID) ([]*entities.Package, error) {
	return r.packages, nil
}

func (r *endPolicyRepo) GetReserveCountByShipmentID(context.Context, values.ShipmentID, ...values.ReserveStatus) (int, error) {
	return 0, nil
}

func (r *endPolicyRepo) HasSourcePackages(context.Context, *entities.Route) (bool, error) {
	return true, nil
}

type endEventPublisher struct {
	events []events.ShipmentEvent
}

func (p *endEventPublisher) PublishShipmentEvent(_ context.Context, event events.ShipmentEvent) error {
	p.events = append(p.events, event)
	return nil
}

type endRouteRepo struct {
	route *entities.Route
}

func (r *endRouteRepo) GetRouteByID(_ context.Context, _ values.RouteID) (*entities.Route, error) {
	if r.route == nil {
		return nil, ports.ErrRouteNotFound
	}
	return r.route, nil
}

func (r *endRouteRepo) GetRouteByShipmentID(_ context.Context, _ values.ShipmentID) (*entities.Route, error) {
	if r.route == nil {
		return nil, ports.ErrRouteNotFound
	}
	return r.route, nil
}

func (r *endRouteRepo) SaveRoute(_ context.Context, _ *entities.Route) error {
	return nil
}

type endRouteEvents struct {
	routeEvents []events.RouteEvent
}

func (e *endRouteEvents) PublishRouteEvent(_ context.Context, event events.RouteEvent) error {
	e.routeEvents = append(e.routeEvents, event)
	return nil
}

// ==== harness ====

type endHarness struct {
	srv      *Server
	store    *fakeShipmentStore
	shipment *entities.Shipment
	route    *entities.Route
	events   *endEventPublisher
	routeEvs *endRouteEvents
}

// newEndHarness wires a real progress/route-point service over in-memory
// repositories and seeds the shipment read-model store so the handlers can
// resolve the driver's active shipment. activeShipmentID drives the read
// model; shipment == nil makes the repositories report ErrShipmentNotFound.
func newEndHarness(
	t *testing.T,
	activeShipmentID uuid.UUID,
	shipment *entities.Shipment,
	route *entities.Route,
	packages []*entities.Package,
) *endHarness {
	t.Helper()

	store := &fakeShipmentStore{}
	if activeShipmentID != uuid.Nil {
		store.driverShipments = []readmodels.Shipment{{ID: activeShipmentID}}
		store.shipment = &readmodels.Shipment{ID: activeShipmentID}
	}
	if route != nil {
		store.route = &readmodels.Route{ID: uuid.UUID(route.ID), ShipmentID: uuid.UUID(route.ShipmentID)}
		for _, p := range route.Points {
			store.routePoints = append(store.routePoints, readmodels.RoutePoint{
				ID:      uuid.UUID(p.ID),
				SeqNum:  p.SeqNum,
				PointID: int64(p.PointID),
				Status:  int(p.Status),
			})
		}
	}

	events := &endEventPublisher{}
	progress := shipping_application.NewShipmentProgressService(
		policies.NewShipmentPolicy(&endPolicyRepo{shipment: shipment, packages: packages}),
		&endShipmentRepo{shipment: shipment},
		events,
		fakeTx{},
	)

	routeEvents := &endRouteEvents{}
	var routePointService *shipping_application.RoutePointService
	if route != nil {
		routePointService = shipping_application.NewRoutePointService(
			&endRouteRepo{route: route},
			&fakePackageRepo{packages: packages},
			routeEvents,
			progress,
			fakeTx{},
		)
	}

	app := publicdriver.NewForTesting(
		&fakeDriverProvider{driverID: values.NewID()},
		newTestQueries(store),
		nil,
		nil,
		progress,
		routePointService,
		nil,
		nil,
		fakeTx{},
	)

	srv := New(slog.New(slog.NewTextHandler(io.Discard, nil)), app)

	return &endHarness{
		srv:      srv,
		store:    store,
		shipment: shipment,
		route:    route,
		events:   events,
		routeEvs: routeEvents,
	}
}

func newEndInProgress(t *testing.T) *entities.Shipment {
	t.Helper()

	shipment := entities.NewShipment()
	if err := shipment.AssignDriver(values.NewID(), 42); err != nil {
		t.Fatal(err)
	}
	if err := shipment.Loading(); err != nil {
		t.Fatal(err)
	}
	if err := shipment.Start(); err != nil {
		t.Fatal(err)
	}
	return shipment
}

// endRoute returns a two-point route with the source COMPLETED and the target
// (last point) DRIVER_ARRIVED, so the driver-departed handler resolves the
// target as the arrived point and the target departure completes the shipment.
func endRoute(shippedID values.ShipmentID) (*entities.Route, *entities.RoutePoint) {
	source := &entities.RoutePoint{
		ID:      values.NewID(),
		PointID: values.PointID(1001),
		SeqNum:  0,
		Status:  values.RoutePointStatusCompleted,
	}
	target := &entities.RoutePoint{
		ID:      values.NewID(),
		PointID: values.PointID(1002),
		SeqNum:  1,
		Status:  values.RoutePointStatusDriverArrived,
	}
	route := &entities.Route{
		ID:         values.NewID(),
		ShipmentID: shippedID,
		Points:     []*entities.RoutePoint{source, target},
	}
	route.RebuildSequence()
	return route, target
}

// noArrivedRoute returns a route where the source is COMPLETED and the target
// is WAITING_ARRIVAL, so no route point is DRIVER_ARRIVED (the handler must
// fail with FailedPrecondition before reaching the departure service).
func noArrivedRoute(shippedID values.ShipmentID) *entities.Route {
	source := &entities.RoutePoint{
		ID:      values.NewID(),
		PointID: values.PointID(1001),
		SeqNum:  0,
		Status:  values.RoutePointStatusCompleted,
	}
	target := &entities.RoutePoint{
		ID:      values.NewID(),
		PointID: values.PointID(1002),
		SeqNum:  1,
		Status:  values.RoutePointStatusWaitingArrival,
	}
	route := &entities.Route{
		ID:         values.NewID(),
		ShipmentID: shippedID,
		Points:     []*entities.RoutePoint{source, target},
	}
	route.RebuildSequence()
	return route
}

func pendingUnloadAt(rp values.RoutePointID) *entities.Package {
	return &entities.Package{
		ID:     values.NewID(),
		Status: values.PackageStatusLoaded,
		Actions: []*entities.PackageAction{
			{
				RoutePointID: rp,
				Type:         values.RoutePointActionTypeUnload,
				Status:       values.RoutePointPackageStatusTODO,
			},
		},
	}
}

// ==== EndShipment handler tests ====

func TestEndShipment_Success(t *testing.T) {
	ctx := context.Background()
	shipment := newEndInProgress(t)
	h := newEndHarness(t, uuid.UUID(shipment.ID), shipment, nil, nil)

	if _, err := h.srv.EndShipment(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("EndShipment: %v", err)
	}
	if shipment.Status != values.ShipmentStatusCompleted {
		t.Fatalf("shipment must be COMPLETED, got %v", shipment.Status)
	}
	if len(h.events.events) != 1 {
		t.Fatalf("expected one published ShipmentCompleted event, got %d", len(h.events.events))
	}
}

func TestEndShipment_NotReadyToComplete(t *testing.T) {
	ctx := context.Background()
	shipment := entities.NewShipment() // NEW status: not completable.
	h := newEndHarness(t, uuid.UUID(shipment.ID), shipment, nil, nil)

	_, err := h.srv.EndShipment(ctx, &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", status.Code(err))
	}
	if shipment.Status != values.ShipmentStatusNew {
		t.Fatalf("shipment status must not change, got %v", shipment.Status)
	}
	if len(h.events.events) != 0 {
		t.Fatalf("no event must be published on failure, got %d", len(h.events.events))
	}
}

func TestEndShipment_NotFound(t *testing.T) {
	ctx := context.Background()
	activeID := uuid.New()
	h := newEndHarness(t, activeID, nil, nil, nil) // shipment == nil → NotFound.

	_, err := h.srv.EndShipment(ctx, &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", status.Code(err))
	}
}

func TestEndShipment_NoActiveShipment(t *testing.T) {
	ctx := context.Background()
	h := newEndHarness(t, uuid.Nil, nil, nil, nil)

	_, err := h.srv.EndShipment(ctx, &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", status.Code(err))
	}
}

// ==== UpdateRoutePointDriverDeparted handler tests ====

func TestUpdateRoutePointDriverDeparted_Success(t *testing.T) {
	ctx := context.Background()
	shipment := newEndInProgress(t)
	route, target := endRoute(shipment.ID)
	h := newEndHarness(t, uuid.UUID(shipment.ID), shipment, route, nil)

	if _, err := h.srv.UpdateRoutePointDriverDeparted(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("UpdateRoutePointDriverDeparted: %v", err)
	}
	if target.Status != values.RoutePointStatusCompleted {
		t.Fatalf("target must be COMPLETED, got %v", target.Status)
	}
	if shipment.Status != values.ShipmentStatusCompleted {
		t.Fatalf("shipment must be COMPLETED, got %v", shipment.Status)
	}
	if !hasPointCompletedEvent(h.routeEvs, target.ID) {
		t.Fatal("expected a PointCompleted event for the target")
	}
}

func TestUpdateRoutePointDriverDeparted_PendingUnload(t *testing.T) {
	ctx := context.Background()
	shipment := newEndInProgress(t)
	route, target := endRoute(shipment.ID)
	h := newEndHarness(t, uuid.UUID(shipment.ID), shipment, route, []*entities.Package{pendingUnloadAt(target.ID)})

	_, err := h.srv.UpdateRoutePointDriverDeparted(ctx, &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected pending unload error")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
	if target.Status != values.RoutePointStatusPackagesNotUnloaded {
		t.Fatalf("target must be PACKAGES_NOT_UNLOADED, got %v", target.Status)
	}
	if shipment.Status != values.ShipmentStatusInProgress {
		t.Fatalf("shipment must not be completed, got %v", shipment.Status)
	}
	if len(h.events.events) != 0 {
		t.Fatalf("no ShipmentCompleted event must be published, got %d", len(h.events.events))
	}
}

func TestUpdateRoutePointDriverDeparted_NoActiveShipment(t *testing.T) {
	ctx := context.Background()
	h := newEndHarness(t, uuid.Nil, nil, nil, nil)

	_, err := h.srv.UpdateRoutePointDriverDeparted(ctx, &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", status.Code(err))
	}
}

func TestUpdateRoutePointDriverDeparted_NoArrivedRoutePoint(t *testing.T) {
	ctx := context.Background()
	shipment := newEndInProgress(t)
	route := noArrivedRoute(shipment.ID)
	h := newEndHarness(t, uuid.UUID(shipment.ID), shipment, route, nil)

	_, err := h.srv.UpdateRoutePointDriverDeparted(ctx, &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", status.Code(err))
	}
}

func hasPointCompletedEvent(publisher *endRouteEvents, routePointID values.RoutePointID) bool {
	for _, e := range publisher.routeEvents {
		if p, ok := e.(events.PointCompletedEvent); ok && p.RoutePointID == routePointID {
			return true
		}
	}
	return false
}
