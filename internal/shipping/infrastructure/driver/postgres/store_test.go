package pgdriver_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	lmterrors "last-mile-task/internal/errors"
	"last-mile-task/internal/platform/database"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure/adapters/postgres/packages"
	"last-mile-task/internal/shipping/infrastructure/adapters/postgres/reserves"
	"last-mile-task/internal/shipping/infrastructure/adapters/postgres/routes"
	"last-mile-task/internal/shipping/infrastructure/adapters/postgres/shipments"
	"last-mile-task/internal/shipping/infrastructure/driver/postgres"
	"last-mile-task/internal/shipping/infrastructure/postgrestest"
	"last-mile-task/internal/shipping/queries/driver/shipment"
	"last-mile-task/internal/shipping/queries/driver/shipment/readmodels"
)

// fixture seeds every table the store reads, using the write adapters, so the
// integration test covers the full write/read path through the same schema.
type fixture struct {
	store      *pgdriver.Store
	base       *database.Repository
	shipmentID uuid.UUID
	driverID   uuid.UUID
	routeID    uuid.UUID
	points     []uuid.UUID
	loadPoint  uuid.UUID
	unloadPt   uuid.UUID
	pkgID      uuid.UUID
	returnPkg  uuid.UUID
}

func setup(t *testing.T) *fixture {
	t.Helper()

	base := postgrestest.Repo(t)

	ctx := context.Background()
	f := &fixture{
		store:      pgdriver.New(base),
		base:       base,
		shipmentID: uuid.New(),
		driverID:   uuid.New(),
		routeID:    uuid.New(),
	}

	f.loadPoint = uuid.New()
	f.unloadPt = uuid.New()
	f.points = []uuid.UUID{
		uuid.New(),  // source (load)
		f.loadPoint, // TODO: keep load on a middle point for realism
		f.unloadPt,
	}

	// shipment
	shipmentRepo := shipments.NewRepository(base)
	require.NoError(t, shipmentRepo.SaveShipment(ctx, &entities.Shipment{
		ID:           values.ShipmentID(f.shipmentID),
		ShortID:      values.ShipmentShortID(777001),
		Status:       values.ShipmentStatusInProgress,
		DriverID:     values.DriverID(f.driverID),
		SupplierID:   values.SupplierID(15),
		LicensePlate: values.LicensePlate("А001АА77"),
	}))

	// route with three points
	routeRepo := routes.NewRepository(base)
	require.NoError(t, routeRepo.SaveRoute(ctx, &entities.Route{
		ID:         values.RouteID(f.routeID),
		ShipmentID: values.ShipmentID(f.shipmentID),
		TemplateID: values.RouteTemplateID(42),
		Distance:   1000,
		Duration:   500,
		Points: []*entities.RoutePoint{
			{ID: values.RoutePointID(f.points[0]), PointID: values.PointID(1), SeqNum: 0, Distance: 0, Status: values.RoutePointStatusWaitingArrival},
			{ID: values.RoutePointID(f.points[1]), PointID: values.PointID(2), SeqNum: 1, Distance: 100, Status: values.RoutePointStatusDriverArrived},
			{ID: values.RoutePointID(f.points[2]), PointID: values.PointID(3), SeqNum: 2, Distance: 900, Status: values.RoutePointStatusCompleted},
		},
	}))

	// packages (direct + return), both with load at source and unload at target
	now := time.Now().UTC()
	pkgRepo := packages.NewRepository(base)
	f.pkgID = uuid.New()
	require.NoError(t, pkgRepo.SavePackage(ctx, &entities.Package{
		ID:         values.PackageID(f.pkgID),
		ShipmentID: values.ShipmentID(f.shipmentID),
		Status:     values.PackageStatusInTransit,
		IsReturn:   false,
		TareID:     values.TareID(100500),
		TareType:   values.TareType("BOX"),
		Actions: []*entities.PackageAction{
			{RoutePointID: values.RoutePointID(f.points[0]), Type: values.RoutePointActionTypeLoad, Status: values.RoutePointPackageStatusDone, StatusUpdatedAt: &now},
			{RoutePointID: values.RoutePointID(f.points[2]), Type: values.RoutePointActionTypeUnload, Status: values.RoutePointPackageStatusTODO},
		},
	}))

	f.returnPkg = uuid.New()
	require.NoError(t, pkgRepo.SavePackage(ctx, &entities.Package{
		ID:         values.PackageID(f.returnPkg),
		ShipmentID: values.ShipmentID(f.shipmentID),
		Status:     values.PackageStatusLoaded,
		IsReturn:   true,
		TareID:     values.TareID(100501),
		TareType:   values.TareType("BOX"),
		Actions: []*entities.PackageAction{
			{RoutePointID: values.RoutePointID(f.points[1]), Type: values.RoutePointActionTypeLoad, Status: values.RoutePointPackageStatusDone, StatusUpdatedAt: &now},
			{RoutePointID: values.RoutePointID(f.points[2]), Type: values.RoutePointActionTypeUnload, Status: values.RoutePointPackageStatusTODO},
		},
	}))

	// active reserve
	reserveRepo := reserves.NewRepository(base)
	expires := time.Now().Add(30 * time.Minute)
	require.NoError(t, reserveRepo.SaveReserve(ctx, &entities.Reserve{
		ID:           values.ReserveID(uuid.New()),
		ShipmentID:   values.ShipmentID(f.shipmentID),
		RoutePointID: values.RoutePointID(f.points[0]),
		Status:       values.ReserveStatusActive,
		Gate:         values.ReserveGate{ID: values.GateID(9), Name: "Gate 9"},
		Task:         values.ReserveTask{ID: values.ReserveTaskID("task-777")},
		ScannedAt:    &now,
		ExpiresAt:    &expires,
	}))

	return f
}

func TestStore_GetShipmentByID(t *testing.T) {
	ctx := context.Background()
	f := setup(t)

	got, err := f.store.GetShipmentByID(ctx, f.shipmentID)
	require.NoError(t, err)
	require.Equal(t, f.shipmentID, got.ID)
	require.Equal(t, int64(777001), got.ShortID)
	require.Equal(t, int(values.ShipmentStatusInProgress), got.Status)
	require.Equal(t, f.driverID, got.DriverID)
	require.Equal(t, uint64(15), got.SupplierID)
	require.Equal(t, "А001АА77", got.LicensePlate)
	require.Nil(t, got.DeletedAt)
	require.False(t, got.CreatedAt.IsZero())
}

func TestStore_GetShipmentByIDNotFound(t *testing.T) {
	ctx := context.Background()
	f := setup(t)

	_, err := f.store.GetShipmentByID(ctx, uuid.New())
	require.Error(t, err)
	require.True(t, lmterrors.Is(err, "lmt_not_found"), "error: %v", err)
}

func TestStore_GetDriverShipmentsByStatuses(t *testing.T) {
	ctx := context.Background()
	f := setup(t)

	got, err := f.store.GetDriverShipmentsByStatuses(ctx, values.DriverID(f.driverID), values.ShipmentStatusInProgress)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, f.shipmentID, got[0].ID)

	got, err = f.store.GetDriverShipmentsByStatuses(ctx, values.DriverID(f.driverID), values.ShipmentStatusCompleted)
	require.NoError(t, err)
	require.Empty(t, got)

	got, err = f.store.GetDriverShipmentsByStatuses(ctx, values.DriverID(uuid.New()), values.ShipmentStatusInProgress)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestStore_GetRoutesAndPoints(t *testing.T) {
	ctx := context.Background()
	f := setup(t)

	routes, err := f.store.GetRoutesByShipmentIDs(ctx, []uuid.UUID{f.shipmentID})
	require.NoError(t, err)
	require.Len(t, routes, 1)
	require.Equal(t, f.routeID, routes[0].ID)
	require.Equal(t, int64(42), routes[0].TemplateID)
	require.Equal(t, int64(1000), routes[0].Distance)

	points, err := f.store.GetRoutePointsByShipmentIDs(ctx, []uuid.UUID{f.shipmentID})
	require.NoError(t, err)
	require.Len(t, points, 3)
	require.Equal(t, f.points[0], points[0].ID)
	require.Equal(t, int(values.RoutePointStatusDriverArrived), points[1].Status)
	require.Equal(t, int64(900), points[2].Distance)

	// no rows for unknown shipment
	routes, err = f.store.GetRoutesByShipmentIDs(ctx, []uuid.UUID{uuid.New()})
	require.NoError(t, err)
	require.Empty(t, routes)

	points, err = f.store.GetRoutePointsByShipmentIDs(ctx, nil)
	require.NoError(t, err)
	require.Nil(t, points)
}

func TestStore_GetPackagesAndActions(t *testing.T) {
	ctx := context.Background()
	f := setup(t)

	pkgs, err := f.store.GetPackagesByShipmentIDs(ctx, []uuid.UUID{f.shipmentID})
	require.NoError(t, err)
	require.Len(t, pkgs, 2)

	byID := map[uuid.UUID]readmodels.Package{}
	for _, p := range pkgs {
		byID[p.ID] = p
	}
	require.Equal(t, true, byID[f.returnPkg].IsReturn)
	require.Equal(t, values.TareID(100500), byID[f.pkgID].TareID)
	require.Equal(t, values.TareType("BOX"), byID[f.pkgID].TareType)

	actions, err := f.store.GetPackageActionsByShipmentIDs(ctx, []uuid.UUID{f.shipmentID})
	require.NoError(t, err)
	require.Len(t, actions, 4)

	// direct package: load done, unload TODO
	unloads := 0
	for _, a := range actions {
		if a.PackageID == f.pkgID && a.Action == int(values.RoutePointActionTypeUnload) {
			unloads++
			require.Equal(t, int(values.RoutePointPackageStatusTODO), a.Status)
			require.Equal(t, f.unloadPt, a.RoutePointID)
		}
	}
	require.Equal(t, 1, unloads)
}

func TestStore_GetDstPointByPackageIDs(t *testing.T) {
	ctx := context.Background()
	f := setup(t)

	dst, err := f.store.GetDstPointByPackageIDs(ctx, []uuid.UUID{f.pkgID, f.returnPkg}, f.shipmentID)
	require.NoError(t, err)
	require.Len(t, dst, 2)
	require.Equal(t, f.unloadPt, dst[f.pkgID])
	require.Equal(t, f.unloadPt, dst[f.returnPkg])
}

func TestStore_HasLoadedReturnPackagesAtRoutePoint(t *testing.T) {
	ctx := context.Background()
	f := setup(t)

	// return package was LOADED at points[1]
	has, err := f.store.HasLoadedReturnPackagesAtRoutePoint(ctx, f.points[1])
	require.NoError(t, err)
	require.True(t, has)

	// direct load action is at points[0], no return package there
	has, err = f.store.HasLoadedReturnPackagesAtRoutePoint(ctx, f.points[0])
	require.NoError(t, err)
	require.False(t, has)
}

func TestStore_GetActiveShipmentReserve(t *testing.T) {
	ctx := context.Background()
	f := setup(t)

	reserve, err := f.store.GetActiveShipmentReserveByShipmentID(ctx, f.shipmentID)
	require.NoError(t, err)
	require.NotNil(t, reserve)
	require.NotNil(t, reserve.ExpiresAt)

	reservesMap, err := f.store.GetActiveShipmentReservesByShipmentIDs(ctx, []uuid.UUID{f.shipmentID})
	require.NoError(t, err)
	require.Len(t, reservesMap, 1)
	require.NotZero(t, reservesMap[f.shipmentID].UpdatedAt)

	// shipment without reserve
	reserve, err = f.store.GetActiveShipmentReserveByShipmentID(ctx, uuid.New())
	require.NoError(t, err)
	require.Nil(t, reserve)

	// empty input
	reservesMap, err = f.store.GetActiveShipmentReservesByShipmentIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, reservesMap)
}

func TestStore_IsShipmentStore(t *testing.T) {
	// compile-time check that the store satisfies the shipment query interface
	var _ shipment.ShipmentStore = (*pgdriver.Store)(nil)
}
