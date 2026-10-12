package reserves

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"

	"last-mile-task/internal/platform/database"
	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
)

var _ ports.ReserveRepository = (*ReserveRepository)(nil)

type ReserveRepository struct {
	*database.Repository
}

func NewRepository(repo *database.Repository) *ReserveRepository {
	return &ReserveRepository{
		Repository: repo,
	}
}

var reserveSelectColumns = []string{
	"reserve_id",
	"route_point_id",
	"shipment_id",
	"status",
	"task_id",
	"scanned_at",
	"gate_id",
	"gate_name",
	"expires_at",
}

// GetReserveByID implements [ports.ReserveRepository].
func (r *ReserveRepository) GetReserveByID(ctx context.Context, id values.ReserveID) (*entities.Reserve, error) {
	query, args, err := r.Builder.
		Select(reserveSelectColumns...).
		From("lmt.reserve").
		Where(sq.Eq{"reserve_id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build reserve query: %w", err)
	}

	var rec reserveRecord
	if err := pgxscan.Get(ctx, r.Querier(ctx), &rec, query, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get reserve by id %s: %w", id, ports.ErrReserveNotFound)
		}
		return nil, fmt.Errorf("get reserve: %w", err)
	}

	return reserveFromRecord(rec), nil
}

// GetReserveByShipmentID implements [ports.ReserveRepository].
func (r *ReserveRepository) GetReserveByShipmentID(ctx context.Context, shipmentID values.ShipmentID) (*entities.Reserve, error) {
	query, args, err := r.Builder.
		Select(reserveSelectColumns...).
		From("lmt.reserve").
		Where(sq.Eq{"shipment_id": shipmentID}).
		Where(sq.NotEq{"status": []int16{
			int16(values.ReserveStatusExpiredWaitingApproval),
			int16(values.ReserveStatusExpiredActive),
			int16(values.ReserveStatusReleased),
			int16(values.ReserveStatusCanceled),
		}}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build reserve query: %w", err)
	}

	var rec reserveRecord
	if err := pgxscan.Get(ctx, r.Querier(ctx), &rec, query, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get reserve by shipment %s: %w", shipmentID, ports.ErrReserveNotFound)
		}
		return nil, fmt.Errorf("get reserve: %w", err)
	}

	return reserveFromRecord(rec), nil
}

// GetReserveByRoutePointID implements [ports.ReserveRepository].
func (r *ReserveRepository) GetReserveByRoutePointID(ctx context.Context, routePointID values.RoutePointID) (*entities.Reserve, error) {
	query, args, err := r.Builder.
		Select(reserveSelectColumns...).
		From("lmt.reserve").
		Where(sq.Eq{"route_point_id": routePointID}).
		Where(sq.NotEq{"status": []int16{
			int16(values.ReserveStatusExpiredWaitingApproval),
			int16(values.ReserveStatusExpiredActive),
			int16(values.ReserveStatusReleased),
			int16(values.ReserveStatusCanceled),
		}}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build reserve query: %w", err)
	}

	var rec reserveRecord
	if err := pgxscan.Get(ctx, r.Querier(ctx), &rec, query, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get reserve by route point %s: %w", routePointID, ports.ErrReserveNotFound)
		}
		return nil, fmt.Errorf("get reserve: %w", err)
	}

	return reserveFromRecord(rec), nil
}

// ListExpiredWaitingApprovalReserves implements [ports.ReserveRepository].
func (r *ReserveRepository) ListExpiredWaitingApprovalReserves(ctx context.Context, limit int) ([]*entities.Reserve, error) {
	query, args, err := r.Builder.
		Select(reserveSelectColumns...).
		From("lmt.reserve").
		Where(sq.Eq{"status": int16(values.ReserveStatusWaitingApproval)}).
		Where(sq.NotEq{"expires_at": nil}).
		Where("expires_at < NOW()").
		OrderBy("expires_at").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build expired reserves query: %w", err)
	}

	var records []reserveRecord
	if err := pgxscan.Select(ctx, r.Querier(ctx), &records, query, args...); err != nil {
		return nil, fmt.Errorf("list expired waiting approval reserves: %w", err)
	}

	reserves := make([]*entities.Reserve, 0, len(records))
	for _, rec := range records {
		reserves = append(reserves, reserveFromRecord(rec))
	}

	return reserves, nil
}

// ListExpiredActiveReserves implements [ports.ReserveRepository].
func (r *ReserveRepository) ListExpiredActiveReserves(ctx context.Context, limit int) ([]*entities.Reserve, error) {
	query, args, err := r.Builder.
		Select(reserveSelectColumns...).
		From("lmt.reserve").
		Where(sq.Eq{"status": int16(values.ReserveStatusActive)}).
		Where(sq.NotEq{"expires_at": nil}).
		Where("expires_at < NOW()").
		OrderBy("expires_at").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build expired active reserves query: %w", err)
	}

	var records []reserveRecord
	if err := pgxscan.Select(ctx, r.Querier(ctx), &records, query, args...); err != nil {
		return nil, fmt.Errorf("list expired active reserves: %w", err)
	}

	reserves := make([]*entities.Reserve, 0, len(records))
	for _, rec := range records {
		reserves = append(reserves, reserveFromRecord(rec))
	}

	return reserves, nil
}

// SaveReserve implements [ports.ReserveRepository].
func (r *ReserveRepository) SaveReserve(ctx context.Context, reserve *entities.Reserve) error {
	status, err := reserveStatusToDB(reserve.Status)
	if err != nil {
		return err
	}

	query, args, err := r.Builder.
		Insert("lmt.reserve").
		SetMap(map[string]any{
			"reserve_id":     reserve.ID,
			"route_point_id": reserve.RoutePointID,
			"shipment_id":    reserve.ShipmentID,
			"status":         status,
			"task_id":        reserve.Task.ID,
			"gate_id":        reserve.Gate.ID,
			"gate_name":      reserve.Gate.Name,
			"scanned_at":     nullableTimeToDB(reserve.ScannedAt),
			"expires_at":     nullableTimeToDB(reserve.ExpiresAt),
		}).
		Suffix(`
ON CONFLICT (reserve_id) DO UPDATE SET
	status = EXCLUDED.status,
	scanned_at = EXCLUDED.scanned_at,
	task_id = EXCLUDED.task_id,
	gate_id = EXCLUDED.gate_id,
	gate_name = EXCLUDED.gate_name,
	expires_at = EXCLUDED.expires_at,
	updated_at = NOW()`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build save reserve query: %w", err)
	}

	if _, err := r.Querier(ctx).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("save reserve: %w", err)
	}

	return nil
}

// GetReserveCountByShipmentID partially implements [policies.ShipmentPolicyRepository]
func (r *ReserveRepository) GetReserveCountByShipmentID(
	ctx context.Context,
	shipmentID values.ShipmentID,
	statuses ...values.ReserveStatus,
) (int, error) {
	dbStatuses := make([]int16, 0, len(statuses))
	for _, st := range statuses {
		s, err := reserveStatusToDB(st)
		if err != nil {
			return 0, err
		}
		dbStatuses = append(dbStatuses, s)
	}

	query := r.Builder.
		Select("COUNT(*)").
		From("lmt.reserve").
		Where(sq.Eq{"shipment_id": shipmentID})
	if len(dbStatuses) > 0 {
		query = query.Where(sq.Eq{"status": dbStatuses})
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build reserve count query: %w", err)
	}

	var count int
	if err := pgxscan.Get(ctx, r.Querier(ctx), &count, sql, args...); err != nil {
		return 0, fmt.Errorf("get reserve count: %w", err)
	}

	return count, nil
}
