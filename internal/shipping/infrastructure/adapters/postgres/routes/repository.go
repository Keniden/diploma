package routes

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"

	"last-mile-task/internal/platform/database"
	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
)

var (
	_ ports.RouteRepository = (*RouteRepository)(nil)
	_ ports.Transactional   = (*RouteRepository)(nil)
	// TODO(LASTMILE-061): the real repository also asserts
	// optimizer.RouteOptimizationRepository and implements SaveOptimizedRoute;
	// internal/shipping/application/optimizer is not ported yet, restore once
	// it lands.
)

type RouteRepository struct {
	*database.Repository
}

func NewRepository(repo *database.Repository) *RouteRepository {
	return &RouteRepository{
		Repository: repo,
	}
}

// GetRouteByID implements [ports.RouteRepository].
func (r *RouteRepository) GetRouteByID(ctx context.Context, id values.RouteID) (*entities.Route, error) {
	query, args, err := r.Builder.
		Select(
			"route_id",
			"shipment_id",
			"routecar_id",
			"distance",
			"duration",
		).
		From("lmt.route").
		Where(sq.Eq{"route_id": id}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build route query: %w", err)
	}

	var rr routeRecord
	if err := pgxscan.Get(ctx, r.Querier(ctx), &rr, query, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get route %s: %w", id, ports.ErrRouteNotFound)
		}
		return nil, fmt.Errorf("get route: %w", err)
	}

	route := routeFromRecord(rr)
	if err := r.loadRouteAggregate(ctx, route); err != nil {
		return nil, err
	}

	return route, nil
}

// GetRouteByShipmentID implements [ports.RouteRepository] and partially [policies.ShipmentPolicyRepository].
func (r *RouteRepository) GetRouteByShipmentID(ctx context.Context, id values.ShipmentID) (*entities.Route, error) {
	query, args, err := r.Builder.
		Select(
			"route_id",
			"shipment_id",
			"routecar_id",
			"distance",
			"duration",
		).
		From("lmt.route").
		Where(sq.Eq{"shipment_id": id}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build route by shipment query: %w", err)
	}

	var rr routeRecord
	if err := pgxscan.Get(ctx, r.Querier(ctx), &rr, query, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get route by shipment %s: %w", id, ports.ErrRouteNotFound)
		}
		return nil, fmt.Errorf("get route by shipment: %w", err)
	}

	route := routeFromRecord(rr)
	if err := r.loadRouteAggregate(ctx, route); err != nil {
		return nil, err
	}

	return route, nil
}

// SaveRoute implements [ports.RouteRepository].
func (r *RouteRepository) SaveRoute(ctx context.Context, route *entities.Route) error {
	routeQuery, routeArgs, err := r.Builder.
		Insert("lmt.route").
		SetMap(map[string]any{
			"route_id":    route.ID,
			"shipment_id": route.ShipmentID,
			"routecar_id": int64(route.TemplateID),
			"distance":    route.Distance,
			"duration":    route.Duration,
		}).
		Suffix(`
ON CONFLICT (route_id) DO UPDATE SET
	shipment_id = EXCLUDED.shipment_id,
	routecar_id = EXCLUDED.routecar_id,
	distance = EXCLUDED.distance,
	duration = EXCLUDED.duration,
	updated_at = NOW()`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build save route query: %w", err)
	}

	if _, err := r.Querier(ctx).Exec(ctx, routeQuery, routeArgs...); err != nil {
		return fmt.Errorf("save route: %w", err)
	}

	now := time.Now()

	pointIDs := make([]values.RoutePointID, 0, len(route.Points))
	for _, rp := range route.Points {
		pointIDs = append(pointIDs, rp.ID)

		pointQuery, pointArgs, err := r.Builder.
			Insert("lmt.route_point").
			SetMap(map[string]any{
				"route_point_id":     rp.ID,
				"route_id":           route.ID,
				"shipment_id":        route.ShipmentID,
				"routecar_point_id":  int64(rp.PointID),
				"seq_num":            rp.SeqNum,
				"distance":           rp.Distance,
				"route_point_status": int(rp.Status),
				"deleted_at":         nil,
			}).
			Suffix(`
ON CONFLICT (route_point_id) DO UPDATE SET
	route_id = EXCLUDED.route_id,
	shipment_id = EXCLUDED.shipment_id,
	routecar_point_id = EXCLUDED.routecar_point_id,
	seq_num = EXCLUDED.seq_num,
	distance = EXCLUDED.distance,
	route_point_status = EXCLUDED.route_point_status,
	updated_at = NOW(),
	deleted_at = NULL`).
			ToSql()
		if err != nil {
			return fmt.Errorf("build save route point query: %w", err)
		}

		if _, execErr := r.Querier(ctx).Exec(ctx, pointQuery, pointArgs...); execErr != nil {
			return fmt.Errorf("save route point: %w", execErr)
		}
	}

	// Soft-delete route points removed from aggregate.
	if len(pointIDs) > 0 {
		var deletedPointIDs []values.RoutePointID

		delPointsQuery, delPointsArgs, err := r.Builder.
			Update("lmt.route_point").
			Set("deleted_at", now).
			Set("updated_at", sq.Expr("NOW()")).
			Where(sq.Eq{
				"route_id":    route.ID,
				"shipment_id": route.ShipmentID,
				"deleted_at":  nil,
			}).
			Where(sq.NotEq{"route_point_id": pointIDs}).
			Suffix("RETURNING route_point_id").
			ToSql()
		if err != nil {
			return fmt.Errorf("build delete route points query: %w", err)
		}

		if err := pgxscan.Select(ctx, r.Querier(ctx), &deletedPointIDs, delPointsQuery, delPointsArgs...); err != nil {
			return fmt.Errorf("soft delete route points: %w", err)
		}

		if len(deletedPointIDs) > 0 {
			delPRPQuery, delPRPArgs, err := r.Builder.
				Update("lmt.package_route_point").
				Set("deleted_at", now).
				Set("updated_at", sq.Expr("NOW()")).
				Where(sq.Eq{
					"shipment_id": route.ShipmentID,
					"deleted_at":  nil,
				}).
				Where(sq.Eq{"route_point_id": deletedPointIDs}).
				ToSql()
			if err != nil {
				return fmt.Errorf("build delete prp by deleted route points query: %w", err)
			}

			if _, err := r.Querier(ctx).Exec(ctx, delPRPQuery, delPRPArgs...); err != nil {
				return fmt.Errorf("soft delete prp by deleted route points: %w", err)
			}
		}
	}

	return nil
}

// SaveOptimizedRoute implements [optimizer.RouteOptimizationRepository].
//
// TODO(LASTMILE-061): restore when internal/shipping/application/optimizer is
// ported (see the var block above); the implementation is ported 1:1 from the
// real repository.
func (r *RouteRepository) SaveOptimizedRoute(ctx context.Context, route *entities.Route) error {
	updRouteQuery, updRouteArgs, err := r.Builder.
		Update("lmt.route").
		Set("distance", route.Distance).
		Set("duration", route.Duration).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"route_id": route.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update route query: %w", err)
	}
	if _, err := r.Querier(ctx).Exec(ctx, updRouteQuery, updRouteArgs...); err != nil {
		return fmt.Errorf("update route: %w", err)
	}

	now := time.Now()

	pointIDs := make([]values.RoutePointID, 0, len(route.Points))
	for _, rp := range route.Points {
		pointIDs = append(pointIDs, rp.ID)
	}

	// Soft-delete route points removed by optimization and soft-delete their package_route_point rows.
	var deletedPointIDs []values.RoutePointID
	{
		q := r.Builder.
			Update("lmt.route_point").
			Set("deleted_at", now).
			Set("updated_at", sq.Expr("NOW()")).
			Where(sq.Eq{
				"route_id":    route.ID,
				"shipment_id": route.ShipmentID,
				"deleted_at":  nil,
			})
		if len(pointIDs) > 0 {
			q = q.Where(sq.NotEq{"route_point_id": pointIDs})
		}

		sql, args, err := q.Suffix("RETURNING route_point_id").ToSql()
		if err != nil {
			return fmt.Errorf("build soft delete optimized-out points query: %w", err)
		}

		if err := pgxscan.Select(ctx, r.Querier(ctx), &deletedPointIDs, sql, args...); err != nil {
			return fmt.Errorf("soft delete optimized-out points: %w", err)
		}
	}

	if len(deletedPointIDs) > 0 {
		delPRPQuery, delPRPArgs, err := r.Builder.
			Update("lmt.package_route_point").
			Set("deleted_at", now).
			Set("updated_at", sq.Expr("NOW()")).
			Where(sq.Eq{
				"shipment_id": route.ShipmentID,
				"deleted_at":  nil,
			}).
			Where(sq.Eq{"route_point_id": deletedPointIDs}).
			ToSql()
		if err != nil {
			return fmt.Errorf("build soft delete prp for optimized-out points query: %w", err)
		}
		if _, err := r.Querier(ctx).Exec(ctx, delPRPQuery, delPRPArgs...); err != nil {
			return fmt.Errorf("soft delete prp for optimized-out points: %w", err)
		}
	}

	for _, rp := range route.Points {
		updPointQuery, updPointArgs, err := r.Builder.
			Update("lmt.route_point").
			Set("seq_num", rp.SeqNum).
			Set("distance", rp.Distance).
			Set("updated_at", sq.Expr("NOW()")).
			Where(sq.Eq{
				"route_point_id": rp.ID,
				"route_id":       route.ID,
				"shipment_id":    route.ShipmentID,
			}).
			ToSql()
		if err != nil {
			return fmt.Errorf("build update route point query: %w", err)
		}
		if _, err := r.Querier(ctx).Exec(ctx, updPointQuery, updPointArgs...); err != nil {
			return fmt.Errorf("update route point: %w", err)
		}
	}

	return nil
}

func (r *RouteRepository) loadRouteAggregate(ctx context.Context, route *entities.Route) error {
	pointsQuery, pointsArgs, err := r.Builder.
		Select(
			"route_point_id",
			"route_id",
			"shipment_id",
			"routecar_point_id",
			"seq_num",
			"distance",
			"route_point_status",
			"deleted_at",
		).
		From("lmt.route_point").
		Where(sq.Eq{
			"route_id":    route.ID,
			"shipment_id": route.ShipmentID,
		}).
		Where(sq.Eq{"deleted_at": nil}).
		OrderBy("seq_num ASC").
		ToSql()
	if err != nil {
		return fmt.Errorf("build route points query: %w", err)
	}

	var pointRecs []routePointRecord
	if scanErr := pgxscan.Select(ctx, r.Querier(ctx), &pointRecs, pointsQuery, pointsArgs...); scanErr != nil {
		return fmt.Errorf("get route points: %w", scanErr)
	}

	route.Points = make([]*entities.RoutePoint, 0, len(pointRecs))

	for _, pr := range pointRecs {
		rp := routePointFromRecord(pr)
		route.Points = append(route.Points, rp)
	}

	return nil
}
