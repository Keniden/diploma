//go:build e2e

// End-to-end tests of the public driver gate-reserve vertical (ARCHITECTURE.md
// §23 flows 4 and 7): real PostgreSQL, real gRPC server with the driver
// wiring, real write repositories and the in-memory sync event bus. Only the
// external provider ports (route template, package demand, driver, gate-reserve
// manager, QR handshake) are faked — the same loose ports the production stubs
// stand in for (TODO LASTMILE-099/098).
//
// Run: go test -tags e2e -run TestE2EReserve -v ./internal/platform/application
package application_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"

	publicdriver "last-mile-task/internal/application/public_driver"
	drivergrpc "last-mile-task/internal/application/public_driver/transport/grpc"
	shipping_application "last-mile-task/internal/shipping/application"
	"last-mile-task/internal/shipping/application/demand"
	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure"
	pgdriver "last-mile-task/internal/shipping/infrastructure/driver/postgres"
	"last-mile-task/internal/shipping/infrastructure/events/bus/inmem"
	"last-mile-task/internal/shipping/infrastructure/postgrestest"
	shipmentq "last-mile-task/internal/shipping/queries/driver/shipment"
	driverpb "last-mile-task/pkg/gen/api/proto/v1/public/driver"
	"last-mile-task/pkg/qr"
)

// reserveHandshake issues QR sessions via pkg/qr with a fixed test secret.
type reserveHandshake struct{}

func (reserveHandshake) CreateSession(s qr.Session) (string, error) {
	return qr.CreateSession(s, []byte("e2e-handshake-secret"))
}

func (reserveHandshake) DecodeQR(token string) (qr.Session, error) {
	return qr.DecodeQR(token, []byte("e2e-handshake-secret"))
}

type reserveHarness struct {
	client     driverpb.DriverClient
	conn       *grpc.ClientConn
	grpcServer *grpc.Server
	pool       *pgxpool.Pool
	driver     values.Driver
	reserveSvc *shipping_application.ReserveService
}

func newReserveHarness(
	t *testing.T,
	driver values.Driver,
	templates ports.RouteTemplateProvider,
) *reserveHarness {
	t.Helper()

	repo := postgrestest.Repo(t)
	postgrestest.CleanTables(t, repo.Pool)

	entityRepo := infrastructure.NewEntityRepository(repo)
	shippingBus := inmem.NewSyncEventBus()

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
	shippingQueries := shipmentq.NewQueries(
		pgdriver.New(repo),
		infrastructure.NewAddressResolver(),
	)

	app := publicdriver.New(publicdriver.Config{
		ShippingQueries:    shippingQueries,
		DriversProvider:    &csDriverProvider{driver: driver},
		AssemblerService:   assembler,
		ProgressService:    progress,
		DemandService:      demandService,
		ReserveService:     reserveService,
		Handshake:          reserveHandshake{},
		RouteTemplatesProvider: templates,
		Tx:                 repo,
	})

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

	return &reserveHarness{
		client:     driverpb.NewDriverClient(conn),
		conn:       conn,
		grpcServer: grpcServer,
		pool:       repo.Pool,
		driver:     driver,
		reserveSvc: reserveService,
	}
}

func (h *reserveHarness) createAssignedShipment(t *testing.T, ctx context.Context) string {
	t.Helper()

	resp, err := h.client.CreateShipment(ctx, &driverpb.CreateShipmentRequest{RouteTemplateId: 1})
	if err != nil {
		t.Fatalf("CreateShipment: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("expected shipment creation success, got %q", resp.GetMessage())
	}

	return resp.GetData().GetShipment().GetId()
}

// TestE2EReserveFlow proves the reserve vertical: CreateReserve persists a
// lmt.reserve row and moves the shipment into RESERVE_LOADING; a second
// reserve conflicts (FailedPrecondition); UpdateReserveQRStatus confirms the
// reserve and moves the shipment into LOADING.
func TestE2EReserveFlow(t *testing.T) {
	ctx := context.Background()
	driver := values.Driver{ID: values.NewID(), SupplierID: 7}
	h := newReserveHarness(t, driver, &csTemplateProvider{byID: map[values.RouteTemplateID]values.RouteTemplate{
		1: csValidTemplate(),
	}})

	shipmentID := h.createAssignedShipment(t, ctx)

	if _, err := h.client.CreateReserve(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("CreateReserve: %v", err)
	}

	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.reserve WHERE shipment_id = $1`, shipmentID); got != 1 {
		t.Fatalf("expected 1 reserve row, got %d", got)
	}
	if got := csCount(t, ctx, h.pool, `SELECT shipment_status FROM lmt.shipment WHERE shipment_id = $1`, shipmentID); got != int(values.ShipmentStatusReserveLoading) {
		t.Fatalf("expected RESERVE_LOADING (%d), got %d", values.ShipmentStatusReserveLoading, got)
	}

	// Conflict: active reserve already exists.
	_, err := h.client.CreateReserve(ctx, &emptypb.Empty{})
	if err == nil || status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition on conflict, got %v", err)
	}
	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.reserve WHERE shipment_id = $1`, shipmentID); got != 1 {
		t.Fatalf("expected 1 reserve row after conflict, got %d", got)
	}

	// Not scanned yet.
	if _, err := h.client.GetReserveQRStatus(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("GetReserveQRStatus: %v", err)
	}
	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.reserve WHERE shipment_id = $1 AND scanned_at IS NULL`, shipmentID); got != 1 {
		t.Fatalf("expected an unscanned reserve, got %d", got)
	}

	// Confirm the reserve via UpdateReserveQRStatus.
	if _, err := h.client.UpdateReserveQRStatus(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("UpdateReserveQRStatus: %v", err)
	}
	if got := csCount(t, ctx, h.pool, `SELECT shipment_status FROM lmt.shipment WHERE shipment_id = $1`, shipmentID); got != int(values.ShipmentStatusLoading) {
		t.Fatalf("expected LOADING (%d), got %d", values.ShipmentStatusLoading, got)
	}
	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.reserve WHERE shipment_id = $1 AND status = $2`, shipmentID, int(values.ReserveStatusActive)); got != 1 {
		t.Fatalf("expected 1 ACTIVE reserve, got %d", got)
	}

	// Confirm state is actually scanned now.
	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.reserve WHERE shipment_id = $1 AND scanned_at IS NOT NULL`, shipmentID); got != 1 {
		t.Fatalf("expected a scanned reserve after confirmation, got %d", got)
	}
}

// TestE2EReserveCancel proves CancelReserve cancels a not-confirmed reserve and
// returns the shipment back to ASSIGNED.
func TestE2EReserveCancel(t *testing.T) {
	ctx := context.Background()
	driver := values.Driver{ID: values.NewID(), SupplierID: 7}
	h := newReserveHarness(t, driver, &csTemplateProvider{byID: map[values.RouteTemplateID]values.RouteTemplate{
		1: csValidTemplate(),
	}})

	shipmentID := h.createAssignedShipment(t, ctx)

	if _, err := h.client.CreateReserve(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("CreateReserve: %v", err)
	}
	if got := csCount(t, ctx, h.pool, `SELECT shipment_status FROM lmt.shipment WHERE shipment_id = $1`, shipmentID); got != int(values.ShipmentStatusReserveLoading) {
		t.Fatalf("expected RESERVE_LOADING, got %d", got)
	}

	if _, err := h.client.CancelReserve(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("CancelReserve: %v", err)
	}

	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.reserve WHERE shipment_id = $1 AND status = $2`, shipmentID, int(values.ReserveStatusCanceled)); got != 1 {
		t.Fatalf("expected 1 CANCELED reserve, got %d", got)
	}
	if got := csCount(t, ctx, h.pool, `SELECT shipment_status FROM lmt.shipment WHERE shipment_id = $1`, shipmentID); got != int(values.ShipmentStatusAssigned) {
		t.Fatalf("expected shipment back to ASSIGNED (%d), got %d", values.ShipmentStatusAssigned, got)
	}
}

// TestE2EReserveExpiry proves the expiry pass: an expired waiting-approval
// reserve becomes EXPIRED_WAITING_APPROVAL and an already-expired reserve is
// left untouched on a second run (idempotency).
func TestE2EReserveExpiry(t *testing.T) {
	ctx := context.Background()
	driver := values.Driver{ID: values.NewID(), SupplierID: 7}
	h := newReserveHarness(t, driver, &csTemplateProvider{byID: map[values.RouteTemplateID]values.RouteTemplate{
		1: csValidTemplate(),
	}})

	shipmentID := h.createAssignedShipment(t, ctx)

	if _, err := h.client.CreateReserve(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("CreateReserve: %v", err)
	}

	// Force the reserve into the past to trigger expiry.
	if _, err := h.pool.Exec(ctx, `UPDATE lmt.reserve
		SET expires_at = NOW() - INTERVAL '1 hour'
		WHERE shipment_id = $1`, shipmentID); err != nil {
		t.Fatalf("set expires_at into the past: %v", err)
	}

	if err := h.reserveSvc.ExpireDueWaitingApprovalReserves(ctx); err != nil {
		t.Fatalf("expire due waiting approval: %v", err)
	}

	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.reserve WHERE shipment_id = $1 AND status = $2`, shipmentID, int(values.ReserveStatusExpiredWaitingApproval)); got != 1 {
		t.Fatalf("expected 1 EXPIRED_WAITING_APPROVAL reserve, got %d", got)
	}

	// A second run must not change already-expired reserves (idempotency).
	if err := h.reserveSvc.ExpireDueWaitingApprovalReserves(ctx); err != nil {
		t.Fatalf("second expire run: %v", err)
	}
	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.reserve WHERE shipment_id = $1`, shipmentID); got != 1 {
		t.Fatalf("expected 1 reserve row after idempotent rerun, got %d", got)
	}
	if got := csCount(t, ctx, h.pool, `SELECT count(*) FROM lmt.reserve WHERE shipment_id = $1 AND status = $2`, shipmentID, int(values.ReserveStatusExpiredWaitingApproval)); got != 1 {
		t.Fatalf("expected the reserve to remain EXPIRED_WAITING_APPROVAL after rerun, got %d", got)
	}
}