package packages

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

var _ ports.PackageRepository = (*PackageRepository)(nil)

type PackageRepository struct {
	*database.Repository
}

func NewRepository(repo *database.Repository) *PackageRepository {
	return &PackageRepository{
		Repository: repo,
	}
}

func (r *PackageRepository) loadPackageActions(
	ctx context.Context,
	pkgs []*entities.Package,
) error {
	if len(pkgs) == 0 {
		return nil
	}

	pkgByKey := make(map[string]*entities.Package, len(pkgs))
	pkgIDs := make([]values.PackageID, 0, len(pkgs))
	shipmentIDs := make([]values.ShipmentID, 0, len(pkgs))

	for _, pkg := range pkgs {
		key := pkg.ID.String() + ":" + pkg.ShipmentID.String()
		pkgByKey[key] = pkg
		pkgIDs = append(pkgIDs, pkg.ID)
		shipmentIDs = append(shipmentIDs, pkg.ShipmentID)
	}

	query, args, err := r.Builder.
		Select(
			"package_id",
			"route_point_id",
			"shipment_id",
			"action",
			"action_status",
			"deleted_at",
			"action_status_updated_at",
		).
		From("lmt.package_route_point").
		Where(sq.Eq{
			"package_id":  pkgIDs,
			"shipment_id": shipmentIDs,
		}).
		Where(sq.Eq{"deleted_at": nil}).
		Where(sq.NotEq{"action_status": int(values.RoutePointPackageStatusCanceled)}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build package actions query: %w", err)
	}

	var recs []packageRoutePointRecord
	if err := pgxscan.Select(ctx, r.Querier(ctx), &recs, query, args...); err != nil {
		return fmt.Errorf("get package actions: %w", err)
	}

	for _, rec := range recs {
		key := rec.PackageID.String() + ":" + rec.ShipmentID.String()
		pkg, ok := pkgByKey[key]
		if !ok {
			continue
		}

		pkg.Actions = append(pkg.Actions, &entities.PackageAction{
			RoutePointID:    rec.RoutePointID,
			Type:            packageActionTypeFromDB(rec.Action),
			Status:          packageActionStatusFromDB(rec.ActionStatus),
			StatusUpdatedAt: rec.ActionStatusUpdatedAt,
		})
	}

	return nil
}

// GetPackageByID implements [ports.PackageRepository].
func (r *PackageRepository) GetPackageByID(
	ctx context.Context,
	shipmentID values.ShipmentID,
	id values.PackageID,
) (*entities.Package, error) {
	query, args, err := r.Builder.
		Select(
			"package_id",
			"shipment_id",
			"package_status",
			"is_return",
			"tare_id",
			"tare_type",
			"deleted_at",
		).
		From("lmt.package").
		Where(sq.Eq{
			"package_id":  id,
			"shipment_id": shipmentID,
		}).
		Where(sq.Eq{"deleted_at": nil}).
		Where(sq.NotEq{"package_status": int(values.PackageStatusCanceled)}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build package query: %w", err)
	}

	var rec packageRecord
	if err := pgxscan.Get(ctx, r.Querier(ctx), &rec, query, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get package %s: %w", id, ports.ErrPackageNotFound)
		}
		return nil, fmt.Errorf("get package: %w", err)
	}

	pkg := packageFromRecord(rec)
	if err := r.loadPackageActions(ctx, []*entities.Package{pkg}); err != nil {
		return nil, err
	}

	return pkg, nil
}

// GetPackagesByIDs implements [ports.PackageRepository].
func (r *PackageRepository) GetPackagesByIDs(
	ctx context.Context,
	shipmentID values.ShipmentID,
	ids ...values.PackageID,
) ([]*entities.Package, error) {
	if len(ids) == 0 {
		return []*entities.Package{}, nil
	}

	query, args, err := r.Builder.
		Select(
			"package_id",
			"shipment_id",
			"package_status",
			"is_return",
			"tare_id",
			"tare_type",
			"deleted_at",
		).
		From("lmt.package").
		Where(sq.And{
			sq.Eq{"package_id": ids},
			sq.Eq{"deleted_at": nil},
			sq.Eq{"shipment_id": shipmentID},
			sq.NotEq{"package_status": int(values.PackageStatusCanceled)},
		}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build packages by ids query: %w", err)
	}

	var recs []packageRecord
	if err := pgxscan.Select(ctx, r.Querier(ctx), &recs, query, args...); err != nil {
		return nil, fmt.Errorf("get packages by ids: %w", err)
	}

	out := make([]*entities.Package, 0, len(recs))
	for _, rec := range recs {
		out = append(out, packageFromRecord(rec))
	}

	if err := r.loadPackageActions(ctx, out); err != nil {
		return nil, err
	}

	return out, nil
}

// GetPackagesByShipmentID implements [ports.PackageRepository].
func (r *PackageRepository) GetPackagesByShipmentID(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error) {
	query, args, err := r.Builder.
		Select(
			"package_id",
			"shipment_id",
			"package_status",
			"is_return",
			"tare_id",
			"tare_type",
			"deleted_at",
		).
		From("lmt.package").
		Where(sq.Eq{"shipment_id": shipmentID}).
		Where(sq.Eq{"deleted_at": nil}).
		Where(sq.NotEq{"package_status": int(values.PackageStatusCanceled)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build packages by shipment query: %w", err)
	}

	var recs []packageRecord
	if err := pgxscan.Select(ctx, r.Querier(ctx), &recs, query, args...); err != nil {
		return nil, fmt.Errorf("get packages by shipment: %w", err)
	}

	out := make([]*entities.Package, 0, len(recs))
	for _, rec := range recs {
		out = append(out, packageFromRecord(rec))
	}

	if err := r.loadPackageActions(ctx, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *PackageRepository) GetLoadedPackagesByShipmentID(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error) {
	query, args, err := r.Builder.
		Select(
			"package_id",
			"shipment_id",
			"package_status",
			"is_return",
			"tare_id",
			"tare_type",
			"deleted_at",
		).
		From("lmt.package").
		Where(sq.Eq{"shipment_id": shipmentID}).
		Where(sq.Eq{"deleted_at": nil}).
		Where(sq.Eq{"package_status": []int{
			dbPackageStatusLoaded,
			dbPackageStatusInTransit,
			dbPackageStatusUnloading,
		}}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build loaded packages by shipment query: %w", err)
	}

	var recs []packageRecord
	if err := pgxscan.Select(ctx, r.Querier(ctx), &recs, query, args...); err != nil {
		return nil, fmt.Errorf("get loaded packages by shipment: %w", err)
	}

	out := make([]*entities.Package, 0, len(recs))
	for _, rec := range recs {
		out = append(out, packageFromRecord(rec))
	}

	if err := r.loadPackageActions(ctx, out); err != nil {
		return nil, err
	}

	return out, nil
}

// SavePackage implements [ports.PackageRepository].
func (r *PackageRepository) SavePackage(ctx context.Context, pkg *entities.Package) error {
	_, alreadyInTx := r.Querier(ctx).(pgx.Tx)

	txCtx := ctx
	if !alreadyInTx {
		var err error
		txCtx, err = r.BeginTx(ctx)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer func() { _ = r.Rollback(txCtx) }()
	}

	status := packageStatusToDB(pkg.Status)

	query, args, err := r.Builder.
		Insert("lmt.package").
		SetMap(map[string]any{
			"package_id":     pkg.ID,
			"shipment_id":    pkg.ShipmentID,
			"package_status": status,
			"is_return":      pkg.IsReturn,
			"tare_id":        pkg.TareID,
			"tare_type":      pkg.TareType,
		}).
		Suffix(`
ON CONFLICT (package_id, shipment_id) DO UPDATE SET
	package_status = EXCLUDED.package_status,
	is_return = EXCLUDED.is_return,
	tare_id = EXCLUDED.tare_id,
	tare_type = EXCLUDED.tare_type,
	updated_at = NOW(),
	deleted_at = NULL`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build save package query: %w", err)
	}

	if _, err = r.Querier(txCtx).Exec(txCtx, query, args...); err != nil {
		return fmt.Errorf("save package: %w", err)
	}

	now := time.Now()

	delActionsQuery, delActionsArgs, err := r.Builder.
		Update("lmt.package_route_point").
		Set("deleted_at", now).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"package_id":  pkg.ID,
			"shipment_id": pkg.ShipmentID,
			"deleted_at":  nil,
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete package actions query: %w", err)
	}
	if _, err := r.Querier(txCtx).Exec(txCtx, delActionsQuery, delActionsArgs...); err != nil {
		return fmt.Errorf("soft delete package actions: %w", err)
	}

	for _, action := range pkg.Actions {
		actionDB := packageActionTypeToDB(action.Type)
		statusDB := packageActionStatusToDB(action.Status)

		insQuery, insArgs, err := r.Builder.
			Insert("lmt.package_route_point").
			SetMap(map[string]any{
				"package_id":               pkg.ID,
				"route_point_id":           action.RoutePointID,
				"shipment_id":              pkg.ShipmentID,
				"action":                   actionDB,
				"action_status":            statusDB,
				"action_status_updated_at": action.StatusUpdatedAt,
				"deleted_at":               nil,
			}).
			Suffix(`
ON CONFLICT (package_id, route_point_id, shipment_id) DO UPDATE SET
	action = EXCLUDED.action,
	action_status = EXCLUDED.action_status,
	action_status_updated_at = EXCLUDED.action_status_updated_at,
	updated_at = NOW(),
	deleted_at = NULL`).
			ToSql()
		if err != nil {
			return fmt.Errorf("build upsert package action query: %w", err)
		}

		if _, err := r.Querier(txCtx).Exec(txCtx, insQuery, insArgs...); err != nil {
			return fmt.Errorf("upsert package action: %w", err)
		}
	}

	if !alreadyInTx {
		if err := r.Commit(txCtx); err != nil {
			return fmt.Errorf("commit tx: %w", err)
		}
	}

	return nil
}

// SavePackageScan updates package status and a single scanned action without touching other actions.
func (r *PackageRepository) SavePackageScan(
	ctx context.Context,
	pkg *entities.Package,
	action *entities.PackageAction,
) error {
	_, alreadyInTx := r.Querier(ctx).(pgx.Tx)

	txCtx := ctx
	if !alreadyInTx {
		var err error
		txCtx, err = r.BeginTx(ctx)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer func() { _ = r.Rollback(txCtx) }()
	}

	status := packageStatusToDB(pkg.Status)

	pkgQuery, pkgArgs, err := r.Builder.
		Update("lmt.package").
		Set("package_status", status).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"package_id":  pkg.ID,
			"shipment_id": pkg.ShipmentID,
			"deleted_at":  nil,
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build save package scan query: %w", err)
	}

	if _, err = r.Querier(txCtx).Exec(txCtx, pkgQuery, pkgArgs...); err != nil {
		return fmt.Errorf("save package scan status: %w", err)
	}

	actionDB := packageActionTypeToDB(action.Type)
	statusDB := packageActionStatusToDB(action.Status)

	actionQuery, actionArgs, err := r.Builder.
		Update("lmt.package_route_point").
		Set("action_status", statusDB).
		Set("action_status_updated_at", action.StatusUpdatedAt).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"package_id":     pkg.ID,
			"shipment_id":    pkg.ShipmentID,
			"route_point_id": action.RoutePointID,
			"action":         actionDB,
			"deleted_at":     nil,
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build save package scan action query: %w", err)
	}

	if _, err = r.Querier(txCtx).Exec(txCtx, actionQuery, actionArgs...); err != nil {
		return fmt.Errorf("save package scan action: %w", err)
	}

	if !alreadyInTx {
		if err := r.Commit(txCtx); err != nil {
			return fmt.Errorf("commit tx: %w", err)
		}
	}

	return nil
}

func (r *PackageRepository) DeletePackage(
	ctx context.Context,
	shipmentID values.ShipmentID,
	id values.PackageID,
) error {
	now := time.Now()

	query, args, err := r.Builder.
		Update("lmt.package").
		Set("deleted_at", now).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"package_id":  id,
			"shipment_id": shipmentID,
			"deleted_at":  nil,
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete package query: %w", err)
	}

	res, err := r.Querier(ctx).Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete package: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("package not found: %s", id)
	}

	routePointQuery, routePointArgs, err := r.Builder.
		Update("lmt.package_route_point").
		Set("deleted_at", now).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"package_id":  id,
			"shipment_id": shipmentID,
			"deleted_at":  nil,
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete package route points query: %w", err)
	}

	if _, err := r.Querier(ctx).Exec(ctx, routePointQuery, routePointArgs...); err != nil {
		return fmt.Errorf("delete package route points: %w", err)
	}

	return nil
}

// GetPackageCountByShipmentID partially implements [policies.ShipmentPolicyRepository]
func (r *PackageRepository) GetPackageCountByShipmentID(
	ctx context.Context,
	shipmentID values.ShipmentID,
	statuses ...values.PackageStatus,
) (int, error) {
	dbStatuses := make([]int, 0, len(statuses))
	for _, st := range statuses {
		dbStatuses = append(dbStatuses, packageStatusToDB(st))
	}

	query := r.Builder.
		Select("COUNT(*)").
		From("lmt.package").
		Where(sq.Eq{"shipment_id": shipmentID}).
		Where(sq.Eq{"deleted_at": nil})
	if len(dbStatuses) > 0 {
		query = query.Where(sq.Eq{"package_status": dbStatuses})
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build package count query: %w", err)
	}

	var count int
	if err := pgxscan.Get(ctx, r.Querier(ctx), &count, sql, args...); err != nil {
		return 0, fmt.Errorf("get package count: %w", err)
	}

	return count, nil
}
