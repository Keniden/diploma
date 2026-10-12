package eventdeps_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"last-mile-task/internal/platform/database"
	"last-mile-task/internal/shipping/infrastructure/eventdeps"
	"last-mile-task/internal/shipping/infrastructure/postgrestest"
)

func newStore(t *testing.T) (*eventdeps.Store, *database.Repository) {
	t.Helper()

	base := postgrestest.Repo(t)
	postgrestest.CleanTables(t, base.Pool)
	return eventdeps.NewStore(base), base
}

func seedShipmentFixture(t *testing.T, ctx context.Context, base *database.Repository) (shipmentID, source, dest uuid.UUID) {
	t.Helper()

	shipmentID, source, dest = uuid.New(), uuid.New(), uuid.New()
	routeID := uuid.New()

	stmt := []string{
		`INSERT INTO lmt.shipment (shipment_id, shipment_short_id, shipment_status, driver_id, car_license_plate)
		 VALUES ('` + shipmentID.String() + `', 900001, 4, '` + uuid.New().String() + `', 'E777AA77')`,
		`INSERT INTO lmt.route (route_id, shipment_id, routecar_id)
		 VALUES ('` + routeID.String() + `', '` + shipmentID.String() + `', 1)`,
		`INSERT INTO lmt.route_point (route_point_id, route_id, shipment_id, routecar_point_id, seq_num, route_point_status)
		 VALUES ('` + source.String() + `', '` + routeID.String() + `', '` + shipmentID.String() + `', 1001, 1, 0)`,
		`INSERT INTO lmt.route_point (route_point_id, route_id, shipment_id, routecar_point_id, seq_num, route_point_status)
		 VALUES ('` + dest.String() + `', '` + routeID.String() + `', '` + shipmentID.String() + `', 1002, 2, 0)`,
		`INSERT INTO lmt.package (package_id, shipment_id, package_status, tare_id, tare_type, is_return)
		 VALUES ('` + uuid.New().String() + `', '` + shipmentID.String() + `', 2, 42, 'BOX', false)`,
		`INSERT INTO lmt.package (package_id, shipment_id, package_status, tare_id, tare_type, is_return)
		 VALUES ('` + uuid.New().String() + `', '` + shipmentID.String() + `', 2, 43, 'BOX', true)`,
	}
	for _, s := range stmt {
		_, err := base.Pool.Exec(ctx, s)
		require.NoError(t, err)
	}

	return shipmentID, source, dest
}

func TestStoreShipmentPackagesResolvesDirection(t *testing.T) {
	ctx := context.Background()
	store, base := newStore(t)

	shipmentID, source, dest := seedShipmentFixture(t, ctx, base)

	// Forward package: load at source, unload at dest.
	pkg := uuid.New()
	_, err := base.Pool.Exec(ctx, `INSERT INTO lmt.package (package_id, shipment_id, package_status, tare_id, tare_type, is_return)
		VALUES ($1, $2, 2, 44, 'BOX', false)`, pkg, shipmentID)
	require.NoError(t, err)
	_, err = base.Pool.Exec(ctx, `INSERT INTO lmt.package_route_point (package_id, route_point_id, shipment_id, action, action_status)
		VALUES ($1, $2, $3, 1, 1)`, pkg, source, shipmentID)
	require.NoError(t, err)
	_, err = base.Pool.Exec(ctx, `INSERT INTO lmt.package_route_point (package_id, route_point_id, shipment_id, action, action_status)
		VALUES ($1, $2, $3, 2, 1)`, pkg, dest, shipmentID)
	require.NoError(t, err)

	pkgs, err := store.ShipmentPackages(ctx, shipmentID)
	require.NoError(t, err)

	var found bool
	for _, p := range pkgs {
		if p.ID == pkg {
			found = true
			require.Equal(t, source, uuid.UUID(p.From))
			require.Equal(t, dest, uuid.UUID(p.To))
		}
	}
	require.True(t, found, "expected the seeded package with resolved direction")
}

func TestStorePackagesForDirectionFiltersByLoadPoint(t *testing.T) {
	ctx := context.Background()
	store, base := newStore(t)

	shipmentID, source, dest := seedShipmentFixture(t, ctx, base)

	pkg := uuid.New()
	_, err := base.Pool.Exec(ctx, `INSERT INTO lmt.package (package_id, shipment_id, package_status, tare_id, tare_type, is_return)
		VALUES ($1, $2, 2, 45, 'BOX', true)`, pkg, shipmentID)
	require.NoError(t, err)
	_, err = base.Pool.Exec(ctx, `INSERT INTO lmt.package_route_point (package_id, route_point_id, shipment_id, action, action_status)
		VALUES ($1, $2, $3, 1, 1)`, pkg, source, shipmentID)
	require.NoError(t, err)
	_, err = base.Pool.Exec(ctx, `INSERT INTO lmt.package_route_point (package_id, route_point_id, shipment_id, action, action_status)
		VALUES ($1, $2, $3, 2, 1)`, pkg, dest, shipmentID)
	require.NoError(t, err)

	pkgs, err := store.PackagesForDirection(ctx, shipmentID, source)
	require.NoError(t, err)
	require.Len(t, pkgs, 1)
	require.Equal(t, pkg, pkgs[0].ID)

	// No packages loaded at a different point.
	pkgs, err = store.PackagesForDirection(ctx, shipmentID, dest)
	require.NoError(t, err)
	require.Len(t, pkgs, 0)
}

func TestStoreHasLoadedReturnPackagesAtRoutePoint(t *testing.T) {
	ctx := context.Background()
	store, base := newStore(t)

	shipmentID, source, dest := seedShipmentFixture(t, ctx, base)

	// The seeded is_return=true package has no package_route_point rows, so
	// there are no loaded return packages at the source point yet.
	has, err := store.HasLoadedReturnPackagesAtRoutePoint(ctx, source)
	require.NoError(t, err)
	require.False(t, has)

	// Add a loaded return package at the source point.
	pkg := uuid.New()
	_, err = base.Pool.Exec(ctx, `INSERT INTO lmt.package (package_id, shipment_id, package_status, tare_id, tare_type, is_return)
		VALUES ($1, $2, 2, 46, 'BOX', true)`, pkg, shipmentID)
	require.NoError(t, err)
	_, err = base.Pool.Exec(ctx, `INSERT INTO lmt.package_route_point (package_id, route_point_id, shipment_id, action, action_status)
		VALUES ($1, $2, $3, 1, 2)`, pkg, source, shipmentID)
	require.NoError(t, err)

	has, err = store.HasLoadedReturnPackagesAtRoutePoint(ctx, source)
	require.NoError(t, err)
	require.True(t, has)

	has, err = store.HasLoadedReturnPackagesAtRoutePoint(ctx, dest)
	require.NoError(t, err)
	require.False(t, has)
}