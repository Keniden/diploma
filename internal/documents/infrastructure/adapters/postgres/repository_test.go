package documentspg_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"last-mile-task/internal/documents/domain/entities"
	"last-mile-task/internal/documents/domain/values"
	"last-mile-task/internal/documents/infrastructure/adapters/postgres"
	"last-mile-task/internal/platform/database"
	"last-mile-task/internal/shipping/infrastructure/postgrestest"
)

func newRepo(t *testing.T) (*documentspg.DocumentRepository, *database.Repository) {
	t.Helper()

	base := postgrestest.Repo(t)
	cleanTables(t, base)
	return documentspg.NewRepository(base), base
}

// cleanTables deletes the tables the documents tests touch in FK-safe order
// (documents before shipment/route_point, since lmt.documents references both).
func cleanTables(t *testing.T, base *database.Repository) {
	t.Helper()

	tables := []string{
		"lmt.document_package",
		"lmt.documents",
		"lmt.package_route_point",
		"lmt.package",
		"lmt.route_point",
		"lmt.route",
		"lmt.shipment",
		"lmt.outbox",
	}
	for _, table := range tables {
		_, err := base.Pool.Exec(context.Background(), "DELETE FROM "+table)
		require.NoError(t, err, "clean table %s", table)
	}
}

func seedShipment(t *testing.T, ctx context.Context, base *database.Repository, shipmentID uuid.UUID) (from, to uuid.UUID) {
	t.Helper()

	routeID := uuid.New()
	from = uuid.New()
	to = uuid.New()

	rows := []string{
		`INSERT INTO lmt.shipment (shipment_id, shipment_short_id, shipment_status, driver_id, car_license_plate)
		 VALUES ('` + shipmentID.String() + `', 900001, 4, '` + uuid.New().String() + `', 'E777AA77')`,
		`INSERT INTO lmt.route (route_id, shipment_id, routecar_id)
		 VALUES ('` + routeID.String() + `', '` + shipmentID.String() + `', 1)`,
		`INSERT INTO lmt.route_point (route_point_id, route_id, shipment_id, routecar_point_id, seq_num, route_point_status)
		 VALUES ('` + from.String() + `', '` + routeID.String() + `', '` + shipmentID.String() + `', 1001, 1, 0)`,
		`INSERT INTO lmt.route_point (route_point_id, route_id, shipment_id, routecar_point_id, seq_num, route_point_status)
		 VALUES ('` + to.String() + `', '` + routeID.String() + `', '` + shipmentID.String() + `', 1002, 2, 0)`,
	}
	for _, stmt := range rows {
		_, err := base.Pool.Exec(ctx, stmt)
		require.NoError(t, err)
	}

	return from, to
}

func newDocument(t *testing.T, shipmentID, from, to uuid.UUID, eventUUID string, pkgIDs ...uuid.UUID) *entities.Document {
	t.Helper()

	doc, err := entities.NewDocument(values.DocumentSpec{
		ShipmentID:       shipmentID,
		FromRoutePointID: from,
		ToRoutePointID:   to,
		DocumentType:     values.DocumentTypeWaybillForward,
		EventUUID:        eventUUID,
		ExternalID:       1,
		PackageIDs:       pkgIDs,
	})
	require.NoError(t, err)
	return doc
}

func TestDocumentRepository_SaveAndGetByShipmentID(t *testing.T) {
	ctx := context.Background()
	repo, base := newRepo(t)

	shipmentID := uuid.New()
	from, to := seedShipment(t, ctx, base, shipmentID)

	pkgA, pkgB := uuid.New(), uuid.New()
	doc := newDocument(t, shipmentID, from, to, "evt-1", pkgA, pkgB)

	txCtx, err := base.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.Save(txCtx, doc))
	require.NoError(t, base.Commit(txCtx))

	got, err := repo.GetByShipmentID(ctx, shipmentID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, doc.ID, got[0].ID)
	require.Equal(t, doc.ShipmentID, got[0].ShipmentID)
	require.Equal(t, doc.EventUUID, got[0].EventUUID)
	require.Equal(t, values.DocumentTypeWaybillForward, got[0].DocumentType)
	require.ElementsMatch(t, []uuid.UUID{pkgA, pkgB}, uuidSlice(got[0].PackageIDs))
}

func TestDocumentRepository_GetByEventUUID(t *testing.T) {
	ctx := context.Background()
	repo, base := newRepo(t)

	shipmentID := uuid.New()
	from, to := seedShipment(t, ctx, base, shipmentID)

	doc := newDocument(t, shipmentID, from, to, "evt-42", uuid.New(), uuid.New())

	txCtx, err := base.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.Save(txCtx, doc))
	require.NoError(t, base.Commit(txCtx))

	got, err := repo.GetByEventUUID(ctx, "evt-42")
	require.NoError(t, err)
	require.Equal(t, doc.ID, got.ID)
	require.Equal(t, "evt-42", got.EventUUID)
	require.Len(t, got.PackageIDs, 2)
}

func TestDocumentRepository_GetByEventUUIDNotFound(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	_, err := repo.GetByEventUUID(ctx, "missing")
	require.Error(t, err)
	require.True(t, errors.Is(err, entities.ErrDocumentNotFound), "error: %v", err)
}

func TestDocumentRepository_GetByShipmentIDEmpty(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	got, err := repo.GetByShipmentID(ctx, uuid.New())
	require.NoError(t, err)
	require.Len(t, got, 0)
}

func uuidSlice(ids []values.PackageID) []uuid.UUID {
	out := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		out[i] = uuid.UUID(id)
	}
	return out
}