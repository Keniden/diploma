package pgdriver

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/samber/lo"

	"last-mile-task/internal/shipping/queries/driver/shipment/readmodels"
)

func (s *Store) GetRouteByShipmentID(ctx context.Context, id uuid.UUID) (readmodels.Route, error) {
	routes, err := s.GetRoutesByShipmentIDs(ctx, []uuid.UUID{id})
	if err != nil {
		return readmodels.Route{}, err
	}
	if len(routes) == 0 {
		return readmodels.Route{}, fmt.Errorf("route not found for shipment %s: %w", id, err)
	}
	return routes[0], nil
}

func (s *Store) GetRoutePointsByShipmentID(ctx context.Context, id uuid.UUID) ([]readmodels.RoutePoint, error) {
	return s.GetRoutePointsByShipmentIDs(ctx, []uuid.UUID{id})
}

func (s *Store) GetRoutesByShipmentIDs(ctx context.Context, ids []uuid.UUID) ([]readmodels.Route, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query, args, err := s.Builder.
		Select(
			"route_id",
			"shipment_id",
			"routecar_id",
			"distance",
			"duration",
			"created_at",
			"updated_at",
		).
		From("lmt.route").
		Where(squirrel.Eq{
			"shipment_id": ids,
		}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed when making query to get routes: %w", err)
	}

	var records []routeRecord
	if err = pgxscan.Select(ctx, s.Querier(ctx), &records, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get routes: %w", err)
	}

	return lo.Map(records, func(r routeRecord, _ int) readmodels.Route {
		return mapRecordToRoute(r)
	}), nil
}

func (s *Store) GetRoutePointsByShipmentIDs(ctx context.Context, ids []uuid.UUID) ([]readmodels.RoutePoint, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query, args, err := s.Builder.
		Select(
			"route_point_id",
			"route_id",
			"shipment_id",
			"routecar_point_id",
			"seq_num",
			"route_point_status",
			"distance",
			"created_at",
			"updated_at",
		).
		From("lmt.route_point").
		Where(
			squirrel.And{
				squirrel.Eq{
					"shipment_id": ids,
				},
				squirrel.Eq{
					"deleted_at": nil,
				},
			}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed when making query to get route points: %w", err)
	}

	var records []routePointRecord
	if err = pgxscan.Select(ctx, s.Querier(ctx), &records, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get route points: %w", err)
	}

	return lo.Map(records, func(r routePointRecord, _ int) readmodels.RoutePoint {
		return mapRecordToRoutePoint(r)
	}), nil
}
