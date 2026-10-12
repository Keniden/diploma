package packages_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"last-mile-task/internal/platform/database"
	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure/adapters/postgres/packages"
	"last-mile-task/internal/shipping/infrastructure/postgrestest"
)

func newRepo(t *testing.T) (*packages.PackageRepository, *database.Repository) {
	t.Helper()

	base := postgrestest.Repo(t)
	return packages.NewRepository(base), base
}

func newPackage(id, shipmentID, routePointID uuid.UUID, status values.PackageStatus) *entities.Package {
	return &entities.Package{
		ID:         values.PackageID(id),
		ShipmentID: values.ShipmentID(shipmentID),
		Status:     status,
		IsReturn:   false,
		TareID:     values.TareID(999001),
		TareType:   values.TareType("BOX"),
		Actions: []*entities.PackageAction{
			{
				RoutePointID: values.RoutePointID(routePointID),
				Type:         values.RoutePointActionTypeLoad,
				Status:       values.RoutePointPackageStatusTODO,
			},
			{
				RoutePointID: values.RoutePointID(uuid.New()),
				Type:         values.RoutePointActionTypeUnload,
				Status:       values.RoutePointPackageStatusTODO,
			},
		},
	}
}

func TestPackageRepository_SaveAndGet(t *testing.T) {
	ctx := context.Background()
	repo, base := newRepo(t)

	shipmentID := uuid.New()
	pkg := newPackage(uuid.New(), shipmentID, uuid.New(), values.PackageStatusReadyToLoad)

	require.NoError(t, repo.SavePackage(ctx, pkg))

	got, err := repo.GetPackageByID(ctx, values.ShipmentID(shipmentID), values.PackageID(pkg.ID))
	require.NoError(t, err)
	require.Equal(t, pkg.ID, got.ID)
	require.Equal(t, pkg.ShipmentID, got.ShipmentID)
	require.Equal(t, values.PackageStatusReadyToLoad, got.Status)
	require.Equal(t, values.TareID(999001), got.TareID)
	require.Equal(t, values.TareType("BOX"), got.TareType)
	require.Len(t, got.Actions, 2)

	byType := map[values.PackageActionType]*entities.PackageAction{}
	for _, a := range got.Actions {
		byType[a.Type] = a
	}
	require.Equal(t, values.RoutePointPackageStatusTODO, byType[values.RoutePointActionTypeLoad].Status)
	require.Nil(t, byType[values.RoutePointActionTypeLoad].StatusUpdatedAt)
	require.Equal(t, values.RoutePointPackageStatusTODO, byType[values.RoutePointActionTypeUnload].Status)

	// upsert keeps the same rows: re-save without the unload action
	pkg.Actions = pkg.Actions[:1]
	require.NoError(t, repo.SavePackage(ctx, pkg))

	got, err = repo.GetPackageByID(ctx, values.ShipmentID(shipmentID), values.PackageID(pkg.ID))
	require.NoError(t, err)
	require.Len(t, got.Actions, 1)
	require.Equal(t, values.RoutePointActionTypeLoad, got.Actions[0].Type)

	var activePRP int
	require.NoError(t, base.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM lmt.package_route_point WHERE package_id = $1 AND deleted_at IS NULL", pkg.ID).Scan(&activePRP))
	require.Equal(t, 1, activePRP)
}

func TestPackageRepository_GetByShipmentAndIDs(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	shipmentID := uuid.New()
	pkg1 := newPackage(uuid.New(), shipmentID, uuid.New(), values.PackageStatusReadyToLoad)
	pkg2 := newPackage(uuid.New(), shipmentID, uuid.New(), values.PackageStatusLoaded)
	require.NoError(t, repo.SavePackage(ctx, pkg1))
	require.NoError(t, repo.SavePackage(ctx, pkg2))

	byShipment, err := repo.GetPackagesByShipmentID(ctx, values.ShipmentID(shipmentID))
	require.NoError(t, err)
	require.Len(t, byShipment, 2)

	byIDs, err := repo.GetPackagesByIDs(ctx, values.ShipmentID(shipmentID), values.PackageID(pkg1.ID), values.PackageID(pkg2.ID))
	require.NoError(t, err)
	require.Len(t, byIDs, 2)

	byIDs, err = repo.GetPackagesByIDs(ctx, values.ShipmentID(shipmentID), values.PackageID(uuid.New()))
	require.NoError(t, err)
	require.Empty(t, byIDs)

	empty, err := repo.GetPackagesByIDs(ctx, values.ShipmentID(shipmentID))
	require.NoError(t, err)
	require.Empty(t, empty)

	none, err := repo.GetPackagesByShipmentID(ctx, values.ShipmentID(uuid.New()))
	require.NoError(t, err)
	require.Empty(t, none)
}

func TestPackageRepository_GetLoadedPackagesByShipmentID(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	shipmentID := uuid.New()
	for _, status := range []values.PackageStatus{
		values.PackageStatusReadyToLoad,
		values.PackageStatusLoaded,
		values.PackageStatusInTransit,
		values.PackageStatusUnloading,
		values.PackageStatusUnloaded,
	} {
		require.NoError(t, repo.SavePackage(ctx, newPackage(uuid.New(), shipmentID, uuid.New(), status)),
			"save package %d", status)
	}

	loaded, err := repo.GetLoadedPackagesByShipmentID(ctx, values.ShipmentID(shipmentID))
	require.NoError(t, err)
	require.Len(t, loaded, 3)

	statuses := map[values.PackageStatus]bool{}
	for _, p := range loaded {
		statuses[p.Status] = true
	}
	require.True(t, statuses[values.PackageStatusLoaded])
	require.True(t, statuses[values.PackageStatusInTransit])
	require.True(t, statuses[values.PackageStatusUnloading])
}

func TestPackageRepository_SavePackageScan(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	shipmentID := uuid.New()
	pkg := newPackage(uuid.New(), shipmentID, uuid.New(), values.PackageStatusReadyToLoad)
	require.NoError(t, repo.SavePackage(ctx, pkg))

	now := time.Now().UTC()
	pkg.Status = values.PackageStatusLoaded
	pkg.Actions[0].Status = values.RoutePointPackageStatusDone
	pkg.Actions[0].StatusUpdatedAt = &now

	require.NoError(t, repo.SavePackageScan(ctx, pkg, pkg.Actions[0]))

	got, err := repo.GetPackageByID(ctx, values.ShipmentID(shipmentID), values.PackageID(pkg.ID))
	require.NoError(t, err)
	require.Equal(t, values.PackageStatusLoaded, got.Status)

	var scanned bool
	var untouched bool
	for _, a := range got.Actions {
		if a.Type == values.RoutePointActionTypeLoad {
			require.Equal(t, values.RoutePointPackageStatusDone, a.Status)
			require.NotNil(t, a.StatusUpdatedAt)
			scanned = true
		}
		if a.Type == values.RoutePointActionTypeUnload {
			require.Equal(t, values.RoutePointPackageStatusTODO, a.Status)
			require.Nil(t, a.StatusUpdatedAt)
			untouched = true
		}
	}
	require.True(t, scanned, "load action must be scanned")
	require.True(t, untouched, "unload action must stay untouched")
}

func TestPackageRepository_DeletePackage(t *testing.T) {
	ctx := context.Background()
	repo, base := newRepo(t)

	shipmentID := uuid.New()
	pkg := newPackage(uuid.New(), shipmentID, uuid.New(), values.PackageStatusReadyToLoad)
	require.NoError(t, repo.SavePackage(ctx, pkg))

	require.NoError(t, repo.DeletePackage(ctx, values.ShipmentID(shipmentID), values.PackageID(pkg.ID)))

	_, err := repo.GetPackageByID(ctx, values.ShipmentID(shipmentID), values.PackageID(pkg.ID))
	require.True(t, errors.Is(err, ports.ErrPackageNotFound), "error: %v", err)

	var prpDeleted int
	require.NoError(t, base.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM lmt.package_route_point WHERE package_id = $1 AND deleted_at IS NOT NULL",
		pkg.ID).Scan(&prpDeleted))
	require.Equal(t, 2, prpDeleted)

	err = repo.DeletePackage(ctx, values.ShipmentID(shipmentID), values.PackageID(pkg.ID))
	require.Error(t, err)
}

func TestPackageRepository_CanceledPackageIsHidden(t *testing.T) {
	ctx := context.Background()
	repo, base := newRepo(t)

	shipmentID := uuid.New()
	pkg := newPackage(uuid.New(), shipmentID, uuid.New(), values.PackageStatusReadyToLoad)
	require.NoError(t, repo.SavePackage(ctx, pkg))

	_, err := base.Pool.Exec(ctx,
		"UPDATE lmt.package SET package_status = $1 WHERE package_id = $2",
		int(values.PackageStatusCanceled), pkg.ID)
	require.NoError(t, err)

	_, err = repo.GetPackageByID(ctx, values.ShipmentID(shipmentID), values.PackageID(pkg.ID))
	require.True(t, errors.Is(err, ports.ErrPackageNotFound), "error: %v", err)

	byShipment, err := repo.GetPackagesByShipmentID(ctx, values.ShipmentID(shipmentID))
	require.NoError(t, err)
	require.Empty(t, byShipment)
}

func TestPackageRepository_GetPackageCountByShipmentID(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	shipmentID := uuid.New()
	require.NoError(t, repo.SavePackage(ctx, newPackage(uuid.New(), shipmentID, uuid.New(), values.PackageStatusReadyToLoad)))
	require.NoError(t, repo.SavePackage(ctx, newPackage(uuid.New(), shipmentID, uuid.New(), values.PackageStatusLoaded)))

	count, err := repo.GetPackageCountByShipmentID(ctx, values.ShipmentID(shipmentID))
	require.NoError(t, err)
	require.Equal(t, 2, count)

	count, err = repo.GetPackageCountByShipmentID(ctx, values.ShipmentID(shipmentID), values.PackageStatusLoaded)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestPackageRepository_WithTxNestedSave(t *testing.T) {
	ctx := context.Background()
	repo, base := newRepo(t)

	shipmentID := uuid.New()
	pkg := newPackage(uuid.New(), shipmentID, uuid.New(), values.PackageStatusReadyToLoad)

	require.NoError(t, base.WithTx(ctx, func(txCtx context.Context) error {
		if err := repo.SavePackage(txCtx, pkg); err != nil {
			return err
		}
		return repo.SavePackage(txCtx, pkg)
	}))

	got, err := repo.GetPackageByID(ctx, values.ShipmentID(shipmentID), values.PackageID(pkg.ID))
	require.NoError(t, err)
	require.Len(t, got.Actions, 2)
}
