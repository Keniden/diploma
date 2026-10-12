package drivergrpc

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

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
	"last-mile-task/pkg/qr"
)

// ==== reserve port mocks (kept local: the application_test mocks are not
// importable from this package) ====

type reserveRepo struct {
	getReserveByShipmentIDFn   func(ctx context.Context, id values.ShipmentID) (*entities.Reserve, error)
	getReserveByRoutePointIDFn func(ctx context.Context, id values.RoutePointID) (*entities.Reserve, error)
	saveReserveFn              func(ctx context.Context, r *entities.Reserve) error
}

func (r *reserveRepo) GetReserveByID(context.Context, values.ReserveID) (*entities.Reserve, error) {
	return nil, nil
}

func (r *reserveRepo) GetReserveByShipmentID(ctx context.Context, id values.ShipmentID) (*entities.Reserve, error) {
	return r.getReserveByShipmentIDFn(ctx, id)
}

func (r *reserveRepo) GetReserveByRoutePointID(ctx context.Context, id values.RoutePointID) (*entities.Reserve, error) {
	if r.getReserveByRoutePointIDFn != nil {
		return r.getReserveByRoutePointIDFn(ctx, id)
	}
	return nil, ports.ErrReserveNotFound
}

func (r *reserveRepo) ListExpiredWaitingApprovalReserves(context.Context, int) ([]*entities.Reserve, error) {
	return nil, nil
}

func (r *reserveRepo) ListExpiredActiveReserves(context.Context, int) ([]*entities.Reserve, error) {
	return nil, nil
}

func (r *reserveRepo) SaveReserve(ctx context.Context, res *entities.Reserve) error {
	if r.saveReserveFn != nil {
		return r.saveReserveFn(ctx, res)
	}
	return nil
}

type reserveShipmentRepo struct {
	shipment *entities.Shipment
}

func (r *reserveShipmentRepo) GetShipmentByID(_ context.Context, _ values.ShipmentID) (*entities.Shipment, error) {
	if r.shipment == nil {
		return nil, ports.ErrShipmentNotFound
	}
	return r.shipment, nil
}

func (r *reserveShipmentRepo) SaveShipment(_ context.Context, _ *entities.Shipment) error {
	return nil
}

type reserveRouteRepo struct {
	route *entities.Route
}

func (r *reserveRouteRepo) GetRouteByID(_ context.Context, _ values.RouteID) (*entities.Route, error) {
	return r.route, nil
}

func (r *reserveRouteRepo) GetRouteByShipmentID(_ context.Context, _ values.ShipmentID) (*entities.Route, error) {
	return r.route, nil
}

func (r *reserveRouteRepo) SaveRoute(_ context.Context, _ *entities.Route) error {
	return nil
}

type reserveManager struct{}

func (reserveManager) ReserveOrQueue(_ context.Context, shipment *entities.Shipment, source *entities.RoutePoint, _ []*entities.RoutePoint) (*entities.Reserve, error) {
	expires := values.Now().Add(10 * time.Minute)
	return &entities.Reserve{
		ID:           values.NewID(),
		ShipmentID:   shipment.ID,
		RoutePointID: source.ID,
		Status:       values.ReserveStatusWaitingApproval,
		Gate:         values.ReserveGate{ID: 7, Name: "Gate A"},
		Task:         values.ReserveTask{ID: "task-7"},
		ExpiresAt:    &expires,
	}, nil
}

func (reserveManager) CancelReserve(context.Context, *entities.Reserve, values.PointID) error {
	return nil
}

func (reserveManager) ApproveReserveTask(context.Context, values.ReserveTask) (time.Time, error) {
	return values.Now().Add(time.Hour), nil
}

type reservePolicyRepo struct {
	count    int
	route    *entities.Route
	shipment *entities.Shipment
}

func (r *reservePolicyRepo) GetShipmentByID(context.Context, values.ShipmentID) (*entities.Shipment, error) {
	return r.shipment, nil
}

func (r *reservePolicyRepo) GetRouteByShipmentID(_ context.Context, _ values.ShipmentID) (*entities.Route, error) {
	return r.route, nil
}

func (r *reservePolicyRepo) GetShipmentCountByDriverID(context.Context, values.DriverID, ...values.ShipmentStatus) (int, error) {
	return 0, nil
}

func (r *reservePolicyRepo) GetPackageCountByShipmentID(context.Context, values.ShipmentID, ...values.PackageStatus) (int, error) {
	return 0, nil
}

func (r *reservePolicyRepo) GetPackagesByShipmentID(context.Context, values.ShipmentID) ([]*entities.Package, error) {
	return nil, nil
}

func (r *reservePolicyRepo) GetReserveCountByShipmentID(_ context.Context, _ values.ShipmentID, _ ...values.ReserveStatus) (int, error) {
	return r.count, nil
}

func (r *reservePolicyRepo) HasSourcePackages(context.Context, *entities.Route) (bool, error) {
	return true, nil
}

type reserveEvents struct{}

func (reserveEvents) PublishReserveEvent(context.Context, events.ReserveEvent) error {
	return nil
}

type reserveHandshake struct {
	token string
}

func (h reserveHandshake) CreateSession(qr.Session) (string, error) {
	return h.token, nil
}

func (reserveHandshake) DecodeQR(string) (qr.Session, error) {
	return qr.Session{}, nil
}

// ==== harness ====

type reserveServerHarness struct {
	srv      *Server
	shipment *entities.Shipment
	route    *entities.Route
	reserves *reserveRepo
}

func reserveActiveStore(shipmentID, routeID, sourceRoutePointID uuid.UUID) *fakeShipmentStore {
	return &fakeShipmentStore{
		driverShipments: []readmodels.Shipment{{ID: shipmentID}},
		shipment:        &readmodels.Shipment{ID: shipmentID},
		route:           &readmodels.Route{ID: routeID, ShipmentID: shipmentID},
		routePoints:     []readmodels.RoutePoint{{ID: sourceRoutePointID, SeqNum: 0, PointID: 12345}},
	}
}

func newReserveServer(
	provider publicdriver.DriversProvider,
	store *fakeShipmentStore,
	shipment *entities.Shipment,
	route *entities.Route,
	count int,
) *reserveServerHarness {
	reserves := &reserveRepo{}

	policy := policies.NewShipmentPolicy(&reservePolicyRepo{count: count, route: route, shipment: shipment})
	svc := shipping_application.NewReserveService(
		reserves,
		&reserveShipmentRepo{shipment: shipment},
		&reserveRouteRepo{route: route},
		reserveManager{},
		policy,
		reserveEvents{},
		fakeTx{},
	)

	app := publicdriver.New(publicdriver.Config{
		DriversProvider: provider,
		ShippingQueries: newTestQueries(store),
		ReserveService:  svc,
		Handshake:       reserveHandshake{token: "signed-token"},
	})

	return &reserveServerHarness{
		srv:      New(slog.New(slog.NewTextHandler(io.Discard, nil)), app),
		shipment: shipment,
		route:    route,
		reserves: reserves,
	}
}

func sourceAndTarget() (*entities.Shipment, *entities.Route, uuid.UUID, uuid.UUID) {
	shipment, route := assignedRoute()
	return shipment, route, uuid.UUID(shipment.ID), uuid.UUID(route.Points[0].ID)
}

func assignedRoute() (*entities.Shipment, *entities.Route) {
	sourceID := values.NewID()
	targetID := values.NewID()
	shipment := entities.NewShipment()
	_ = shipment.AssignDriver(values.NewID(), 42)
	route := &entities.Route{
		ID:         values.NewID(),
		ShipmentID: shipment.ID,
		Points: []*entities.RoutePoint{
			{ID: sourceID, PointID: 101, SeqNum: 0},
			{ID: values.NewID(), PointID: 102, SeqNum: 1},
			{ID: targetID, PointID: 103, SeqNum: 2},
		},
	}
	return shipment, route
}

// ==== tests: 3.4 reserve handlers ====

func TestCreateReserve_Success(t *testing.T) {
	ctx := context.Background()
	shipment, route, shipmentID, sourceID := sourceAndTarget()
	driverID := values.NewID()

	h := newReserveServer(&fakeDriverProvider{driverID: driverID}, reserveActiveStore(uuid.UUID(shipmentID), uuid.UUID(route.ID), sourceID), shipment, route, 0)

	_, err := h.srv.CreateReserve(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("CreateReserve: %v", err)
	}
	if h.shipment.Status != values.ShipmentStatusReserveLoading {
		t.Fatalf("expected RESERVE_LOADING, got %v", h.shipment.Status)
	}
}

func TestCreateReserve_NoActiveShipment(t *testing.T) {
	ctx := context.Background()
	shipment, route, _, _ := sourceAndTarget()

	h := newReserveServer(&fakeDriverProvider{driverID: values.NewID()}, &fakeShipmentStore{}, shipment, route, 0)

	_, err := h.srv.CreateReserve(ctx, &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected an error when the driver has no active shipment")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", status.Code(err))
	}
}

func TestCreateReserve_AlreadyReserved(t *testing.T) {
	ctx := context.Background()
	shipment, route, shipmentID, sourceID := sourceAndTarget()

	h := newReserveServer(&fakeDriverProvider{driverID: values.NewID()}, reserveActiveStore(uuid.UUID(shipmentID), uuid.UUID(route.ID), sourceID), shipment, route, 1)

	_, err := h.srv.CreateReserve(ctx, &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected an error for an existing reserve")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", status.Code(err))
	}
	if h.shipment.Status != values.ShipmentStatusAssigned {
		t.Fatalf("shipment must not transition, got %v", h.shipment.Status)
	}
}

func TestGetReserveQRStatus_Success(t *testing.T) {
	ctx := context.Background()
	shipment, route, shipmentID, sourceID := sourceAndTarget()

	h := newReserveServer(&fakeDriverProvider{driverID: values.NewID()}, reserveActiveStore(uuid.UUID(shipmentID), uuid.UUID(route.ID), sourceID), shipment, route, 0)

	scannedAt := values.Now().Add(-time.Minute)
	h.reserves.getReserveByShipmentIDFn = func(_ context.Context, _ values.ShipmentID) (*entities.Reserve, error) {
		return &entities.Reserve{
			ID:           values.NewID(),
			ShipmentID:   shipment.ID,
			RoutePointID: route.Points[0].ID,
			Status:       values.ReserveStatusActive,
			ScannedAt:    &scannedAt,
		}, nil
	}

	if _, err := h.srv.GetReserveQRStatus(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("GetReserveQRStatus: %v", err)
	}
}

func TestGetReserveQRStatus_NoActiveShipment(t *testing.T) {
	ctx := context.Background()
	shipment, route, _, _ := sourceAndTarget()

	h := newReserveServer(&fakeDriverProvider{driverID: values.NewID()}, &fakeShipmentStore{}, shipment, route, 0)

	_, err := h.srv.GetReserveQRStatus(ctx, &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected an error when the driver has no active shipment")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", status.Code(err))
	}
}

func TestUpdateReserveQRStatus_Success(t *testing.T) {
	ctx := context.Background()
	shipment, route, shipmentID, sourceID := sourceAndTarget()
	if err := shipment.ReserveLoading(); err != nil {
		t.Fatal(err)
	}

	h := newReserveServer(&fakeDriverProvider{driverID: values.NewID()}, reserveActiveStore(uuid.UUID(shipmentID), uuid.UUID(route.ID), sourceID), shipment, route, 0)

	reserve := &entities.Reserve{
		ID:           values.NewID(),
		ShipmentID:   shipment.ID,
		RoutePointID: route.Points[0].ID,
		Status:       values.ReserveStatusWaitingApproval,
		Task:         values.ReserveTask{ID: "task-7"},
	}
	h.reserves.getReserveByRoutePointIDFn = func(_ context.Context, _ values.RoutePointID) (*entities.Reserve, error) {
		return reserve, nil
	}

	if _, err := h.srv.UpdateReserveQRStatus(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("UpdateReserveQRStatus: %v", err)
	}
	if reserve.Status != values.ReserveStatusActive {
		t.Fatalf("expected reserve ACTIVE, got %v", reserve.Status)
	}
	if h.shipment.Status != values.ShipmentStatusLoading {
		t.Fatalf("expected shipment LOADING, got %v", h.shipment.Status)
	}
}

func TestUpdateReserveQRStatus_NotFound(t *testing.T) {
	ctx := context.Background()
	shipment, route, shipmentID, sourceID := sourceAndTarget()
	if err := shipment.ReserveLoading(); err != nil {
		t.Fatal(err)
	}

	h := newReserveServer(&fakeDriverProvider{driverID: values.NewID()}, reserveActiveStore(uuid.UUID(shipmentID), uuid.UUID(route.ID), sourceID), shipment, route, 0)

	// reserveRepo.GetReserveByRoutePointID default returns ErrReserveNotFound.
	_, err := h.srv.UpdateReserveQRStatus(ctx, &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected an error when the reserve is not found")
	}
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", status.Code(err))
	}
}

func TestCancelReserve_Success(t *testing.T) {
	ctx := context.Background()
	shipment, route, shipmentID, sourceID := sourceAndTarget()
	if err := shipment.ReserveLoading(); err != nil {
		t.Fatal(err)
	}

	h := newReserveServer(&fakeDriverProvider{driverID: values.NewID()}, reserveActiveStore(uuid.UUID(shipmentID), uuid.UUID(route.ID), sourceID), shipment, route, 0)

	h.reserves.getReserveByShipmentIDFn = func(_ context.Context, _ values.ShipmentID) (*entities.Reserve, error) {
		return &entities.Reserve{
			ID:           values.NewID(),
			ShipmentID:   shipment.ID,
			RoutePointID: route.Points[0].ID,
			Status:       values.ReserveStatusQueued,
		}, nil
	}

	if _, err := h.srv.CancelReserve(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("CancelReserve: %v", err)
	}
	if h.shipment.Status != values.ShipmentStatusAssigned {
		t.Fatalf("expected shipment back to ASSIGNED, got %v", h.shipment.Status)
	}
}

func TestCancelReserve_Active(t *testing.T) {
	ctx := context.Background()
	shipment, route, shipmentID, sourceID := sourceAndTarget()
	if err := shipment.ReserveLoading(); err != nil {
		t.Fatal(err)
	}

	h := newReserveServer(&fakeDriverProvider{driverID: values.NewID()}, reserveActiveStore(uuid.UUID(shipmentID), uuid.UUID(route.ID), sourceID), shipment, route, 0)

	h.reserves.getReserveByShipmentIDFn = func(_ context.Context, _ values.ShipmentID) (*entities.Reserve, error) {
		return &entities.Reserve{
			ID:           values.NewID(),
			ShipmentID:   shipment.ID,
			RoutePointID: route.Points[0].ID,
			Status:       values.ReserveStatusActive,
		}, nil
	}

	_, err := h.srv.CancelReserve(ctx, &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected an error when canceling an active reserve")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", status.Code(err))
	}
}