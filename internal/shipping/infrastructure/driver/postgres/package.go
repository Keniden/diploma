package pgdriver

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/samber/lo"

	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/queries/driver/shipment/readmodels"
)

func (s *Store) GetPackagesByShipmentID(ctx context.Context, id uuid.UUID) ([]readmodels.Package, error) {
	return s.GetPackagesByShipmentIDs(ctx, []uuid.UUID{id})
}

func (s *Store) GetPackageActionsByShipmentID(ctx context.Context, id uuid.UUID) ([]readmodels.PackageAction, error) {
	return s.GetPackageActionsByShipmentIDs(ctx, []uuid.UUID{id})
}

func (s *Store) GetPackagesByShipmentIDs(ctx context.Context, ids []uuid.UUID) ([]readmodels.Package, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query, args, err := s.Builder.
		Select(
			"package_id",
			"shipment_id",
			"is_return",
			"tare_id",
			"tare_type",
			"package_status",
			"created_at",
			"updated_at",
			"deleted_at",
		).
		From("lmt.package").
		Where(
			squirrel.And{
				squirrel.Eq{
					"shipment_id": ids,
				},
				squirrel.Eq{
					"deleted_at": nil,
				},
				squirrel.NotEq{"package_status": int(values.PackageStatusCanceled)},
			},
		).ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed when making query to get packages: %w", err)
	}

	var records []packageRecord
	if err = pgxscan.Select(ctx, s.Querier(ctx), &records, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get packages: %w", err)
	}

	return lo.Map(records, func(r packageRecord, _ int) readmodels.Package {
		return mapRecordToPackage(r)
	}), nil
}

func (s *Store) GetPackageActionsByShipmentIDs(ctx context.Context, ids []uuid.UUID) ([]readmodels.PackageAction, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query, args, err := s.Builder.
		Select(
			"prp.package_id",
			"prp.route_point_id",
			"p.shipment_id",
			"prp.action",
			"prp.action_status as status",
			"prp.action_status_updated_at as action_timestamp",
			"prp.created_at",
			"prp.updated_at",
		).
		From("lmt.package_route_point prp").
		Join("lmt.package p ON prp.package_id = p.package_id").
		Where(squirrel.Eq{
			"p.shipment_id": ids,
		}).
		Where(squirrel.Eq{"p.deleted_at": nil}).
		Where(squirrel.Eq{"prp.deleted_at": nil}).
		Where(squirrel.NotEq{"p.package_status": int(values.PackageStatusCanceled)}).
		Where(squirrel.NotEq{"prp.action_status": int(values.RoutePointPackageStatusCanceled)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("builder: %w", err)
	}

	var records []packageActionRecord
	if err = pgxscan.Select(ctx, s.Querier(ctx), &records, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get package actions: %w", err)
	}

	return lo.Map(records, func(r packageActionRecord, _ int) readmodels.PackageAction {
		return mapRecordToPackageAction(r)
	}), nil
}

func (s *Store) HasLoadedReturnPackagesAtRoutePoint(ctx context.Context, routePointID uuid.UUID) (bool, error) {
	query, args, err := s.Builder.
		Select("1").
		From("lmt.package_route_point prp").
		Join("lmt.package p ON prp.package_id = p.package_id").
		Where(squirrel.Eq{
			"prp.route_point_id": routePointID,
			"prp.action":         int(values.RoutePointActionTypeLoad),
			"prp.deleted_at":     nil,
			"p.is_return":        true,
			"p.package_status": []int{
				int(values.PackageStatusLoaded),
				int(values.PackageStatusInTransit),
			},
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build has loaded returns query: %w", err)
	}

	rows, err := s.Querier(ctx).Query(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("exec has loaded returns: %w", err)
	}
	defer rows.Close()

	return rows.Next(), nil
}

func (s *Store) GetDstPointByPackageIDs(ctx context.Context, ids []uuid.UUID, shipmentID uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query, args, err := s.Builder.
		Select("package_id", "route_point_id").
		From("lmt.package_route_point").
		Where(squirrel.And{
			squirrel.Eq{"action": int(values.RoutePointActionTypeUnload)},
			squirrel.Eq{"package_id": ids},
			squirrel.Eq{"shipment_id": shipmentID},
			squirrel.NotEq{"action_status": int(values.RoutePointPackageStatusCanceled)},
		}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("builder: %w", err)
	}

	var records []dstPointRecord
	if err = pgxscan.Select(ctx, s.Querier(ctx), &records, query, args...); err != nil {
		return nil, fmt.Errorf("get dst points: %w", err)
	}

	out := make(map[uuid.UUID]uuid.UUID, len(records))
	for _, r := range records {
		out[r.PackageID] = r.RoutePointID
	}
	return out, nil
}
