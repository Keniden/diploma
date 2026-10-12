package application_test

import (
	"context"
	"testing"
	"time"

	"last-mile-task/internal/shipping/application"
	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/domain/values"
)

type reserveServiceHarness struct {
	svc           *application.ReserveService
	reserveMgr    *mockReserveManager
	reserves      *mockReserveRepository
	shipments     *mockShipmentRepository
	routes        *mockRouteRepository
	reserveEvents *mockReserveEventPublisher
}

type reserveServiceOptions struct {
	shipment *entities.Shipment
	route    *entities.Route
	count    int
}

func newReserveService(t *testing.T, opts reserveServiceOptions) *reserveServiceHarness {
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
		getRouteByShipmentIDFn: func(_ context.Context, id values.ShipmentID) (*entities.Route, error) {
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
	reserveEvents := &mockReserveEventPublisher{
		publishFn: func(_ context.Context, _ events.ReserveEvent) error { return nil },
	}

	tx := &mockTransactional{
		withTxFn: func(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) },
	}

	return &reserveServiceHarness{
		svc: application.NewReserveService(
			reserves,
			shipments,
			routes,
			reserveMgr,
			policy,
			reserveEvents,
			tx,
		),
		reserveMgr:    reserveMgr,
		reserves:      reserves,
		shipments:     shipments,
		routes:        routes,
		reserveEvents: reserveEvents,
	}
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

func TestReserveService_QueueReserveByShipmentID_Source(t *testing.T) {
	ctx := context.Background()
	shipment, route := assignedShipmentWithRoute()
	h := newReserveService(t, reserveServiceOptions{shipment: shipment, route: route, count: 0})

	reserve, err := h.svc.QueueReserveByShipmentID(ctx, shipment.ID, route.Points[0].ID)
	if err != nil {
		t.Fatalf("QueueReserveByShipmentID: %v", err)
	}
	if reserve.RoutePointID != route.Points[0].ID {
		t.Fatalf("reserve route point: got %v, want %v", reserve.RoutePointID, route.Points[0].ID)
	}
	if shipment.Status != values.ShipmentStatusReserveLoading {
		t.Fatalf("expected RESERVE_LOADING, got %v", shipment.Status)
	}
}

func TestReserveService_QueueReserveByShipmentID_Target(t *testing.T) {
	ctx := context.Background()
	shipment, route := inProgressShipmentWithRoute()
	h := newReserveService(t, reserveServiceOptions{shipment: shipment, route: route, count: 0})

	target, err := route.Target()
	if err != nil {
		t.Fatal(err)
	}

	reserve, err := h.svc.QueueReserveByShipmentID(ctx, shipment.ID, target.ID)
	if err != nil {
		t.Fatalf("QueueReserveByShipmentID: %v", err)
	}
	if reserve.RoutePointID != target.ID {
		t.Fatalf("reserve route point: got %v, want %v", reserve.RoutePointID, target.ID)
	}
	if shipment.Status != values.ShipmentStatusReserveUnloading {
		t.Fatalf("expected RESERVE_UNLOADING, got %v", shipment.Status)
	}
}

func TestReserveService_QueueReserveByShipmentID_Conflict(t *testing.T) {
	ctx := context.Background()
	shipment, route := assignedShipmentWithRoute()
	h := newReserveService(t, reserveServiceOptions{shipment: shipment, route: route, count: 1})

	_, err := h.svc.QueueReserveByShipmentID(ctx, shipment.ID, route.Points[0].ID)
	if err == nil {
		t.Fatal("expected error for an existing reserve")
	}
	if shipment.Status != values.ShipmentStatusAssigned {
		t.Fatalf("shipment must not transition on conflict, got %v", shipment.Status)
	}
}

func TestReserveService_QueueReserveByShipmentID_MiddlePoint(t *testing.T) {
	ctx := context.Background()
	shipment, route := assignedShipmentWithRoute()
	h := newReserveService(t, reserveServiceOptions{shipment: shipment, route: route, count: 0})

	_, err := h.svc.QueueReserveByShipmentID(ctx, shipment.ID, route.Points[1].ID)
	if err == nil {
		t.Fatal("expected error for a middle route point")
	}
}

func TestReserveService_CancelReserveByShipmentID_Reverts(t *testing.T) {
	ctx := context.Background()
	shipment, route := assignedShipmentWithRoute()
	h := newReserveService(t, reserveServiceOptions{shipment: shipment, route: route, count: 0})

	if _, err := h.svc.QueueReserveByShipmentID(ctx, shipment.ID, route.Points[0].ID); err != nil {
		t.Fatalf("queue: %v", err)
	}
	if shipment.Status != values.ShipmentStatusReserveLoading {
		t.Fatalf("expected RESERVE_LOADING before cancel, got %v", shipment.Status)
	}

	reserve := &entities.Reserve{
		ID:           values.NewID(),
		ShipmentID:   shipment.ID,
		RoutePointID: route.Points[0].ID,
		Status:       values.ReserveStatusQueued,
	}
	h.reserves.getReserveByShipmentIDFn = func(_ context.Context, _ values.ShipmentID) (*entities.Reserve, error) {
		return reserve, nil
	}
	shipment.Status = values.ShipmentStatusReserveLoading

	if err := h.svc.CancelReserveByShipmentID(ctx, shipment.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if reserve.Status != values.ReserveStatusCanceled {
		t.Fatalf("expected reserve CANCELED, got %v", reserve.Status)
	}
	if shipment.Status != values.ShipmentStatusAssigned {
		t.Fatalf("expected shipment back to ASSIGNED, got %v", shipment.Status)
	}
}

func TestReserveService_ApproveReserveByRoutePointID(t *testing.T) {
	ctx := context.Background()
	shipment, route := assignedShipmentWithRoute()
	h := newReserveService(t, reserveServiceOptions{shipment: shipment, route: route, count: 0})

	// Queue first to move shipment into RESERVE_LOADING.
	if _, err := h.svc.QueueReserveByShipmentID(ctx, shipment.ID, route.Points[0].ID); err != nil {
		t.Fatalf("queue: %v", err)
	}
	if shipment.Status != values.ShipmentStatusReserveLoading {
		t.Fatalf("expected RESERVE_LOADING, got %v", shipment.Status)
	}

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

	if err := h.svc.ApproveReserveByRoutePointID(ctx, shipment.ID, route.Points[0].ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if reserve.Status != values.ReserveStatusActive {
		t.Fatalf("expected reserve ACTIVE, got %v", reserve.Status)
	}
	if reserve.ScannedAt == nil {
		t.Fatal("expected scanned_at to be set on approval")
	}
	if shipment.Status != values.ShipmentStatusLoading {
		t.Fatalf("expected shipment LOADING, got %v", shipment.Status)
	}
}

func TestReserveService_ExpireDue(t *testing.T) {
	ctx := context.Background()
	shipment, route := assignedShipmentWithRoute()
	h := newReserveService(t, reserveServiceOptions{shipment: shipment, route: route, count: 0})

	waiting := &entities.Reserve{
		ID:         values.NewID(),
		ShipmentID: shipment.ID,
		Status:     values.ReserveStatusWaitingApproval,
	}
	active := &entities.Reserve{
		ID:         values.NewID(),
		ShipmentID: shipment.ID,
		Status:     values.ReserveStatusActive,
	}
	saved := make(map[values.ReserveID]values.ReserveStatus)
	h.reserves.listExpiredWaitingApprovalReservesFn = func(_ context.Context, _ int) ([]*entities.Reserve, error) {
		return []*entities.Reserve{waiting}, nil
	}
	h.reserves.listExpiredActiveReservesFn = func(_ context.Context, _ int) ([]*entities.Reserve, error) {
		return []*entities.Reserve{active}, nil
	}
	h.reserves.saveReserveFn = func(_ context.Context, r *entities.Reserve) error {
		saved[r.ID] = r.Status
		return nil
	}

	if err := h.svc.ExpireDueWaitingApprovalReserves(ctx); err != nil {
		t.Fatalf("expire waiting: %v", err)
	}
	if waiting.Status != values.ReserveStatusExpiredWaitingApproval {
		t.Fatalf("expected EXPIRED_WAITING_APPROVAL, got %v", waiting.Status)
	}
	if saved[waiting.ID] != values.ReserveStatusExpiredWaitingApproval {
		t.Fatalf("expected waiting reserve saved as expired, got %v", saved[waiting.ID])
	}

	if err := h.svc.ExpireDueActiveReserves(ctx); err != nil {
		t.Fatalf("expire active: %v", err)
	}
	if active.Status != values.ReserveStatusExpiredActive {
		t.Fatalf("expected EXPIRED_ACTIVE, got %v", active.Status)
	}
	if saved[active.ID] != values.ReserveStatusExpiredActive {
		t.Fatalf("expected active reserve saved as expired, got %v", saved[active.ID])
	}
}

func TestReserveService_GetReserveByRoutePointID(t *testing.T) {
	ctx := context.Background()
	shipment, route := assignedShipmentWithRoute()
	h := newReserveService(t, reserveServiceOptions{shipment: shipment, route: route, count: 0})

	want := &entities.Reserve{ID: values.NewID(), ShipmentID: shipment.ID}
	h.reserves.getReserveByRoutePointIDFn = func(_ context.Context, _ values.RoutePointID) (*entities.Reserve, error) {
		return want, nil
	}

	got, err := h.svc.GetReserveByRoutePointID(ctx, route.Points[0].ID)
	if err != nil {
		t.Fatalf("get reserve by route point: %v", err)
	}
	if got != want {
		t.Fatalf("expected the repository reserve back, got %+v", got)
	}
}