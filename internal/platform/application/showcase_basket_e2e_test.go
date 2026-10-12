//go:build e2e

// End-to-end tests of the driver showcase + basket vertical (ARCHITECTURE.md
// §23 flows 1-2): real PostgreSQL (migrations + clean tables), real gRPC server
// with the driver wiring, real basket adapter over lmt.basket and the
// deterministic showcase slice stubs.
//
// Run: go test -tags e2e -run 'TestE2EShowcaseBasket' -v ./internal/platform/application
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
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"

	publicdriver "last-mile-task/internal/application/public_driver"
	drivergrpc "last-mile-task/internal/application/public_driver/transport/grpc"
	"last-mile-task/internal/platform/database"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure/postgrestest"
	basketpg "last-mile-task/internal/shipping/queries/driver/basket/basketpg"
	showcasepg "last-mile-task/internal/shipping/queries/driver/showcase/showcasepg"
	driverpb "last-mile-task/pkg/gen/api/proto/v1/public/driver"
)

// sbHarness wires the showcase/basket slice over real PostgreSQL.
type sbHarness struct {
	client driverpb.DriverClient
	app    *publicdriver.Application
	repo   *database.Repository
	driver values.Driver
}

func newSBHarness(t *testing.T, driver values.Driver) *sbHarness {
	return newSBHarnessCleaned(t, driver, true)
}

// newSBHarnessCleaned builds the showcase/basket harness, optionally wiping the
// shared test tables first. Use clean=false when a test needs a second driver
// against the same database rows (e.g. proving a foreign driver cannot mutate
// another driver's basket) — otherwise the wipe would erase the owner's rows.
func newSBHarnessCleaned(t *testing.T, driver values.Driver, clean bool) *sbHarness {
	t.Helper()

	repo := postgrestest.Repo(t)
	if clean {
		postgrestest.CleanTables(t, repo.Pool)
	}

	routeProvider := showcasepg.NewRouteProvider(nil)
	showcaseStore := showcasepg.New(
		showcasepg.NewOfficeProvider(nil),
		routeProvider,
		showcasepg.NewCargoProvider(),
	)
	basketStore := basketpg.New(repo, routeProvider)

	app := publicdriver.New(publicdriver.Config{
		DriversProvider: &csDriverProvider{driver: driver},
		ShowcaseStore:   showcaseStore,
		BasketStore:     basketStore,
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

	return &sbHarness{
		client: driverpb.NewDriverClient(conn),
		app:    app,
		repo:   repo,
		driver: driver,
	}
}

func addItemCtx(ctx context.Context, routeID int64) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "route_id", sbItoa(routeID))
}

func deleteItemCtx(ctx context.Context, itemID uuid.UUID) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "item_id", itemID.String())
}

func listRoutesCtx(ctx context.Context, pointID int64) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "point_id", sbItoa(pointID))
}

func sbBasketCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, driverID uuid.UUID) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM lmt.basket WHERE driver_id = $1`, driverID,
	).Scan(&n); err != nil {
		t.Fatalf("basket count: %v", err)
	}
	return n
}

// TestE2EShowcaseBasket_AddItem proves flow 1 (basket): AddItem writes the row
// into lmt.basket, a duplicate AddItem is a no-op, ListItems returns the
// current driver's basket, DeleteItem/ClearBasket remove the rows and a foreign
// item maps to NotFound.
func TestE2EShowcaseBasket_AddItem(t *testing.T) {
	ctx := context.Background()
	driver := values.Driver{ID: values.NewID(), SupplierID: 7}
	h := newSBHarness(t, driver)

	if _, err := h.client.AddItem(addItemCtx(ctx, 101), &emptypb.Empty{}); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if n := sbBasketCount(t, ctx, h.repo.Pool, uuid.UUID(driver.ID)); n != 1 {
		t.Fatalf("expected 1 basket row after AddItem, got %d", n)
	}

	// Duplicate AddItem must not create a second row.
	if _, err := h.client.AddItem(addItemCtx(ctx, 101), &emptypb.Empty{}); err != nil {
		t.Fatalf("AddItem (duplicate): %v", err)
	}
	if n := sbBasketCount(t, ctx, h.repo.Pool, uuid.UUID(driver.ID)); n != 1 {
		t.Fatalf("expected no duplicate, got %d rows", n)
	}

	// ListItems (app layer: the Empty contract drops the RPC payload) resolves
	// the route display data through the route stub.
	items, err := h.app.ListItems(ctx)
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(items) != 1 || items[0].RouteID != 101 || items[0].RouteName == "" {
		t.Fatalf("unexpected basket items: %+v", items)
	}

	// DeleteItem removes the item.
	if _, err := h.client.DeleteItem(deleteItemCtx(ctx, items[0].ItemID), &emptypb.Empty{}); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}
	if n := sbBasketCount(t, ctx, h.repo.Pool, uuid.UUID(driver.ID)); n != 0 {
		t.Fatalf("expected an empty basket after DeleteItem, got %d", n)
	}

	// ClearBasket on an empty basket is not an error.
	if _, err := h.client.ClearBasket(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("ClearBasket: %v", err)
	}
}

// TestE2EShowcaseBasket_AddItemNotFound proves an unknown route maps to
// NotFound and leaves the basket unchanged.
func TestE2EShowcaseBasket_AddItemNotFound(t *testing.T) {
	ctx := context.Background()
	driver := values.Driver{ID: values.NewID(), SupplierID: 7}
	h := newSBHarness(t, driver)

	_, err := h.client.AddItem(addItemCtx(ctx, 999), &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected a NotFound error for an unknown route")
	}
	if n := sbBasketCount(t, ctx, h.repo.Pool, uuid.UUID(driver.ID)); n != 0 {
		t.Fatalf("expected the basket to be unchanged, got %d", n)
	}
}

// TestE2EShowcaseBasket_DeleteForeignItem proves deleting an item of another
// driver maps to NotFound without changing that basket.
func TestE2EShowcaseBasket_DeleteForeignItem(t *testing.T) {
	ctx := context.Background()
	owner := values.Driver{ID: values.NewID(), SupplierID: 7}
	h := newSBHarness(t, owner)

	if _, err := h.client.AddItem(addItemCtx(ctx, 101), &emptypb.Empty{}); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	items, err := h.app.ListItems(ctx)
	if err != nil || len(items) != 1 {
		t.Fatalf("ListItems: %v (items=%d)", err, len(items))
	}

	// A different driver tries to delete the owner's item. Build its harness
	// against the same rows (clean=false) so the owner's basket survives.
	other := values.Driver{ID: values.NewID(), SupplierID: 7}
	ho := newSBHarnessCleaned(t, other, false)

	_, err = ho.client.DeleteItem(deleteItemCtx(ctx, items[0].ItemID), &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected a NotFound error for a foreign item")
	}
	if n := sbBasketCount(t, ctx, h.repo.Pool, uuid.UUID(owner.ID)); n != 1 {
		t.Fatalf("expected the owner basket to be unchanged, got %d", n)
	}
}

// TestE2EShowcase_ListRoutesByPoint proves flow 2 (showcase): ListRoutesByPoint
// returns deterministic routes for a point and an unknown point maps to
// NotFound.
func TestE2EShowcase_ListRoutesByPoint(t *testing.T) {
	ctx := context.Background()
	driver := values.Driver{ID: values.NewID(), SupplierID: 7}
	h := newSBHarness(t, driver)

	routes, _, err := h.app.ListRoutesByPoint(ctx, publicdriver.ListRoutesByPointRequest{
		PointID: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListRoutesByPoint: %v", err)
	}
	if len(routes) == 0 {
		t.Fatal("expected at least one route for point 1")
	}
	found := false
	for _, r := range routes {
		if r.ID == 101 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected route 101 to serve point 1, got %+v", routes)
	}

	if _, _, err := h.app.ListRoutesByPoint(ctx, publicdriver.ListRoutesByPointRequest{
		PointID: 999, PageSize: 10,
	}); err == nil {
		t.Fatal("expected a NotFound error for an unknown point")
	}

	// The wire must surface the unknown point as NotFound, not Internal:
	// pointExists returns a typed resource error that mapErrorToGRPC resolves.
	if _, err := h.client.ListRoutesByPoint(listRoutesCtx(ctx, 999), &emptypb.Empty{}); status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound over gRPC for an unknown point, got %v", status.Code(err))
	}
}

// TestE2EShowcase_ListPoints proves the showcase points pagination over the
// deterministic slice stub.
func TestE2EShowcase_ListPoints(t *testing.T) {
	ctx := context.Background()
	driver := values.Driver{ID: values.NewID(), SupplierID: 7}
	h := newSBHarness(t, driver)

	points, resp, err := h.app.ListPoints(ctx, publicdriver.ListPointsRequest{PageSize: 2})
	if err != nil {
		t.Fatalf("ListPoints: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(points))
	}
	if !resp.HasMore {
		t.Fatal("expected more pages")
	}
}

func sbItoa(n int64) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = digits[n%10]
		n /= 10
	}
	return string(buf[i:])
}