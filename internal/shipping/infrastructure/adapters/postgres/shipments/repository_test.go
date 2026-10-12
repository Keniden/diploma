package shipments_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"last-mile-task/internal/platform/database"
	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure/adapters/postgres/shipments"
	"last-mile-task/internal/shipping/infrastructure/postgrestest"
)

func newRepo(t *testing.T) (*shipments.ShipmentRepository, *database.Repository) {
	t.Helper()

	base := postgrestest.Repo(t)
	return shipments.NewRepository(base), base
}

func shipment(id uuid.UUID, shortID int64, status values.ShipmentStatus, driverID uuid.UUID, supplierID uint64) *entities.Shipment {
	return &entities.Shipment{
		ID:           values.ShipmentID(id),
		ShortID:      values.ShipmentShortID(shortID),
		Status:       status,
		DriverID:     values.DriverID(driverID),
		SupplierID:   values.SupplierID(supplierID),
		LicensePlate: values.LicensePlate(""),
	}
}

func TestShipmentRepository_SaveAndGetByID(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	shipmentID := uuid.New()
	driverID := uuid.New()
	s := shipment(shipmentID, 123456, values.ShipmentStatusAssigned, driverID, 42)
	s.LicensePlate = values.LicensePlate("А123ВС777")

	require.NoError(t, repo.SaveShipment(ctx, s))

	got, err := repo.GetShipmentByID(ctx, values.ShipmentID(shipmentID))
	require.NoError(t, err)
	require.Equal(t, values.ShipmentID(shipmentID), got.ID)
	require.Equal(t, values.ShipmentShortID(123456), got.ShortID)
	require.Equal(t, values.ShipmentStatusAssigned, got.Status)
	require.Equal(t, values.DriverID(driverID), got.DriverID)
	require.Equal(t, values.SupplierID(42), got.SupplierID)
	require.Equal(t, values.LicensePlate("А123ВС777"), got.LicensePlate)
}

func TestShipmentRepository_SaveUpsertClearsLicensePlate(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	shipmentID := uuid.New()
	driverID := uuid.New()

	require.NoError(t, repo.SaveShipment(ctx, shipment(shipmentID, 100, values.ShipmentStatusAssigned, driverID, 7)))

	updated := shipment(shipmentID, 101, values.ShipmentStatusInProgress, driverID, 8)
	updated.LicensePlate = values.LicensePlate("B777OP77")
	require.NoError(t, repo.SaveShipment(ctx, updated))

	got, err := repo.GetShipmentByID(ctx, values.ShipmentID(shipmentID))
	require.NoError(t, err)
	require.Equal(t, values.ShipmentStatusInProgress, got.Status)
	require.Equal(t, values.SupplierID(8), got.SupplierID)
	require.Equal(t, values.LicensePlate("B777OP77"), got.LicensePlate)
}

func TestShipmentRepository_GetShipmentByIDNotFound(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	_, err := repo.GetShipmentByID(ctx, values.ShipmentID(uuid.New()))
	require.Error(t, err)
	require.True(t, errors.Is(err, ports.ErrShipmentNotFound), "error: %v", err)
}

func TestShipmentRepository_GetShipmentByIDSoftDeleted(t *testing.T) {
	ctx := context.Background()
	repo, base := newRepo(t)

	shipmentID := uuid.New()
	require.NoError(t, repo.SaveShipment(ctx, shipment(shipmentID, 200, values.ShipmentStatusNew, uuid.New(), 1)))

	_, err := base.Pool.Exec(ctx, "UPDATE lmt.shipment SET deleted_at = NOW() WHERE shipment_id = $1", shipmentID)
	require.NoError(t, err)

	_, err = repo.GetShipmentByID(ctx, values.ShipmentID(shipmentID))
	require.True(t, errors.Is(err, ports.ErrShipmentNotFound), "error: %v", err)
}

func TestShipmentRepository_GetShipmentCountByDriverID(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	driverID := uuid.New()
	for i := int64(0); i < 3; i++ {
		require.NoError(t, repo.SaveShipment(ctx, shipment(uuid.New(), 300+i, values.ShipmentStatusAssigned, driverID, 1)))
	}

	count, err := repo.GetShipmentCountByDriverID(ctx, values.DriverID(driverID))
	require.NoError(t, err)
	require.Equal(t, 3, count)

	count, err = repo.GetShipmentCountByDriverID(ctx, values.DriverID(driverID), values.ShipmentStatusAssigned)
	require.NoError(t, err)
	require.Equal(t, 3, count)

	count, err = repo.GetShipmentCountByDriverID(ctx, values.DriverID(driverID), values.ShipmentStatusInProgress)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// other drivers are not counted
	count, err = repo.GetShipmentCountByDriverID(ctx, values.DriverID(uuid.New()))
	require.NoError(t, err)
	require.Equal(t, 0, count)
}
