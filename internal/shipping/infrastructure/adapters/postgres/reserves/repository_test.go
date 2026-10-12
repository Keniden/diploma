package reserves_test

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
	"last-mile-task/internal/shipping/infrastructure/adapters/postgres/reserves"
	"last-mile-task/internal/shipping/infrastructure/postgrestest"
)

func newRepo(t *testing.T) (*reserves.ReserveRepository, *database.Repository) {
	t.Helper()

	base := postgrestest.Repo(t)
	return reserves.NewRepository(base), base
}

func reserve(shipmentID, routePointID uuid.UUID, status values.ReserveStatus) *entities.Reserve {
	r := &entities.Reserve{
		ID:           values.ReserveID(uuid.New()),
		ShipmentID:   values.ShipmentID(shipmentID),
		RoutePointID: values.RoutePointID(routePointID),
		Status:       status,
		Gate: values.ReserveGate{
			ID:   values.GateID(123),
			Name: "Gate A",
		},
	}
	switch status {
	case values.ReserveStatusWaitingApproval:
		expires := time.Now().Add(10 * time.Minute)
		r.ExpiresAt = &expires
		r.Task = values.ReserveTask{ID: values.ReserveTaskID("task-1")}
	}
	return r
}

func TestReserveRepository_SaveAndGetByID(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	shipmentID := uuid.New()
	routePointID := uuid.New()
	r := reserve(shipmentID, routePointID, values.ReserveStatusWaitingApproval)

	require.NoError(t, repo.SaveReserve(ctx, r))

	got, err := repo.GetReserveByID(ctx, values.ReserveID(r.ID))
	require.NoError(t, err)
	require.Equal(t, r.ID, got.ID)
	require.Equal(t, values.ShipmentID(shipmentID), got.ShipmentID)
	require.Equal(t, values.RoutePointID(routePointID), got.RoutePointID)
	require.Equal(t, values.ReserveStatusWaitingApproval, got.Status)
	require.Equal(t, values.ReserveTaskID("task-1"), got.Task.ID)
	require.Equal(t, values.GateID(123), got.Gate.ID)
	require.Equal(t, "Gate A", got.Gate.Name)
	require.NotNil(t, got.ExpiresAt)
	require.Nil(t, got.ScannedAt)

	// approve -> update via upsert
	require.NoError(t, got.Approve())
	require.NoError(t, repo.SaveReserve(ctx, got))

	got2, err := repo.GetReserveByID(ctx, values.ReserveID(r.ID))
	require.NoError(t, err)
	require.Equal(t, values.ReserveStatusActive, got2.Status)
	require.NotNil(t, got2.ScannedAt)
}

func TestReserveRepository_GetReserveByShipmentAndRoutePoint(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	shipmentID := uuid.New()
	routePointID := uuid.New()

	// released reserve must not be returned (it is the latest)
	released := reserve(shipmentID, routePointID, values.ReserveStatusWaitingApproval)
	require.NoError(t, repo.SaveReserve(ctx, released))
	released.Status = values.ReserveStatusReleased
	require.NoError(t, repo.SaveReserve(ctx, released))

	// active reserve created afterwards
	active := reserve(shipmentID, routePointID, values.ReserveStatusWaitingApproval)
	require.NoError(t, repo.SaveReserve(ctx, active))
	active.Status = values.ReserveStatusActive
	require.NoError(t, repo.SaveReserve(ctx, active))

	got, err := repo.GetReserveByShipmentID(ctx, values.ShipmentID(shipmentID))
	require.NoError(t, err)
	require.Equal(t, active.ID, got.ID)

	got, err = repo.GetReserveByRoutePointID(ctx, values.RoutePointID(routePointID))
	require.NoError(t, err)
	require.Equal(t, active.ID, got.ID)

	_, err = repo.GetReserveByShipmentID(ctx, values.ShipmentID(uuid.New()))
	require.True(t, errors.Is(err, ports.ErrReserveNotFound), "error: %v", err)
}

// cleanStaleReserves removes reserve rows expired more than a minute ago.
// The adapter integration tests share one test database across runs and
// packages, so stale expired rows from previous runs would fill the LIMIT
// window of the ListExpired* queries and make their assertions flaky.
func cleanStaleReserves(t *testing.T, ctx context.Context, repo *reserves.ReserveRepository) {
	t.Helper()

	query := `DELETE FROM lmt.reserve
	          WHERE expires_at IS NOT NULL
	            AND expires_at < NOW() - INTERVAL '1 minute'`
	if _, err := repo.Querier(ctx).Exec(ctx, query); err != nil {
		t.Fatalf("clean stale reserves: %v", err)
	}
}

func TestReserveRepository_ListExpiredReserves(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	cleanStaleReserves(t, ctx, repo)

	expiredWaiting := reserve(uuid.New(), uuid.New(), values.ReserveStatusWaitingApproval)
	expired := time.Now().Add(-time.Hour)
	expiredWaiting.ExpiresAt = &expired
	require.NoError(t, repo.SaveReserve(ctx, expiredWaiting))

	expiredActive := reserve(uuid.New(), uuid.New(), values.ReserveStatusActive)
	expiredActive.ExpiresAt = &expired
	require.NoError(t, repo.SaveReserve(ctx, expiredActive))

	notExpired := reserve(uuid.New(), uuid.New(), values.ReserveStatusWaitingApproval)
	require.NoError(t, repo.SaveReserve(ctx, notExpired))

	waiting, err := repo.ListExpiredWaitingApprovalReserves(ctx, 10)
	require.NoError(t, err)
	found := false
	for _, r := range waiting {
		if r.ID == expiredWaiting.ID {
			found = true
		}
		require.NotEqual(t, notExpired.ID, r.ID, "not expired reserve must not be listed")
	}
	require.True(t, found, "expired waiting approval reserve must be listed")

	active, err := repo.ListExpiredActiveReserves(ctx, 10)
	require.NoError(t, err)
	found = false
	for _, r := range active {
		if r.ID == expiredActive.ID {
			found = true
		}
		require.NotEqual(t, notExpired.ID, r.ID, "not expired reserve must not be listed")
	}
	require.True(t, found, "expired active reserve must be listed")
}

func TestReserveRepository_GetReserveCountByShipmentID(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	shipmentID := uuid.New()
	require.NoError(t, repo.SaveReserve(ctx, reserve(shipmentID, uuid.New(), values.ReserveStatusQueued)))
	require.NoError(t, repo.SaveReserve(ctx, reserve(shipmentID, uuid.New(), values.ReserveStatusWaitingApproval)))

	count, err := repo.GetReserveCountByShipmentID(ctx, values.ShipmentID(shipmentID))
	require.NoError(t, err)
	require.Equal(t, 2, count)

	count, err = repo.GetReserveCountByShipmentID(ctx, values.ShipmentID(shipmentID), values.ReserveStatusQueued)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// unknown status must be rejected
	_, err = repo.GetReserveCountByShipmentID(ctx, values.ShipmentID(shipmentID), values.ReserveStatus(99))
	require.Error(t, err)
}

func TestReserveRepository_UnknownStatusRejectedOnSave(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	r := reserve(uuid.New(), uuid.New(), values.ReserveStatus(99))
	require.Error(t, repo.SaveReserve(ctx, r))
}
