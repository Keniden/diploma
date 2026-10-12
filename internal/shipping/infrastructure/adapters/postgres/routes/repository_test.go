package routes_test

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
	"last-mile-task/internal/shipping/infrastructure/adapters/postgres/routes"
	"last-mile-task/internal/shipping/infrastructure/postgrestest"
)

func newRepo(t *testing.T) (*routes.RouteRepository, *database.Repository) {
	t.Helper()

	base := postgrestest.Repo(t)
	return routes.NewRepository(base), base
}

func point(seqNum int32, pointID int64, status values.RoutePointStatus) *entities.RoutePoint {
	return &entities.RoutePoint{
		ID:       values.RoutePointID(uuid.New()),
		PointID:  values.PointID(pointID),
		SeqNum:   seqNum,
		Distance: int64(pointID) * 100,
		Status:   status,
	}
}

func route(routeID, shipmentID uuid.UUID, points ...*entities.RoutePoint) *entities.Route {
	return &entities.Route{
		ID:         values.RouteID(routeID),
		ShipmentID: values.ShipmentID(shipmentID),
		TemplateID: values.RouteTemplateID(777),
		Distance:   1234,
		Duration:   5678,
		Points:     points,
	}
}

func TestRouteRepository_SaveAndGetByID(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	routeID := uuid.New()
	shipmentID := uuid.New()
	p1 := point(0, 1001, values.RoutePointStatusWaitingArrival)
	p2 := point(1, 1002, values.RoutePointStatusDriverArrived)
	p3 := point(2, 1003, values.RoutePointStatusCompleted)

	require.NoError(t, repo.SaveRoute(ctx, route(routeID, shipmentID, p1, p2, p3)))

	got, err := repo.GetRouteByID(ctx, values.RouteID(routeID))
	require.NoError(t, err)
	require.Equal(t, values.RouteID(routeID), got.ID)
	require.Equal(t, values.ShipmentID(shipmentID), got.ShipmentID)
	require.Equal(t, values.RouteTemplateID(777), got.TemplateID)
	require.Equal(t, int64(1234), got.Distance)
	require.Equal(t, int64(5678), got.Duration)
	require.Len(t, got.Points, 3)

	// points are ordered by seq_num and fully mapped
	require.Equal(t, values.PointID(1001), got.Points[0].PointID)
	require.Equal(t, values.RoutePointStatusWaitingArrival, got.Points[0].Status)
	require.Equal(t, values.PointID(1002), got.Points[1].PointID)
	require.Equal(t, values.RoutePointStatusDriverArrived, got.Points[1].Status)
	require.Equal(t, values.PointID(1003), got.Points[2].PointID)
	require.Equal(t, values.RoutePointStatusCompleted, got.Points[2].Status)
	require.Equal(t, p1.ID, got.Points[0].ID)
}

func TestRouteRepository_GetRouteByShipmentID(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	shipmentID := uuid.New()
	oldRouteID := uuid.New()
	newRouteID := uuid.New()

	require.NoError(t, repo.SaveRoute(ctx, route(oldRouteID, shipmentID, point(0, 1, values.RoutePointStatusWaitingArrival), point(1, 2, values.RoutePointStatusWaitingArrival))))
	// emulate a later route for the same shipment
	time.Sleep(1100 * time.Millisecond) // created_at has second-level default granularity in the fixture below

	require.NoError(t, repo.SaveRoute(ctx, route(newRouteID, shipmentID, point(0, 3, values.RoutePointStatusWaitingArrival), point(1, 4, values.RoutePointStatusWaitingArrival))))

	got, err := repo.GetRouteByShipmentID(ctx, values.ShipmentID(shipmentID))
	require.NoError(t, err)
	require.Equal(t, values.RouteID(newRouteID), got.ID)
}

func TestRouteRepository_GetRouteByIDNotFound(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	_, err := repo.GetRouteByID(ctx, values.RouteID(uuid.New()))
	require.True(t, errors.Is(err, ports.ErrRouteNotFound), "error: %v", err)

	_, err = repo.GetRouteByShipmentID(ctx, values.ShipmentID(uuid.New()))
	require.True(t, errors.Is(err, ports.ErrRouteNotFound), "error: %v", err)
}

func TestRouteRepository_SaveRouteSoftDeletesRemovedPoints(t *testing.T) {
	ctx := context.Background()
	repo, base := newRepo(t)

	routeID := uuid.New()
	shipmentID := uuid.New()
	p1 := point(0, 1001, values.RoutePointStatusWaitingArrival)
	p2 := point(1, 1002, values.RoutePointStatusWaitingArrival)
	p3 := point(2, 1003, values.RoutePointStatusWaitingArrival)

	require.NoError(t, repo.SaveRoute(ctx, route(routeID, shipmentID, p1, p2, p3)))

	// package_route_point rows for the removed point
	_, err := base.Pool.Exec(ctx, `
		INSERT INTO lmt.package_route_point (package_id, route_point_id, shipment_id, action, action_status)
		VALUES ($1, $2, $3, 1, 1)`,
		uuid.New(), p3.ID, shipmentID)
	require.NoError(t, err)

	require.NoError(t, repo.SaveRoute(ctx, route(routeID, shipmentID, p1, p2)))

	got, err := repo.GetRouteByID(ctx, values.RouteID(routeID))
	require.NoError(t, err)
	require.Len(t, got.Points, 2)

	var p3DeletedAt *time.Time
	require.NoError(t, base.Pool.QueryRow(ctx,
		"SELECT deleted_at FROM lmt.route_point WHERE route_point_id = $1", p3.ID).Scan(&p3DeletedAt))
	require.NotNil(t, p3DeletedAt, "removed route point must be soft deleted")

	var prpActive int
	require.NoError(t, base.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM lmt.package_route_point WHERE route_point_id = $1 AND deleted_at IS NULL",
		p3.ID).Scan(&prpActive))
	require.Equal(t, 0, prpActive)
}

func TestRouteRepository_SaveOptimizedRoute(t *testing.T) {
	ctx := context.Background()
	repo, base := newRepo(t)

	routeID := uuid.New()
	shipmentID := uuid.New()
	p1 := point(0, 1001, values.RoutePointStatusWaitingArrival)
	p2 := point(1, 1002, values.RoutePointStatusWaitingArrival)
	p3 := point(2, 1003, values.RoutePointStatusWaitingArrival)
	require.NoError(t, repo.SaveRoute(ctx, route(routeID, shipmentID, p1, p2, p3)))

	// optimization drops the middle point and swaps the distance
	optimized := route(routeID, shipmentID, p1, p3)
	optimized.Distance = 9999
	optimized.Duration = 8888
	require.NoError(t, repo.SaveOptimizedRoute(ctx, optimized))

	got, err := repo.GetRouteByID(ctx, values.RouteID(routeID))
	require.NoError(t, err)
	require.Equal(t, int64(9999), got.Distance)
	require.Equal(t, int64(8888), got.Duration)
	require.Len(t, got.Points, 2)
	require.Equal(t, p1.ID, got.Points[0].ID)
	require.Equal(t, p3.ID, got.Points[1].ID)

	var p2DeletedAt *time.Time
	require.NoError(t, base.Pool.QueryRow(ctx,
		"SELECT deleted_at FROM lmt.route_point WHERE route_point_id = $1", p2.ID).Scan(&p2DeletedAt))
	require.NotNil(t, p2DeletedAt)
}
