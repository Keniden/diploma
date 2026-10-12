package eventdeps

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"

	docdirect "last-mile-task/internal/documents/domain/directions"
	"last-mile-task/internal/shipping/domain/values"
)

// documentPackageRow is the per-package direction (load/unload route points)
// resolved for the documents domain.
type documentPackageRow struct {
	PackageID       uuid.UUID `db:"package_id"`
	TareID          int64     `db:"tare_id"`
	TareType        string    `db:"tare_type"`
	FromRoutePoint  *uuid.UUID `db:"from_route_point_id"`
	ToRoutePoint    *uuid.UUID `db:"to_route_point_id"`
}

// ShipmentPackages returns all non-canceled packages of a shipment with their
// load (From) and unload (To) route points resolved, so the documents handler
// can group them into forward/return directions.
func (s *Store) ShipmentPackages(
	ctx context.Context,
	shipmentID uuid.UUID,
) ([]docdirect.Package, error) {
	return s.packagesWithDirection(ctx, sq.Eq{
		"p.shipment_id": shipmentID,
	}, uuid.Nil)
}

// PackagesForDirection returns the packages loaded at the given route point
// (From), with their destination (To) route point resolved.
func (s *Store) PackagesForDirection(
	ctx context.Context,
	shipmentID uuid.UUID,
	fromRoutePointID uuid.UUID,
) ([]docdirect.Package, error) {
	return s.packagesWithDirection(ctx, sq.Eq{
		"p.shipment_id": shipmentID,
	}, fromRoutePointID)
}

// HasLoadedReturnPackagesAtRoutePoint reports whether the route point has
// loaded return packages; this drives ensureReturnDocsScheduled.
func (s *Store) HasLoadedReturnPackagesAtRoutePoint(
	ctx context.Context,
	routePointID uuid.UUID,
) (bool, error) {
	query, args, err := s.repo.Builder.
		Select("1").
		From("lmt.package_route_point prp").
		Join("lmt.package p ON prp.package_id = p.package_id").
		Where(sq.Eq{
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
		return false, fmt.Errorf("build has loaded return documents query: %w", err)
	}

	rows, err := s.repo.Querier(ctx).Query(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("exec has loaded return documents: %w", err)
	}
	defer rows.Close()

	return rows.Next(), nil
}

// packagesWithDirection runs the direction-resolving query for a shipment,
// optionally scoped to a single load route point (From).
func (s *Store) packagesWithDirection(
	ctx context.Context,
	where sq.Eq,
	fromRoutePointID uuid.UUID,
) ([]docdirect.Package, error) {
	builder := s.repo.Builder.
		Select(
			"p.package_id",
			"p.tare_id",
			"p.tare_type",
		).
		Column("load_prp.route_point_id AS from_route_point_id").
		Column("unload_prp.route_point_id AS to_route_point_id").
		From("lmt.package p").
		LeftJoin(
			"lmt.package_route_point load_prp"+
				" ON load_prp.package_id = p.package_id"+
				" AND load_prp.action = ? AND load_prp.deleted_at IS NULL",
			int(values.RoutePointActionTypeLoad),
		).
		LeftJoin(
			"lmt.package_route_point unload_prp"+
				" ON unload_prp.package_id = p.package_id"+
				" AND unload_prp.action = ? AND unload_prp.deleted_at IS NULL",
			int(values.RoutePointActionTypeUnload),
		).
		Where(where).
		Where(sq.Eq{"p.deleted_at": nil}).
		Where(sq.NotEq{"p.package_status": int(values.PackageStatusCanceled)}).
		Where(sq.NotEq{"unload_prp.action_status": int(values.RoutePointPackageStatusCanceled)})

	if fromRoutePointID != uuid.Nil {
		builder = builder.Where(sq.Eq{"load_prp.route_point_id": fromRoutePointID})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build document packages query: %w", err)
	}

	var rows []documentPackageRow
	if err := pgxscan.Select(ctx, s.repo.Querier(ctx), &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list document packages: %w", err)
	}

	out := make([]docdirect.Package, 0, len(rows))
	for _, row := range rows {
		if row.FromRoutePoint == nil || row.ToRoutePoint == nil {
			// A package without a resolved load/unload pair cannot form a
			// direction; skip it (the route may be partial).
			continue
		}

		out = append(out, docdirect.Package{
			ID:       row.PackageID,
			TareID:   row.TareID,
			TareType: row.TareType,
			Action:   docdirect.PackageActionLoad,
			From:     *row.FromRoutePoint,
			To:       *row.ToRoutePoint,
		})
	}

	return out, nil
}