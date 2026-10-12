//go:build e2e

// Package application_test contains the end-to-end slice test of the public
// driver vertical: real PostgreSQL (migrations + seeds), real gRPC server
// with driver wiring, and the WMS exporter worker in mock mode.
//
// Run: go test -tags e2e -run TestE2ESlice -v ./internal/platform/application
//
// TODO(LASTMILE-108): the real repository keeps e2e tests in the root e2e/
// directory, which is not part of ARCHITECTURE.md §4; the skeleton keeps the
// slice e2e here to stay within the documented tree.
package application_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"last-mile-task/internal/platform/application"
	"last-mile-task/internal/platform/application/builder"
	cargo_outbox "last-mile-task/internal/platform/outbox/cargo"
	wms_outbox "last-mile-task/internal/platform/outbox/wms"
	driverpb "last-mile-task/pkg/gen/api/proto/v1/public/driver"
	pkgpkg "last-mile-task/pkg/gen/api/proto/v1/public/driver/package"
	routepointv1 "last-mile-task/pkg/gen/api/proto/v1/public/driver/route_point"
)

const (
	e2eDBURL     = "postgres://postgres:postgres@localhost:5432/last_mile_e2e?sslmode=disable"
	e2eGRPCAddr  = "127.0.0.1:45111"
	e2eLifecycle = "127.0.0.1:45112"
)

func TestE2ESlice(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	dsn := envOr("E2E_DATABASE_URL", e2eDBURL)
	migrationsDir := envOr("E2E_MIGRATIONS_DIR", filepath.Join("..", "..", "..", "migrations"))
	if _, err := os.Stat(migrationsDir); err != nil {
		t.Fatalf("migrations dir: %v", err)
	}

	// --- Fresh schema + migrations -------------------------------------------
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open admin db: %v", err)
	}
	t.Cleanup(func() { _ = admin.Close() })

	freshMigrate(t, ctx, admin, migrationsDir)

	// --- Seeds ----------------------------------------------------------------
	driverID := uuid.New()
	shipmentID := uuid.New()
	routeID := uuid.New()
	routePointSource := uuid.New()
	routePointDest := uuid.New()
	pkgA := uuid.New()
	pkgB := uuid.New()

	seed := []string{
		// shipment_status = 4 (LOADING), package_status = 1 (READY_TO_LOAD),
		// route_point_status = 0 (WAITING_ARRIVAL), package_route_point
		// action = 1 (LOAD) / 2 (UNLOAD), action_status = 1 (TODO).
		`INSERT INTO lmt.shipment (shipment_id, shipment_short_id, shipment_status, driver_id, car_license_plate)
		 VALUES ('` + shipmentID.String() + `', 900001, 4, '` + driverID.String() + `', 'E777AA77')`,
		`INSERT INTO lmt.route (route_id, shipment_id, routecar_id)
		 VALUES ('` + routeID.String() + `', '` + shipmentID.String() + `', 1)`,
		`INSERT INTO lmt.route_point (route_point_id, route_id, shipment_id, routecar_point_id, seq_num, route_point_status)
		 VALUES ('` + routePointSource.String() + `', '` + routeID.String() + `', '` + shipmentID.String() + `', 1001, 1, 0)`,
		`INSERT INTO lmt.route_point (route_point_id, route_id, shipment_id, routecar_point_id, seq_num, route_point_status)
		 VALUES ('` + routePointDest.String() + `', '` + routeID.String() + `', '` + shipmentID.String() + `', 1002, 2, 0)`,
		`INSERT INTO lmt.package (package_id, shipment_id, package_status, tare_id, tare_type)
		 VALUES ('` + pkgA.String() + `', '` + shipmentID.String() + `', 1, 42, 'BOX')`,
		`INSERT INTO lmt.package (package_id, shipment_id, package_status, tare_id, tare_type)
		 VALUES ('` + pkgB.String() + `', '` + shipmentID.String() + `', 1, 43, 'BOX')`,
		`INSERT INTO lmt.package_route_point (package_id, route_point_id, shipment_id, action, action_status)
		 VALUES ('` + pkgA.String() + `', '` + routePointSource.String() + `', '` + shipmentID.String() + `', 1, 1)`,
		`INSERT INTO lmt.package_route_point (package_id, route_point_id, shipment_id, action, action_status)
		 VALUES ('` + pkgA.String() + `', '` + routePointDest.String() + `', '` + shipmentID.String() + `', 2, 1)`,
		`INSERT INTO lmt.package_route_point (package_id, route_point_id, shipment_id, action, action_status)
		 VALUES ('` + pkgB.String() + `', '` + routePointSource.String() + `', '` + shipmentID.String() + `', 1, 1)`,
		`INSERT INTO lmt.package_route_point (package_id, route_point_id, shipment_id, action, action_status)
		 VALUES ('` + pkgB.String() + `', '` + routePointDest.String() + `', '` + shipmentID.String() + `', 2, 1)`,
	}
	for _, stmt := range seed {
		if _, err := admin.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("seed: %v\n%s", err, stmt)
		}
	}

	// --- Boot the service -----------------------------------------------------
	t.Setenv("POSTGRES_DSN", dsn)
	t.Setenv("GRPC_ADDR", e2eGRPCAddr)
	t.Setenv("LIFECYCLE_ADDR", e2eLifecycle)
	t.Setenv("OUTBOX_POLL_INTERVAL", "100ms")
	t.Setenv("OUTBOX_BATCH_SIZE", "10")
	t.Setenv("MOCK_ENABLED", "true")
	t.Setenv("LOG_LEVEL", "info")

	b, err := builder.New(ctx, "e2e")
	if err != nil {
		t.Fatalf("builder: %v", err)
	}
	app, err := application.New(b)
	if err != nil {
		t.Fatalf("application: %v", err)
	}

	appCtx, appCancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- app.Run(appCtx) }()
	t.Cleanup(func() {
		appCancel()
		select {
		case err := <-runErr:
			if err != nil {
				t.Logf("application Run returned: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Log("application did not stop in time")
		}
	})

	waitReadyz(t, ctx, e2eLifecycle)

	// --- gRPC client -----------------------------------------------------------
	conn, err := grpc.NewClient(e2eGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	client := driverpb.NewDriverClient(conn)

	// Test 1: load scan at the source point → success, state updated, outbox
	// messages (cargo + WMS) delivered by the worker.
	resp := scan(t, ctx, client, driverID, shipmentID, []scanItem{{pkgA, routePointSource}})
	if got := resp.GetData().GetFailedPackages(); len(got) != 0 {
		t.Fatalf("expected no failed packages, got %+v", got)
	}

	assertDBInt(t, ctx, admin,
		`SELECT package_status FROM lmt.package WHERE package_id = '`+pkgA.String()+`'`, 2 /* LOADED */)
	assertDBInt(t, ctx, admin,
		`SELECT action_status FROM lmt.package_route_point
		  WHERE package_id = '`+pkgA.String()+`' AND route_point_id = '`+routePointSource.String()+`'`, 2 /* DONE */)
	// The WMS message is consumed by the worker (COMPLETED). Cargo messages
	// are only emitted for UNLOAD scans (cargo publisher filters by action),
	// so a LOAD scan must not produce any.
	assertOutboxDelivered(t, ctx, admin, wms_outbox.MessageTypeTaresLoad, 1)
	assertOutboxCount(t, ctx, admin, cargo_outbox.MessageTypeScannedPackages, "TODO", 0)

	// Test 2: the driver read model sees the shipment with its route points.
	getActiveShipment(t, ctx, client, driverID, shipmentID, pkgA)

	// Test 3: repeated scan of the same package → partial success, the package
	// lands in failed_packages, nothing new is written to outbox.
	resp2 := scan(t, ctx, client, driverID, shipmentID, []scanItem{{pkgA, routePointSource}})
	failed := resp2.GetData().GetFailedPackages()
	if len(failed) != 1 || failed[0].GetId() != pkgA.String() {
		t.Fatalf("expected pkgA failed once, got %+v", failed)
	}
	assertOutboxCount(t, ctx, admin, cargo_outbox.MessageTypeScannedPackages, "TODO", 0)
	assertOutboxCount(t, ctx, admin, wms_outbox.MessageTypeTaresLoad, "COMPLETED", 1)

	// Test 4: untouched package is still pending.
	assertDBInt(t, ctx, admin,
		`SELECT package_status FROM lmt.package WHERE package_id = '`+pkgB.String()+`'`, 1 /* READY_TO_LOAD */)
}

// --- helpers ------------------------------------------------------------------

type scanItem struct {
	pkgID   uuid.UUID
	pointID uuid.UUID
}

func scan(
	t *testing.T,
	ctx context.Context,
	client driverpb.DriverClient,
	driverID, shipmentID uuid.UUID,
	items []scanItem,
) *driverpb.SetPackagesStatusResponse {
	t.Helper()

	pkgs := make([]*driverpb.SetPackagesStatusRequest_PackageAtPoint, 0, len(items))
	for _, it := range items {
		pkgs = append(pkgs, &driverpb.SetPackagesStatusRequest_PackageAtPoint{
			Id:          it.pkgID.String(),
			PointId:     it.pointID.String(),
			Action:      pkgpkg.PointAction_POINT_ACTION_LOAD,
			Status:      pkgpkg.PointActionStatus_POINT_ACTION_STATUS_DONE,
			TareSticker: strPtr("STK1"),
		})
	}

	reqCtx := metadata.AppendToOutgoingContext(ctx, "x-driver-id", driverID.String())
	resp, err := client.SetPackagesStatus(reqCtx, &driverpb.SetPackagesStatusRequest{
		ShipmentId: shipmentID.String(),
		Packages:   pkgs,
	})
	if err != nil {
		t.Fatalf("SetPackagesStatus: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("expected success, got %v", resp.GetMessage())
	}
	return resp
}

func getActiveShipment(
	t *testing.T,
	ctx context.Context,
	client driverpb.DriverClient,
	driverID, shipmentID, pkgID uuid.UUID,
) {
	t.Helper()

	reqCtx := metadata.AppendToOutgoingContext(ctx, "x-driver-id", driverID.String())
	resp, err := client.GetActiveShipmentByDriverID(reqCtx, &driverpb.GetActiveShipmentByDriverIDRequest{})
	if err != nil {
		t.Fatalf("GetActiveShipmentByDriverID: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("expected success, got %v", resp.GetMessage())
	}
	ship := resp.GetData().GetShipment()
	if ship == nil {
		t.Fatalf("expected shipment data")
	}
	if ship.GetId() != shipmentID.String() {
		t.Fatalf("shipment id: got %s want %s", ship.GetId(), shipmentID.String())
	}
	if got := ship.GetRoute().GetPoints(); len(got) < 2 {
		t.Fatalf("expected ≥2 route points, got %d", len(got))
	}

	// The seeded route points are in WAITING_ARRIVAL state.
	for _, p := range ship.GetRoute().GetPoints() {
		if p.GetStatus() == routepointv1.RoutePointStatus_ROUTE_POINT_STATUS_WAITING_ARRIVAL {
			return
		}
	}
	t.Fatalf("expected at least one WAITING_ARRIVAL route point, got %+v", ship.GetRoute().GetPoints())
}

func assertOutboxDelivered(t *testing.T, ctx context.Context, db *sql.DB, messageType string, want int) {
	t.Helper()
	assertOutboxCount(t, ctx, db, messageType, "COMPLETED", want)
}

func assertOutboxCount(t *testing.T, ctx context.Context, db *sql.DB, messageType, status string, want int) {
	t.Helper()

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		got := countOutbox(t, ctx, db, messageType, status)
		if got == want {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("outbox %s/%s: expected %d rows, got %d", messageType, status, want, countOutbox(t, ctx, db, messageType, status))
}

func countOutbox(t *testing.T, ctx context.Context, db *sql.DB, messageType, status string) int {
	t.Helper()
	var got int
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM lmt.outbox WHERE type = $1 AND status = $2`, messageType, status,
	).Scan(&got); err != nil {
		t.Fatalf("outbox count: %v", err)
	}
	return got
}

func assertDBInt(t *testing.T, ctx context.Context, db *sql.DB, query string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRowContext(ctx, query).Scan(&got); err != nil {
		t.Fatalf("assertDBInt %q: %v", query, err)
	}
	if got != want {
		t.Fatalf("assertDBInt %q: got %d want %d", query, got, want)
	}
}

func waitReadyz(t *testing.T, ctx context.Context, addr string) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			t.Fatalf("context expired while waiting for readyz: %v", ctx.Err())
		default:
		}

		resp, err := http.Get("http://" + addr + "/readyz")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	t.Fatalf("service did not become ready at %s", addr)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func strPtr(s string) *string { return &s }

var _ = stdlib.GetDefaultDriver // keep the pgx stdlib driver linked for goose

// TestE2EWMSRetry proves the at-least-once delivery of the WMS exporter
// worker: the endpoint fails twice (HTTP 500), the outbox message is
// rescheduled, and only the third attempt completes the message.
func TestE2EWMSRetry(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	dsn := envOr("E2E_DATABASE_URL_RETRY", "postgres://postgres:postgres@localhost:5432/last_mile_e2e_retry?sslmode=disable")
	migrationsDir := envOr("E2E_MIGRATIONS_DIR", filepath.Join("..", "..", "..", "migrations"))

	ensureDB(t, ctx, dsn, "last_mile_e2e_retry")

	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open admin db: %v", err)
	}
	t.Cleanup(func() { _ = admin.Close() })

	freshMigrate(t, ctx, admin, migrationsDir)

	driverID := uuid.New()
	shipmentID := uuid.New()
	routeID := uuid.New()
	routePointSource := uuid.New()
	pkgA := uuid.New()
	seedSlice(t, ctx, admin, sliceSeed{driverID, shipmentID, routeID, routePointSource, pkgA, true})

	// Controllable WMS endpoint: fail the first two requests, then succeed.
	var attempts atomic.Int32
	var lastPath atomic.Value
	wmsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		lastPath.Store(r.URL.Path)
		if r.URL.Path != "/load" {
			http.NotFound(w, r)
			return
		}
		if n <= 2 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(wmsServer.Close)

	t.Setenv("POSTGRES_DSN", dsn)
	t.Setenv("GRPC_ADDR", "127.0.0.1:45113")
	t.Setenv("LIFECYCLE_ADDR", "127.0.0.1:45114")
	t.Setenv("OUTBOX_POLL_INTERVAL", "100ms")
	t.Setenv("OUTBOX_BATCH_SIZE", "10")
	t.Setenv("MOCK_ENABLED", "false")
	t.Setenv("WMS_BASE_URL", wmsServer.URL)
	t.Setenv("LOG_LEVEL", "info")

	b, err := builder.New(ctx, "e2e-retry")
	if err != nil {
		t.Fatalf("builder: %v", err)
	}
	t.Logf("retry config: wms_base_url=%q mock=%v", b.Config().WMS.BaseURL, b.Config().Mock.Enabled)
	app, err := application.New(b)
	if err != nil {
		t.Fatalf("application: %v", err)
	}

	appCtx, appCancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- app.Run(appCtx) }()
	t.Cleanup(func() {
		appCancel()
		select {
		case <-runErr:
		case <-time.After(10 * time.Second):
			t.Log("application did not stop in time")
		}
		// Stop the readers/servers: without Shutdown the outbox readers keep
		// polling and leak into the next test (same lmt.outbox table).
		if err := app.Shutdown(); err != nil {
			t.Logf("application shutdown: %v", err)
		}
	})

	waitReadyz(t, ctx, "127.0.0.1:45114")

	conn, err := grpc.NewClient("127.0.0.1:45113", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	resp := scan(t, ctx, driverpb.NewDriverClient(conn), driverID, shipmentID,
		[]scanItem{{pkgA, routePointSource}})
	if got := resp.GetData().GetFailedPackages(); len(got) != 0 {
		t.Fatalf("expected no failed packages, got %+v", got)
	}

	// The message is delivered at the third attempt and completed. The reader
	// uses the default 10s poll interval (same as the real repo), so the
	// retries land at ~0s / ~10s / ~20s.
	deadline := time.Now().Add(40 * time.Second)
	for time.Now().Before(deadline) {
		if attempts.Load() >= 3 && countOutbox(t, ctx, admin, wms_outbox.MessageTypeTaresLoad, "COMPLETED") == 1 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("wms attempts=%d path=%v completed=%d", attempts.Load(), lastPath.Load(),
		countOutbox(t, ctx, admin, wms_outbox.MessageTypeTaresLoad, "COMPLETED"))
}

// freshMigrate resets the lmt schema and applies all migrations.
func freshMigrate(t *testing.T, ctx context.Context, admin *sql.DB, migrationsDir string) {
	t.Helper()

	if _, err := admin.ExecContext(ctx, `DROP SCHEMA IF EXISTS lmt CASCADE`); err != nil {
		t.Fatalf("drop schema: %v", err)
	}
	if _, err := admin.ExecContext(ctx, `DROP TABLE IF EXISTS public.goose_db_version`); err != nil {
		t.Fatalf("drop goose version table: %v", err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("set dialect: %v", err)
	}
	if err := goose.UpContext(ctx, admin, migrationsDir); err != nil {
		t.Fatalf("goose up: %v", err)
	}
}

// sliceSeed carries the ids of the seeded vertical-slice fixture.
type sliceSeed struct {
	driverID         uuid.UUID
	shipmentID       uuid.UUID
	routeID          uuid.UUID
	routePointSource uuid.UUID
	pkg              uuid.UUID
	withDestPoint    bool
}

// seedSlice inserts shipment/route/route_point/package/package_route_point
// rows for the load slice (source point LOAD + optional destination UNLOAD).
func seedSlice(t *testing.T, ctx context.Context, admin *sql.DB, s sliceSeed) {
	t.Helper()

	routePointDest := uuid.New()
	seed := []string{
		`INSERT INTO lmt.shipment (shipment_id, shipment_short_id, shipment_status, driver_id, car_license_plate)
		 VALUES ('` + s.shipmentID.String() + `', 900001, 4, '` + s.driverID.String() + `', 'E777AA77')`,
		`INSERT INTO lmt.route (route_id, shipment_id, routecar_id)
		 VALUES ('` + s.routeID.String() + `', '` + s.shipmentID.String() + `', 1)`,
		`INSERT INTO lmt.route_point (route_point_id, route_id, shipment_id, routecar_point_id, seq_num, route_point_status)
		 VALUES ('` + s.routePointSource.String() + `', '` + s.routeID.String() + `', '` + s.shipmentID.String() + `', 1001, 1, 0)`,
		`INSERT INTO lmt.package (package_id, shipment_id, package_status, tare_id, tare_type)
		 VALUES ('` + s.pkg.String() + `', '` + s.shipmentID.String() + `', 1, 42, 'BOX')`,
		`INSERT INTO lmt.package_route_point (package_id, route_point_id, shipment_id, action, action_status)
		 VALUES ('` + s.pkg.String() + `', '` + s.routePointSource.String() + `', '` + s.shipmentID.String() + `', 1, 1)`,
	}
	if s.withDestPoint {
		seed = append(seed,
			`INSERT INTO lmt.route_point (route_point_id, route_id, shipment_id, routecar_point_id, seq_num, route_point_status)
			 VALUES ('`+routePointDest.String()+`', '`+s.routeID.String()+`', '`+s.shipmentID.String()+`', 1002, 2, 0)`,
			`INSERT INTO lmt.package_route_point (package_id, route_point_id, shipment_id, action, action_status)
			 VALUES ('`+s.pkg.String()+`', '`+routePointDest.String()+`', '`+s.shipmentID.String()+`', 2, 1)`,
		)
	}

	for _, stmt := range seed {
		if _, err := admin.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("seed: %v\n%s", err, stmt)
		}
	}
}

// ensureDB creates the target database if it does not exist yet (connect via
// the maintenance "postgres" database derived from the DSN).
func ensureDB(t *testing.T, ctx context.Context, dsn, dbName string) {
	t.Helper()

	adminDSN := strings.Replace(dsn, dbName, "postgres", 1)
	admin, err := sql.Open("pgx", adminDSN)
	if err != nil {
		t.Fatalf("open maintenance db: %v", err)
	}
	defer func() { _ = admin.Close() }()

	if _, err := admin.ExecContext(ctx,
		`CREATE DATABASE `+dbName); err != nil {
		// 42P04 = duplicate_database: already exists — fine.
		if pgErr, ok := err.(*pgconn.PgError); !ok || pgErr.Code != "42P04" {
			t.Fatalf("create database %s: %v", dbName, err)
		}
	}
}
