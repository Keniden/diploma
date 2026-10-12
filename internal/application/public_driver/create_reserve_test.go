package publicdriver

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	shipping_application "last-mile-task/internal/shipping/application"
	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/pkg/qr"
)

// ==== port mocks (mirror internal/shipping/application/mocks_test.go) ====

type mockTransactional struct {
	withTxFn func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (m *mockTransactional) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.withTxFn(ctx, fn)
}

func newMockTransactional() *mockTransactional {
	return &mockTransactional{
		withTxFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		},
	}
}

type mockShipmentRepository struct {
	getShipmentByIDFn func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error)
	saveShipmentFn    func(ctx context.Context, shipment *entities.Shipment) error
}

func (m *mockShipmentRepository) GetShipmentByID(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
	return m.getShipmentByIDFn(ctx, id)
}

func (m *mockShipmentRepository) SaveShipment(ctx context.Context, shipment *entities.Shipment) error {
	return m.saveShipmentFn(ctx, shipment)
}

type mockRouteRepository struct {
	getRouteByIDFn         func(ctx context.Context, id values.RouteID) (*entities.Route, error)
	getRouteByShipmentIDFn func(ctx context.Context, id values.ShipmentID) (*entities.Route, error)
	saveRouteFn            func(ctx context.Context, route *entities.Route) error
}

func (m *mockRouteRepository) GetRouteByID(ctx context.Context, id values.RouteID) (*entities.Route, error) {
	return m.getRouteByIDFn(ctx, id)
}

func (m *mockRouteRepository) GetRouteByShipmentID(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
	return m.getRouteByShipmentIDFn(ctx, id)
}

func (m *mockRouteRepository) SaveRoute(ctx context.Context, route *entities.Route) error {
	return m.saveRouteFn(ctx, route)
}

type mockReserveRepository struct {
	getReserveByIDFn                     func(ctx context.Context, id values.ReserveID) (*entities.Reserve, error)
	getReserveByShipmentIDFn             func(ctx context.Context, shipmentID values.ShipmentID) (*entities.Reserve, error)
	getReserveByRoutePointIDFn           func(ctx context.Context, routePointID values.RoutePointID) (*entities.Reserve, error)
	listExpiredWaitingApprovalReservesFn func(ctx context.Context, limit int) ([]*entities.Reserve, error)
	listExpiredActiveReservesFn          func(ctx context.Context, limit int) ([]*entities.Reserve, error)
	saveReserveFn                        func(ctx context.Context, reserve *entities.Reserve) error
}

func (m *mockReserveRepository) GetReserveByID(ctx context.Context, id values.ReserveID) (*entities.Reserve, error) {
	if m.getReserveByIDFn != nil {
		return m.getReserveByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockReserveRepository) GetReserveByShipmentID(ctx context.Context, shipmentID values.ShipmentID) (*entities.Reserve, error) {
	return m.getReserveByShipmentIDFn(ctx, shipmentID)
}

func (m *mockReserveRepository) GetReserveByRoutePointID(ctx context.Context, routePointID values.RoutePointID) (*entities.Reserve, error) {
	return m.getReserveByRoutePointIDFn(ctx, routePointID)
}

func (m *mockReserveRepository) ListExpiredWaitingApprovalReserves(ctx context.Context, limit int) ([]*entities.Reserve, error) {
	if m.listExpiredWaitingApprovalReservesFn != nil {
		return m.listExpiredWaitingApprovalReservesFn(ctx, limit)
	}
	return nil, nil
}

func (m *mockReserveRepository) ListExpiredActiveReserves(ctx context.Context, limit int) ([]*entities.Reserve, error) {
	if m.listExpiredActiveReservesFn != nil {
		return m.listExpiredActiveReservesFn(ctx, limit)
	}
	return nil, nil
}

func (m *mockReserveRepository) SaveReserve(ctx context.Context, reserve *entities.Reserve) error {
	return m.saveReserveFn(ctx, reserve)
}

type mockReserveManager struct {
	reserveOrQueueFn     func(ctx context.Context, shipment *entities.Shipment, source *entities.RoutePoint, targets []*entities.RoutePoint) (*entities.Reserve, error)
	cancelReserveFn      func(ctx context.Context, reserve *entities.Reserve, pointID values.PointID) error
	approveReserveTaskFn func(ctx context.Context, task values.ReserveTask) (time.Time, error)
}

func (m *mockReserveManager) ReserveOrQueue(ctx context.Context, shipment *entities.Shipment, source *entities.RoutePoint, targets []*entities.RoutePoint) (*entities.Reserve, error) {
	return m.reserveOrQueueFn(ctx, shipment, source, targets)
}

func (m *mockReserveManager) CancelReserve(ctx context.Context, reserve *entities.Reserve, pointID values.PointID) error {
	if m.cancelReserveFn != nil {
		return m.cancelReserveFn(ctx, reserve, pointID)
	}
	return nil
}

func (m *mockReserveManager) ApproveReserveTask(ctx context.Context, task values.ReserveTask) (time.Time, error) {
	if m.approveReserveTaskFn != nil {
		return m.approveReserveTaskFn(ctx, task)
	}
	return values.Now().Add(time.Hour), nil
}

type mockReserveEventPublisher struct {
	publishFn func(ctx context.Context, event events.ReserveEvent) error
}

func (m *mockReserveEventPublisher) PublishReserveEvent(ctx context.Context, event events.ReserveEvent) error {
	return m.publishFn(ctx, event)
}

type mockShipmentPolicyRepository struct {
	getShipmentByIDFn           func(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error)
	getRouteByShipmentIDFn      func(ctx context.Context, id values.ShipmentID) (*entities.Route, error)
	getShipmentCountByDriverFn  func(ctx context.Context, driverID values.DriverID, statuses ...values.ShipmentStatus) (int, error)
	getPackageCountByShipmentFn func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.PackageStatus) (int, error)
	getPackagesByShipmentIDFn   func(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error)
	getReserveCountByShipmentFn func(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.ReserveStatus) (int, error)
	hasSourcePackagesFn         func(ctx context.Context, route *entities.Route) (bool, error)
}

func (m *mockShipmentPolicyRepository) GetShipmentByID(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
	return m.getShipmentByIDFn(ctx, id)
}

func (m *mockShipmentPolicyRepository) GetRouteByShipmentID(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
	return m.getRouteByShipmentIDFn(ctx, id)
}

func (m *mockShipmentPolicyRepository) GetShipmentCountByDriverID(ctx context.Context, driverID values.DriverID, statuses ...values.ShipmentStatus) (int, error) {
	return m.getShipmentCountByDriverFn(ctx, driverID, statuses...)
}

func (m *mockShipmentPolicyRepository) GetPackageCountByShipmentID(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.PackageStatus) (int, error) {
	return m.getPackageCountByShipmentFn(ctx, shipmentID, statuses...)
}

func (m *mockShipmentPolicyRepository) GetPackagesByShipmentID(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error) {
	if m.getPackagesByShipmentIDFn != nil {
		return m.getPackagesByShipmentIDFn(ctx, shipmentID)
	}
	return nil, nil
}

func (m *mockShipmentPolicyRepository) GetReserveCountByShipmentID(ctx context.Context, shipmentID values.ShipmentID, statuses ...values.ReserveStatus) (int, error) {
	return m.getReserveCountByShipmentFn(ctx, shipmentID, statuses...)
}

func (m *mockShipmentPolicyRepository) HasSourcePackages(ctx context.Context, route *entities.Route) (bool, error) {
	if m.hasSourcePackagesFn != nil {
		return m.hasSourcePackagesFn(ctx, route)
	}
	return true, nil
}

type mockHandshake struct {
	sessions []qr.Session
	token    string
	err      error
	decoded  qr.Session
	decodeErr error
}

func (m *mockHandshake) CreateSession(session qr.Session) (string, error) {
	m.sessions = append(m.sessions, session)
	if m.err != nil {
		return "", m.err
	}
	return m.token, nil
}

func (m *mockHandshake) DecodeQR(_ string) (qr.Session, error) {
	if m.decodeErr != nil {
		return qr.Session{}, m.decodeErr
	}
	return m.decoded, nil
}

// ==== harness ====

type reserveAppOptions struct {
	shipment *entities.Shipment
	route    *entities.Route
	count    int
}

type reserveHarness struct {
	app         *Application
	reserveMgr  *mockReserveManager
	reserves    *mockReserveRepository
	shipments   *mockShipmentRepository
	routes      *mockRouteRepository
	reserveEvts *mockReserveEventPublisher
	handshake   *mockHandshake
}

func assignedShipmentWithRoute() (*entities.Shipment, *entities.Route) {
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

func inProgressShipmentWithRoute() (*entities.Shipment, *entities.Route) {
	shipment, route := assignedShipmentWithRoute()
	_ = shipment.ReserveLoading()
	_ = shipment.Loading()
	_ = shipment.Start()
	return shipment, route
}

func newReserveApp(t *testing.T, opts reserveAppOptions) *reserveHarness {
	t.Helper()

	reserveMgr := &mockReserveManager{
		reserveOrQueueFn: func(_ context.Context, shipment *entities.Shipment, source *entities.RoutePoint, _ []*entities.RoutePoint) (*entities.Reserve, error) {
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
		},
	}

	policyRepo := &mockShipmentPolicyRepository{
		getShipmentByIDFn: func(_ context.Context, id values.ShipmentID) (*entities.Shipment, error) {
			if opts.shipment != nil {
				return opts.shipment, nil
			}
			return nil, ports.ErrShipmentNotFound
		},
		getRouteByShipmentIDFn: func(_ context.Context, _ values.ShipmentID) (*entities.Route, error) {
			return opts.route, nil
		},
		getReserveCountByShipmentFn: func(_ context.Context, _ values.ShipmentID, _ ...values.ReserveStatus) (int, error) {
			return opts.count, nil
		},
	}
	policy := policies.NewShipmentPolicy(policyRepo)

	shipments := &mockShipmentRepository{
		getShipmentByIDFn: func(_ context.Context, id values.ShipmentID) (*entities.Shipment, error) {
			if opts.shipment != nil {
				return opts.shipment, nil
			}
			return nil, ports.ErrShipmentNotFound
		},
		saveShipmentFn: func(_ context.Context, _ *entities.Shipment) error { return nil },
	}
	routes := &mockRouteRepository{
		getRouteByShipmentIDFn: func(_ context.Context, _ values.ShipmentID) (*entities.Route, error) {
			return opts.route, nil
		},
		saveRouteFn: func(_ context.Context, _ *entities.Route) error { return nil },
	}
	reserves := &mockReserveRepository{
		saveReserveFn: func(_ context.Context, _ *entities.Reserve) error { return nil },
	}
	reserveEvts := &mockReserveEventPublisher{
		publishFn: func(_ context.Context, _ events.ReserveEvent) error { return nil },
	}

	tx := newMockTransactional()
	svc := shipping_application.NewReserveService(
		reserves,
		shipments,
		routes,
		reserveMgr,
		policy,
		reserveEvts,
		tx,
	)

	handshake := &mockHandshake{token: "signed-token"}

	return &reserveHarness{
		app: &Application{
			reserveService: svc,
			handshake:      handshake,
		},
		reserveMgr:  reserveMgr,
		reserves:    reserves,
		shipments:   shipments,
		routes:      routes,
		reserveEvts: reserveEvts,
		handshake:   handshake,
	}
}

// ==== tests: 3.1 CreateReserve ====

func TestCreateReserve_Source_Success(t *testing.T) {
	ctx := context.Background()
	shipment, route := assignedShipmentWithRoute()
	h := newReserveApp(t, reserveAppOptions{shipment: shipment, route: route, count: 0})

	res, err := h.app.CreateReserve(ctx, CreateReserveRequest{
		ShipmentID:   uuid.UUID(shipment.ID),
		RoutePointID: uuid.UUID(route.Points[0].ID),
	})
	if err != nil {
		t.Fatalf("CreateReserve: %v", err)
	}
	if res.Session != "signed-token" {
		t.Fatalf("expected the handshake session, got %q", res.Session)
	}
	if res.ReserveID == uuid.Nil {
		t.Fatal("expected a non-nil reserve id")
	}
	if shipment.Status != values.ShipmentStatusReserveLoading {
		t.Fatalf("expected RESERVE_LOADING, got %v", shipment.Status)
	}

	if len(h.handshake.sessions) != 1 {
		t.Fatalf("expected 1 handshake session, got %d", len(h.handshake.sessions))
	}
	sess := h.handshake.sessions[0]
	if sess.ShipmentID != uuid.UUID(shipment.ID).String() {
		t.Fatalf("session shipment id: got %s, want %s", sess.ShipmentID, shipment.ID)
	}
	if sess.RoutePointID != uuid.UUID(route.Points[0].ID).String() {
		t.Fatalf("session route point id: got %s, want %s", sess.RoutePointID, route.Points[0].ID)
	}
	if sess.TaskID != "task-7" {
		t.Fatalf("session task id: got %s, want task-7", sess.TaskID)
	}
}

func TestCreateReserve_Target_Success(t *testing.T) {
	ctx := context.Background()
	shipment, route := inProgressShipmentWithRoute()
	h := newReserveApp(t, reserveAppOptions{shipment: shipment, route: route, count: 0})

	target, err := route.Target()
	if err != nil {
		t.Fatal(err)
	}

	res, err := h.app.CreateReserve(ctx, CreateReserveRequest{
		ShipmentID:   uuid.UUID(shipment.ID),
		RoutePointID: uuid.UUID(target.ID),
	})
	if err != nil {
		t.Fatalf("CreateReserve: %v", err)
	}
	if res.Session == "" {
		t.Fatal("expected a handshake session for the target reserve")
	}
	if shipment.Status != values.ShipmentStatusReserveUnloading {
		t.Fatalf("expected RESERVE_UNLOADING, got %v", shipment.Status)
	}
}

func TestCreateReserve_Conflict_NoStateChange(t *testing.T) {
	ctx := context.Background()
	shipment, route := assignedShipmentWithRoute()
	h := newReserveApp(t, reserveAppOptions{shipment: shipment, route: route, count: 1})

	_, err := h.app.CreateReserve(ctx, CreateReserveRequest{
		ShipmentID:   uuid.UUID(shipment.ID),
		RoutePointID: uuid.UUID(route.Points[0].ID),
	})
	if err == nil {
		t.Fatal("expected an error for an existing reserve")
	}
	if shipment.Status != values.ShipmentStatusAssigned {
		t.Fatalf("shipment must not transition on conflict, got %v", shipment.Status)
	}
	if len(h.handshake.sessions) != 0 {
		t.Fatalf("handshake must not be issued on conflict, got %d sessions", len(h.handshake.sessions))
	}
}

func TestCreateReserve_HandshakeError_Fails(t *testing.T) {
	ctx := context.Background()
	shipment, route := assignedShipmentWithRoute()
	h := newReserveApp(t, reserveAppOptions{shipment: shipment, route: route, count: 0})
	h.handshake.err = context.DeadlineExceeded

	_, err := h.app.CreateReserve(ctx, CreateReserveRequest{
		ShipmentID:   uuid.UUID(shipment.ID),
		RoutePointID: uuid.UUID(route.Points[0].ID),
	})
	if err == nil {
		t.Fatal("expected an error when the handshake fails")
	}
}