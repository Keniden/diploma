//go:build e2e

// End-to-end tests of the public driver create-shipment vertical: real
// PostgreSQL (migrations + clean tables), real gRPC server with the driver
// wiring, real write repositories and the in-memory sync event bus with the
// cargo-shipment handler. Only the external provider ports (route template,
// package demand, driver) are faked — the same loose ports the production
// stubs stand in for (TODO LASTMILE-110/111/112).
//
// Run: go test -tags e2e -run 'TestE2ECreateShipment' -v ./internal/platform/application
package application_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	publicdriver "last-mile-task/internal/application/public_driver"
	drivergrpc "last-mile-task/internal/application/public_driver/transport/grpc"
	"last-mile-task/internal/platform/database"
	shipping_application "last-mile-task/internal/shipping/application"
	"last-mile-task/internal/shipping/application/demand"
	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure"
	pgdriver "last-mile-task/internal/shipping/infrastructure/driver/postgres"
	"last-mile-task/internal/shipping/infrastructure/events/bus/inmem"
	"last-mile-task/internal/shipping/infrastructure/events/handlers"
	"last-mile-task/internal/shipping/infrastructure/postgrestest"
	shipmentq "last-mile-task/internal/shipping/queries/driver/shipment"
	driverpb "last-mile-task/pkg/gen/api/proto/v1/public/driver"
)

// --- fakes -----------------------------------------------------------------

type csDriverProvider struct{ driver values.Driver }

func (p *csDriverProvider) GetDriverFromContext(context.Context) (values.Driver, error) {
	return p.driver, nil
}

type csTemplateProvider struct{ byID map[values.RouteTemplateID]values.RouteTemplate }

func (p *csTemplateProvider) GetRouteTemplateByID(
	_ context.Context, id values.RouteTemplateID,
) (values.RouteTemplate, error) {
	tpl, ok := p.byID[id]
	if !ok {
		return values.RouteTemplate{}, ports.ErrRouteTemplateNotFound
	}
	return tpl, nil
}

type csDemandProvider struct{}

func (csDemandProvider) GetPackageDemandByRoute(
	_ context.Context, route *entities.Route,
) ([]values.PackageDemand, error) {
	source, err := route.Source()
	if err != nil {
		return nil, err
	}
	target, err := route.Target()
	if err != nil {
		return nil, err
	}
	return []values.PackageDemand{{
		PackageID:     values.NewID(),
		TareID:        1,
		TareType:      "BOX",
		LoadPointID:   source.ID,
		UnloadPointID: target.ID,
		Direction:     values.PackageDemandDirectionForward,
	}}, nil
}

type csSourcePackages struct{}

func (csSourcePackages) HasSourcePackages(context.Context, *entities.Route) (bool, error) {
	return true, nil
}

// recordingCargo records the cargo-shipment calls received via the sync bus.
type recordingCargo struct {
	createCalls []handlers.CargoShipmentCreateRequest
	setCalls    []handlers.CargoShipmentSetTransportRequest
	driveCalls  []handlers.CargoShipmentDriveShipmentRequest
}

func (c *recordingCargo) CreateShipment(
	_ context.Context, req handlers.CargoShipmentCreateRequest,
) error {
	c.createCalls = append(c.createCalls, req)
	return nil
}

func (c *recordingCargo) SetTransport(
	_ context.Context, req handlers.CargoShipmentSetTransportRequest,
) error {
	c.setCalls = append(c.setCalls, req)
	return nil
}

func (c *recordingCargo) DriveShipment(
	_ context.Context, req handlers.CargoShipmentDriveShipmentRequest,
) error {
	c.driveCalls = append(c.driveCalls, req)
	return nil
}

// --- harness ----------------------------------------------------------------

type csHarness struct {
	client     driverpb.DriverClient
	conn       *grpc.ClientConn
	grpcServer *grpc.Server
	cargo      *recordingCargo
	repo       *database.Repository
	pool       *pgxpool.Pool
	driver     values.Driver
}

// newCSHarness wires the create-shipment slice over real PostgreSQL using the
// real repositories, event bus, handlers and the gRPC server over bufconn.
func newCSHarness(
	t *testing.T,
	templates ports.RouteTemplateProvider,
	driver values.Driver,
) *csHarness {
	t.Helper()

	repo := postgrestest.Repo(t)
	postgrestest.CleanTables(t, repo.Pool)

	entityRepo := infrastructure.NewEntityRepository(repo)

	shippingBus := inmem.NewSyncEventBus()
	cargo := &recordingCargo{}
	shipmentEventHandler := handlers.NewShipmentEventHandler(cargo)
	shippingBus.SubscribeShipmentEvent(events.ShipmentCreated,
		inmem.TypedHandler(shipmentEventHandler.HandleShipmentCreatedEvent))
	shippingBus.SubscribeShipmentEvent(events.ShipmentDriverAssign,
		inmem.TypedHandler(shipmentEventHandler.HandleShipmentDriverAssignEvent))
	shippingBus.SubscribeShipmentEvent(events.ShipmentTransportAssign,
		inmem.TypedHandler(shipmentEventHandler.HandleShipmentTransportAssignEvent))

	policy := policies.NewShipmentPolicy(
		infrastructure.NewShipmentPolicyRepository(
			entityRepo.ShipmentRepository,
			entityRepo.RouteRepository,
			entityRepo.PackageRepository,
			entityRepo.ReserveRepository,
			csSourcePackages{},
		),
	)
	assembler := shipping_application.NewAssembler(
		entityRepo.ShipmentRepository,
		policy,
		entityRepo.RouteRepository,
		entityRepo.PackageRepository,
		shippingBus,
		repo,
	)
	progress := shipping_application.NewShipmentProgressService(
		policy,
		entityRepo.ShipmentRepository,
		shippingBus,
		repo,
	)
	demandService := demand.NewDemandService(
		entityRepo.RouteRepository,
		entityRepo.PackageRepository,
		csDemandProvider{},
		repo,
	)

	shippingQueries := shipmentq.NewQueries(
		pgdriver.New(repo),
		infrastructure.NewAddressResolver(),
	)

	app := publicdriver.NewForTesting(
		&csDriverProvider{driver: driver},
		shippingQueries,
		nil,
		assembler,
		progress,
		nil,
		demandService,
		templates,
		repo,
	)

	srv := drivergrpc.New(slog.New(slog.NewTextHandler(io.Discard, nil)), app)

	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	srv.RegisterGRPC(grpcServer)
	go func() { _ = grpcServer.Serve(lis) }()
	t.Cleanup(grpcServer.Stop)

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return &csHarness{
		client:     driverpb.NewDriverClient(conn),
		conn:       conn,
		grpcServer: grpcServer,
		cargo:      cargo,
		repo:       repo,
		pool:       repo.Pool,
		driver:     driver,
	}
}

// --- fixtures ---------------------------------------------------------------

func csValidTemplate() values.RouteTemplate {
	return values.RouteTemplate{
		RouteTemplateID: 1,
		Distance:        250,
		Duration:        300,
		Points: []values.RoutePointTemplate{
			{RoutePointTemplateID: values.PointID(101), Distance: 0, SeqNum: 0},
			{RoutePointTemplateID: values.PointID(102), Distance: 250, SeqNum: 1},
		},
	}
}

func csShortTemplate() values.RouteTemplate {
	return values.RouteTemplate{
		RouteTemplateID: 2,
		Points: []values.RoutePointTemplate{
			{RoutePointTemplateID: values.PointID(101), Distance: 0, SeqNum: 0},
		},
	}
}

func csCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx, query, args...).Scan(&n); err != nil {
		t.Fatalf("count %q: %v", query, err)
	}
	return n
}

// --- 5.1: CreateShipment ----------------------------------------------------

// TestE2ECreateShipment proves the full creation flow: shipment, route, route
// points, packages and package_route_point rows are persisted, the driver is
// assigned, and the ShipmentCreated/ShipmentDriverAssign events reach the
// cargo-shipment stub.
func TestE2ECreateShipment(t *testing.T) {
	ctx := context.Background()
	driver := values.Driver{ID: values.NewID(), SupplierID: 7}
	h := newCSHarness(t, &csTemplateProvider{byID: map[values.RouteTemplateID]values.RouteTemplate{
		1: csValidTemplate(),
	}}, driver)

	resp, err := h.client.CreateShipment(ctx, &driverpb.CreateShipmentRequest{RouteTemplateId: 1})
	if err != nil {
		t.Fatalf("CreateShipment: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("expected success, got %q", resp.GetMessage())
	}
	shipmentID := resp.GetData().GetShipment().GetId()
	if shipmentID == "" {
		t.Fatal("expected a shipment id in the response")
	}

	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.shipment WHERE shipment_id = $1`, shipmentID); got != 1 {
		t.Fatalf("expected 1 shipment row, got %d", got)
	}
	if got := csCount(t, ctx, h.pool, `SELECT shipment_status FROM lmt.shipment WHERE shipment_id = $1`, shipmentID); got != int(values.ShipmentStatusAssigned) {
		t.Fatalf("expected shipment ASSIGNED, got status %d", got)
	}
	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.route WHERE shipment_id = $1`, shipmentID); got != 1 {
		t.Fatalf("expected 1 route row, got %d", got)
	}
	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.route_point WHERE shipment_id = $1`, shipmentID); got != 2 {
		t.Fatalf("expected 2 route point rows, got %d", got)
	}
	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.package WHERE shipment_id = $1`, shipmentID); got != 1 {
		t.Fatalf("expected 1 package row, got %d", got)
	}
	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.package_route_point WHERE shipment_id = $1`, shipmentID); got != 2 {
		t.Fatalf("expected 2 package_route_point rows, got %d", got)
	}

	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.shipment WHERE driver_id = $1`, uuid.UUID(driver.ID).String()); got != 1 {
		t.Fatalf("expected driver to be assigned to 1 shipment, got %d", got)
	}

	if len(h.cargo.createCalls) != 1 {
		t.Fatalf("expected 1 ShipmentCreated cargo call, got %d", len(h.cargo.createCalls))
	}
	created := h.cargo.createCalls[0]
	if created.ShipmentID.String() != shipmentID || created.SourcePointID != 101 {
		t.Fatalf("unexpected ShipmentCreated payload: shipment=%s source=%d", created.ShipmentID, created.SourcePointID)
	}

	foundDriverAssign := false
	for _, call := range h.cargo.setCalls {
		if call.DriverID == driver.ID.String() && call.CarrierID == "7" {
			foundDriverAssign = true
		}
		if call.VehicleNumber != "" {
			t.Fatalf("CreateShipment must not emit a vehicle assign, got %+v", call)
		}
	}
	if !foundDriverAssign {
		t.Fatalf("expected a ShipmentDriverAssign cargo call for driver %s, got %+v", driver.ID, h.cargo.setCalls)
	}
}

// TestE2ECreateShipmentTemplateNotFound proves an unknown template maps to
// NotFound and leaves no state behind.
func TestE2ECreateShipmentTemplateNotFound(t *testing.T) {
	ctx := context.Background()
	driver := values.Driver{ID: values.NewID()}
	h := newCSHarness(t, &csTemplateProvider{byID: map[values.RouteTemplateID]values.RouteTemplate{}}, driver)

	_, err := h.client.CreateShipment(ctx, &driverpb.CreateShipmentRequest{RouteTemplateId: 999})
	if err == nil {
		t.Fatal("expected error for unknown template")
	}
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", status.Code(err))
	}
	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.shipment`); got != 0 {
		t.Fatalf("expected no shipment rows, got %d", got)
	}
	if len(h.cargo.createCalls) != 0 {
		t.Fatalf("expected no cargo calls, got %d", len(h.cargo.createCalls))
	}
}

// TestE2ECreateShipmentTemplateTooShort proves a template with fewer than 2
// points maps to InvalidArgument and leaves no state behind.
func TestE2ECreateShipmentTemplateTooShort(t *testing.T) {
	ctx := context.Background()
	driver := values.Driver{ID: values.NewID()}
	h := newCSHarness(t, &csTemplateProvider{byID: map[values.RouteTemplateID]values.RouteTemplate{
		2: csShortTemplate(),
	}}, driver)

	_, err := h.client.CreateShipment(ctx, &driverpb.CreateShipmentRequest{RouteTemplateId: 2})
	if err == nil {
		t.Fatal("expected error for a too-short template")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.shipment`); got != 0 {
		t.Fatalf("expected no shipment rows, got %d", got)
	}
}

// --- 5.2: CreateShipmentWithTransport ---------------------------------------

// TestE2ECreateShipmentWithTransport proves the atomic success path: the
// shipment is created, driver and vehicle assigned, the plate stored, and the
// ShipmentTransportAssign event reaches the cargo-shipment stub.
func TestE2ECreateShipmentWithTransport(t *testing.T) {
	ctx := context.Background()
	driver := values.Driver{ID: values.NewID(), SupplierID: 7}
	h := newCSHarness(t, &csTemplateProvider{byID: map[values.RouteTemplateID]values.RouteTemplate{
		1: csValidTemplate(),
	}}, driver)

	resp, err := h.client.CreateShipmentWithTransport(ctx, &driverpb.CreateShipmentWithTransportRequest{
		RouteTemplateId: 1,
		CarLicensePlate: "aa 1234",
	})
	if err != nil {
		t.Fatalf("CreateShipmentWithTransport: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("expected success, got %q", resp.GetMessage())
	}
	shipmentID := resp.GetData().GetShipment().GetId()
	if shipmentID == "" {
		t.Fatal("expected a shipment id in the response")
	}

	if plate := csText(t, ctx, h.pool, `SELECT car_license_plate FROM lmt.shipment WHERE shipment_id = $1`, shipmentID); plate != "AA1234" {
		t.Fatalf("expected normalized plate AA1234, got %q", plate)
	}

	foundVehicle := false
	for _, call := range h.cargo.setCalls {
		if call.VehicleNumber == "AA1234" {
			foundVehicle = true
		}
	}
	if !foundVehicle {
		t.Fatalf("expected a ShipmentTransportAssign cargo call with plate AA1234, got %+v", h.cargo.setCalls)
	}
}

// TestE2ECreateShipmentWithTransportEmptyPlate proves a failed step never
// leaves a partially created shipment: the outer transaction is rolled back,
// so no active shipment remains.
func TestE2ECreateShipmentWithTransportEmptyPlate(t *testing.T) {
	ctx := context.Background()
	driver := values.Driver{ID: values.NewID(), SupplierID: 7}
	h := newCSHarness(t, &csTemplateProvider{byID: map[values.RouteTemplateID]values.RouteTemplate{
		1: csValidTemplate(),
	}}, driver)

	_, err := h.client.CreateShipmentWithTransport(ctx, &driverpb.CreateShipmentWithTransportRequest{
		RouteTemplateId: 1,
	})
	if err == nil {
		t.Fatal("expected error for an empty plate")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.shipment`); got != 0 {
		t.Fatalf("expected no shipment rows after rollback, got %d", got)
	}
	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.package_route_point`); got != 0 {
		t.Fatalf("expected no package_route_point rows after rollback, got %d", got)
	}
	// The flow reached the driver-assign step before the plate error; the
	// vehicle assign must never be emitted.
	for _, call := range h.cargo.setCalls {
		if call.VehicleNumber != "" {
			t.Fatalf("vehicle assign must not be emitted on plate failure, got %+v", call)
		}
	}
}

// --- helpers ---------------------------------------------------------------

func csText(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(ctx, query, args...).Scan(&s); err != nil {
		t.Fatalf("text %q: %v", query, err)
	}
	return s
}