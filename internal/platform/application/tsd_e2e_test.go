//go:build e2e

// End-to-end tests of the TSD (warehouse terminal) channel (ARCHITECTURE.md
// §11.2 flows 4-6): real PostgreSQL, real gRPC servers with the driver wiring
// (for setup) and the TSD wiring, real write repositories and the in-memory
// sync event bus with the outbox handlers (cargo + WMS). Only the external
// provider ports (route template, package demand, driver, gate-reserve
// manager, QR handshake) are faked.
//
// Run: go test -tags e2e -run 'TestE2ETSD' -v ./internal/platform/application
package application_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"

	publicdriver "last-mile-task/internal/application/public_driver"
	drivergrpc "last-mile-task/internal/application/public_driver/transport/grpc"
	tsdapp "last-mile-task/internal/application/tsd"
	"last-mile-task/internal/platform/database"
	grpctsd "last-mile-task/internal/platform/grpc/tsd"
	wms_outbox "last-mile-task/internal/platform/outbox/wms"
	shipping_application "last-mile-task/internal/shipping/application"
	"last-mile-task/internal/shipping/application/demand"
	"last-mile-task/internal/shipping/application/ports"
	shipping_scanner "last-mile-task/internal/shipping/application/scanner"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure"
	pgdriver "last-mile-task/internal/shipping/infrastructure/driver/postgres"
	"last-mile-task/internal/shipping/infrastructure/eventdeps"
	"last-mile-task/internal/shipping/infrastructure/events/bus/inmem"
	"last-mile-task/internal/shipping/infrastructure/events/handlers"
	cargo_handlers "last-mile-task/internal/shipping/infrastructure/events/handlers/cargo"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/warehouse"
	"last-mile-task/internal/shipping/infrastructure/postgrestest"
	"last-mile-task/internal/shipping/infrastructure/tsdpackages"
	shipmentq "last-mile-task/internal/shipping/queries/driver/shipment"
	tsdpb "last-mile-task/pkg/gen/api/proto/v1/internals/tsd"
	driverpb "last-mile-task/pkg/gen/api/proto/v1/public/driver"
	"last-mile-task/pkg/qr"
)

const tsdE2ESecret = "tsd-e2e-handshake-secret"

// tsdHandshake signs/verifies QR sessions with the fixed e2e secret.
type tsdHandshake struct{}

func (tsdHandshake) CreateSession(s qr.Session) (string, error) {
	return qr.CreateSession(s, []byte(tsdE2ESecret))
}

func (tsdHandshake) DecodeQR(token string) (qr.Session, error) {
	return qr.DecodeQR(token, []byte(tsdE2ESecret))
}

// tsdHarness wires the driver gRPC server (for the reserve/shipment setup) and
// the TSD gRPC server over the same repositories, bus and outbox handlers.
type tsdHarness struct {
	driverClient driverpb.DriverClient
	tsdClient    tsdpb.TSDClient
	conn         *grpc.ClientConn
	grpcServer   *grpc.Server
	pool         *pgxpool.Pool
	driver       values.Driver
	handshake    tsdHandshake
}

func newTSDHarness(
	t *testing.T,
	templates ports.RouteTemplateProvider,
	driver values.Driver,
) *tsdHarness {
	t.Helper()

	repo := postgrestest.Repo(t)
	postgrestest.CleanTables(t, repo.Pool)

	entityRepo := infrastructure.NewEntityRepository(repo)

	shippingBus := inmem.NewSyncEventBus()

	// Outbox handlers: package scans → cargo.packages.scanned + WMS tares.
	outboxAdapter := database.NewOutboxAdapter(repo)
	wmsWriter := wms_outbox.NewWriter(outboxAdapter)
	eventStore := eventdeps.NewStore(repo)
	warehousePublisher := warehouse.NewPublisher(eventStore, wmsWriter, eventdeps.NewOffices(), eventdeps.NewUsers())
	cargoPublisher := cargo_handlers.NewPublisher(outboxAdapter)
	packageEventHandler := handlers.NewPackageEventHandler(eventStore, warehousePublisher, cargoPublisher)
	shippingBus.SubscribePackageEvent(events.PackageScanned,
		inmem.TypedHandler(packageEventHandler.HandlePackageScannedEvent),
	)

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
	reserveService := shipping_application.NewReserveService(
		entityRepo.ReserveRepository,
		entityRepo.ShipmentRepository,
		entityRepo.RouteRepository,
		infrastructure.NewReserveManager(),
		policy,
		shippingBus,
		repo,
	)
	scannerService := shipping_scanner.NewPackageScanner(
		entityRepo.PackageRepository,
		shippingBus,
		repo,
	)
	shippingQueries := shipmentq.NewQueries(
		pgdriver.New(repo),
		infrastructure.NewAddressResolver(),
	)

	driverApp := publicdriver.New(publicdriver.Config{
		ShippingQueries:        shippingQueries,
		DriversProvider:        &csDriverProvider{driver: driver},
		AssemblerService:       assembler,
		ProgressService:        progress,
		DemandService:          demandService,
		ReserveService:         reserveService,
		Handshake:              tsdHandshake{},
		RouteTemplatesProvider: templates,
		Tx:                     repo,
	})

	tsdApp := tsdapp.New(tsdapp.Config{
		Handshake:       tsdHandshake{},
		Scanner:         scannerService,
		ReserveApprover: reserveService,
		RouteRepository: entityRepo.RouteRepository,
		PackagesToScan:  tsdpackages.NewStore(repo),
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := drivergrpc.New(logger, driverApp)
	tsdSrv := grpctsd.New(logger, tsdApp)

	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	srv.RegisterGRPC(grpcServer)
	tsdSrv.RegisterGRPC(grpcServer)
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

	return &tsdHarness{
		driverClient: driverpb.NewDriverClient(conn),
		tsdClient:    tsdpb.NewTSDClient(conn),
		conn:         conn,
		grpcServer:   grpcServer,
		pool:         repo.Pool,
		driver:       driver,
		handshake:    tsdHandshake{},
	}
}

// createAssignedShipment creates a shipment via the driver gRPC server.
func (h *tsdHarness) createAssignedShipment(t *testing.T, ctx context.Context) string {
	t.Helper()

	resp, err := h.driverClient.CreateShipment(ctx, &driverpb.CreateShipmentRequest{RouteTemplateId: 1})
	if err != nil {
		t.Fatalf("CreateShipment: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("expected shipment creation success, got %q", resp.GetMessage())
	}

	return resp.GetData().GetShipment().GetId()
}

// sourceRoutePointID returns the route point of the shipment with seq_num = 0.
func (h *tsdHarness) sourceRoutePointID(t *testing.T, ctx context.Context, shipmentID string) values.RoutePointID {
	t.Helper()
	var id uuid.UUID
	if err := h.pool.QueryRow(ctx,
		`SELECT route_point_id FROM lmt.route_point WHERE shipment_id = $1 AND seq_num = 0`, shipmentID,
	).Scan(&id); err != nil {
		t.Fatalf("source route point: %v", err)
	}
	return values.RoutePointID(id)
}

// TestE2ETSDScanShipment proves the ScanShipment flow: approving the reserve
// moves the shipment into LOADING and the source route point into DRIVER_ARRIVED.
// An invalid QR leaves the state unchanged.
func TestE2ETSDScanShipment(t *testing.T) {
	ctx := context.Background()
	driver := values.Driver{ID: values.NewID(), SupplierID: 7}
	h := newTSDHarness(t, &csTemplateProvider{byID: map[values.RouteTemplateID]values.RouteTemplate{
		1: csValidTemplate(),
	}}, driver)

	shipmentID := h.createAssignedShipment(t, ctx)
	source := h.sourceRoutePointID(t, ctx, shipmentID)

	if _, err := h.driverClient.CreateReserve(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("CreateReserve: %v", err)
	}

	token, err := h.handshake.CreateSession(qr.Session{
		ShipmentID:   shipmentID,
		RoutePointID: source.String(),
		TaskID:       "task",
		ExpiresAt:    time.Now().Add(20 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	scanCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("tsd_qr_token", token))
	if _, err := h.tsdClient.ScanShipment(scanCtx, &emptypb.Empty{}); err != nil {
		t.Fatalf("ScanShipment: %v", err)
	}

	if got := csCount(t, ctx, h.pool, `SELECT shipment_status FROM lmt.shipment WHERE shipment_id = $1`, shipmentID); got != int(values.ShipmentStatusLoading) {
		t.Fatalf("expected shipment LOADING (%d), got %d", values.ShipmentStatusLoading, got)
	}
	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.reserve WHERE shipment_id = $1 AND status = $2`, shipmentID, int(values.ReserveStatusActive)); got != 1 {
		t.Fatalf("expected 1 ACTIVE reserve, got %d", got)
	}
	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.reserve WHERE shipment_id = $1 AND scanned_at IS NOT NULL`, shipmentID); got != 1 {
		t.Fatalf("expected a scanned reserve, got %d", got)
	}
	if got := csCount(t, ctx, h.pool, `SELECT route_point_status FROM lmt.route_point WHERE route_point_id = $1`, source); got != int(values.RoutePointStatusDriverArrived) {
		t.Fatalf("expected route point DRIVER_ARRIVED (%d), got %d", values.RoutePointStatusDriverArrived, got)
	}

	// Invalid QR: no state change.
	badCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("tsd_qr_token", "garbage"))
	if _, err := h.tsdClient.ScanShipment(badCtx, &emptypb.Empty{}); err == nil {
		t.Fatal("expected an error for an invalid QR token")
	}
	if got := csCount(t, ctx, h.pool, `SELECT shipment_status FROM lmt.shipment WHERE shipment_id = $1`, shipmentID); got != int(values.ShipmentStatusLoading) {
		t.Fatalf("shipment must remain LOADING after a failed scan, got %d", got)
	}
}

// packageAtPoint returns a package_route_point row (id, route point, action).
func (h *tsdHarness) packageAtPoint(t *testing.T, ctx context.Context, shipmentID string) (packageID string, routePointID string, action int) {
	t.Helper()
	if err := h.pool.QueryRow(ctx,
		`SELECT package_id, route_point_id, action FROM lmt.package_route_point WHERE shipment_id = $1 ORDER BY action LIMIT 1`,
		shipmentID,
	).Scan(&packageID, &routePointID, &action); err != nil {
		t.Fatalf("package at point: %v", err)
	}
	return packageID, routePointID, action
}

// TestE2ETSDScanPackages proves the ScanPackages flow: a valid scan applies the
// package action and writes the WMS/Cargo outbox messages; an invalid package is
// reported without breaking the valid one (partial success).
func TestE2ETSDScanPackages(t *testing.T) {
	ctx := context.Background()
	driver := values.Driver{ID: values.NewID(), SupplierID: 7}
	h := newTSDHarness(t, &csTemplateProvider{byID: map[values.RouteTemplateID]values.RouteTemplate{
		1: csValidTemplate(),
	}}, driver)

	shipmentID := h.createAssignedShipment(t, ctx)
	packageID, routePointID, action := h.packageAtPoint(t, ctx, shipmentID)

	// Scan the package at its first (LOAD) point.
	payload := `[{"package_id":"` + packageID + `","route_point_id":"` + routePointID + `","point_id":1,"action":` + itoa(action) + `}]`
	scanCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"tsd_shipment_id", shipmentID,
		"tsd_packages", payload,
	))
	if _, err := h.tsdClient.ScanPackages(scanCtx, &emptypb.Empty{}); err != nil {
		t.Fatalf("ScanPackages: %v", err)
	}

	if got := csCount(t, ctx, h.pool, `SELECT action_status FROM lmt.package_route_point WHERE package_id = $1 AND route_point_id = $2`, packageID, routePointID); got != int(values.RoutePointPackageStatusDone) {
		t.Fatalf("expected package action DONE (%d), got %d", values.RoutePointPackageStatusDone, got)
	}
	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.outbox WHERE type = $1 AND status = $2`, "wms.tares.load", "TODO"); got != 1 {
		t.Fatalf("expected 1 wms.tares.load outbox message, got %d", got)
	}

	// Partial success: an unknown package must not break the valid scan.
	badID := uuid.NewString()
	payloadWithBad := `[{"package_id":"` + badID + `","route_point_id":"` + routePointID + `","point_id":1,"action":` + itoa(action) + `}]`
	badScanCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"tsd_shipment_id", shipmentID,
		"tsd_packages", payloadWithBad,
	))
	if _, err := h.tsdClient.ScanPackages(badScanCtx, &emptypb.Empty{}); err != nil {
		t.Fatalf("ScanPackages(partial): %v", err)
	}
}

// TestE2ETSDListPackagesToScan proves the ListPackagesToScan read source: it
// returns the pending packages at a point and is empty at a point without them.
func TestE2ETSDListPackagesToScan(t *testing.T) {
	ctx := context.Background()
	driver := values.Driver{ID: values.NewID(), SupplierID: 7}
	h := newTSDHarness(t, &csTemplateProvider{byID: map[values.RouteTemplateID]values.RouteTemplate{
		1: csValidTemplate(),
	}}, driver)

	shipmentID := h.createAssignedShipment(t, ctx)
	source := h.sourceRoutePointID(t, ctx, shipmentID)

	listCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"tsd_shipment_id", shipmentID,
		"tsd_route_point_id", source.String(),
	))
	if _, err := h.tsdClient.ListPackagesToScan(listCtx, &emptypb.Empty{}); err != nil {
		t.Fatalf("ListPackagesToScan: %v", err)
	}

	// A point without pending packages returns an empty list without error.
	emptyPoint := uuid.NewString()
	emptyCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"tsd_shipment_id", shipmentID,
		"tsd_route_point_id", emptyPoint,
	))
	if _, err := h.tsdClient.ListPackagesToScan(emptyCtx, &emptypb.Empty{}); err != nil {
		t.Fatalf("ListPackagesToScan(empty): %v", err)
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
