//go:build e2e

// End-to-end tests of the public driver end-shipment vertical (ARCHITECTURE.md
// §23 flow 8): real PostgreSQL (migrations + clean tables), real gRPC server
// with the driver wiring, real write repositories and the in-memory sync event
// bus with the cargo-shipment + payments outbox handlers.
//
// Run: go test -tags e2e -run 'TestE2EEndShipment' -v ./internal/platform/application
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
	"google.golang.org/protobuf/types/known/emptypb"

	publicdriver "last-mile-task/internal/application/public_driver"
	drivergrpc "last-mile-task/internal/application/public_driver/transport/grpc"
	"last-mile-task/internal/platform/database"
	payments_outbox "last-mile-task/internal/platform/outbox/payments"
	shipping_application "last-mile-task/internal/shipping/application"
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

// esHarness wires the end-shipment slice over real PostgreSQL.
type esHarness struct {
	client     driverpb.DriverClient
	conn       *grpc.ClientConn
	grpcServer *grpc.Server
	repo       *database.Repository
	pool       *pgxpool.Pool
	driver     values.Driver
}

func newESHarness(t *testing.T, driver values.Driver) *esHarness {
	t.Helper()

	repo := postgrestest.Repo(t)
	postgrestest.CleanTables(t, repo.Pool)

	entityRepo := infrastructure.NewEntityRepository(repo)
	outboxAdapter := database.NewOutboxAdapter(repo)
	paymentsWriter := payments_outbox.NewWriter(outboxAdapter)

	shippingBus := inmem.NewSyncEventBus()

	// The ShipmentCompleted event drives the cargo-shipment FINISH trigger and
	// the payments.last_mile_shipment_accrual outbox message (sync handlers).
	shipmentEventHandler := handlers.NewShipmentEventHandler(infrastructure.NewCargoShipment(), paymentsWriter)
	shippingBus.SubscribeShipmentEvent(events.ShipmentCompleted,
		inmem.TypedHandler(shipmentEventHandler.HandleShipmentCompletedEvent))

	policy := policies.NewShipmentPolicy(
		infrastructure.NewShipmentPolicyRepository(
			entityRepo.ShipmentRepository,
			entityRepo.RouteRepository,
			entityRepo.PackageRepository,
			entityRepo.ReserveRepository,
			infrastructure.NewDemandProvider(),
		),
	)
	progress := shipping_application.NewShipmentProgressService(
		policy,
		entityRepo.ShipmentRepository,
		shippingBus,
		repo,
	)
	routePointService := shipping_application.NewRoutePointService(
		entityRepo.RouteRepository,
		entityRepo.PackageRepository,
		shippingBus,
		progress,
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
		nil,
		progress,
		routePointService,
		nil,
		nil,
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

	return &esHarness{
		client:     driverpb.NewDriverClient(conn),
		conn:       conn,
		grpcServer: grpcServer,
		repo:       repo,
		pool:       repo.Pool,
		driver:     driver,
	}
}

// seedCompletedRoute inserts a shipment whose last route point is
// DRIVER_ARRIVED and whose packages have an unload action with the given
// action_status, so the departure test can drive the flow in both
// ready (DONE) and not-ready (TODO) directions.
func seedCompletedRoute(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	driver values.Driver,
	shipmentStatus int,
	unloadActionStatus int,
) (shipmentID, sourceID, targetID, routeID, pkgID uuid.UUID) {
	t.Helper()

	shipmentID = uuid.New()
	routeID = uuid.New()
	sourceID = uuid.New()
	targetID = uuid.New()
	pkgID = uuid.New()

	seed := []string{
		`INSERT INTO lmt.shipment (shipment_id, shipment_short_id, shipment_status, driver_id, car_license_plate)
		 VALUES ('` + shipmentID.String() + `', 910001, ` + esItoa(shipmentStatus) + `, '` + driver.ID.String() + `', 'E777AA77')`,
		`INSERT INTO lmt.route (route_id, shipment_id, routecar_id)
		 VALUES ('` + routeID.String() + `', '` + shipmentID.String() + `', 1)`,
		`INSERT INTO lmt.route_point (route_point_id, route_id, shipment_id, routecar_point_id, seq_num, route_point_status)
		 VALUES ('` + sourceID.String() + `', '` + routeID.String() + `', '` + shipmentID.String() + `', 1001, 1, 4)`,
		`INSERT INTO lmt.route_point (route_point_id, route_id, shipment_id, routecar_point_id, seq_num, route_point_status)
		 VALUES ('` + targetID.String() + `', '` + routeID.String() + `', '` + shipmentID.String() + `', 1002, 2, 1)`,
		`INSERT INTO lmt.package (package_id, shipment_id, package_status, tare_id, tare_type)
		 VALUES ('` + pkgID.String() + `', '` + shipmentID.String() + `', 5, 42, 'BOX')`,
		`INSERT INTO lmt.package_route_point (package_id, route_point_id, shipment_id, action, action_status)
		 VALUES ('` + pkgID.String() + `', '` + sourceID.String() + `', '` + shipmentID.String() + `', 1, 2)`,
		`INSERT INTO lmt.package_route_point (package_id, route_point_id, shipment_id, action, action_status)
		 VALUES ('` + pkgID.String() + `', '` + targetID.String() + `', '` + shipmentID.String() + `', 2, ` + esItoa(unloadActionStatus) + `)`,
	}
	for _, stmt := range seed {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("seed: %v\n%s", err, stmt)
		}
	}

	return shipmentID, sourceID, targetID, routeID, pkgID
}

func esItoa(n int) string {
	const digits = "0123456789"
	if n < 0 {
		return "-" + esItoa(-n)
	}
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = digits[n%10]
		n /= 10
	}
	return string(buf[i:])
}

// TestE2EEndShipment proves flow 8: departing the last route point completes
// the point and the shipment, triggers cargo-shipment FINISH and writes the
// payments.last_mile_shipment_accrual outbox message in the same transaction.
func TestE2EEndShipment(t *testing.T) {
	ctx := context.Background()
	driver := values.Driver{ID: values.NewID(), SupplierID: 7}
	h := newESHarness(t, driver)

	// UNLOADING status (7), unload DONE (2): ready to complete.
	shipmentID, _, _, _, _ := seedCompletedRoute(t, ctx, h.pool, driver, 7, 2)

	if _, err := h.client.UpdateRoutePointDriverDeparted(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("UpdateRoutePointDriverDeparted: %v", err)
	}

	assertInt(t, ctx, h.pool,
		`SELECT shipment_status FROM lmt.shipment WHERE shipment_id = $1`, shipmentID, int(values.ShipmentStatusCompleted))

	// Both route points (source seeded COMPLETED, target completed on departure)
	// must be COMPLETED.
	assertInt(t, ctx, h.pool,
		`SELECT count(*) FROM lmt.route_point WHERE shipment_id = $1 AND route_point_status = 4`, shipmentID, 2)

	// The payments accrual outbox message must be written (same transaction).
	assertOutboxTODO(t, ctx, h.pool, payments_outbox.MessageTypeLastMileShipmentAccrual, 1)
}

// TestE2EEndShipmentNotReady proves a shipment that cannot be completed maps
// to FailedPrecondition and leaves no state behind.
func TestE2EEndShipmentNotReady(t *testing.T) {
	ctx := context.Background()
	driver := values.Driver{ID: values.NewID(), SupplierID: 7}
	h := newESHarness(t, driver)

	// NEW status (1): not completable by the policy.
	shipmentID, _, _, _, _ := seedCompletedRoute(t, ctx, h.pool, driver, 1, 2)

	_, err := h.client.EndShipment(ctx, &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", status.Code(err))
	}
	assertInt(t, ctx, h.pool,
		`SELECT shipment_status FROM lmt.shipment WHERE shipment_id = $1`, shipmentID, int(values.ShipmentStatusNew))
	assertOutboxTODO(t, ctx, h.pool, payments_outbox.MessageTypeLastMileShipmentAccrual, 0)
}

func assertInt(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, arg any, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(ctx, query, arg).Scan(&got); err != nil {
		t.Fatalf("assertInt %q: %v", query, err)
	}
	if got != want {
		t.Fatalf("assertInt %q: want %d, got %d", query, want, got)
	}
}

func assertOutboxTODO(t *testing.T, ctx context.Context, pool *pgxpool.Pool, msgType string, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM lmt.outbox WHERE type = $1 AND status = 'TODO'`, msgType,
	).Scan(&got); err != nil {
		t.Fatalf("outbox count: %v", err)
	}
	if got != want {
		t.Fatalf("outbox %s TODO: want %d, got %d", msgType, want, got)
	}
}