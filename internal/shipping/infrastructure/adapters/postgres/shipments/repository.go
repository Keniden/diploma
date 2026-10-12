package shipments

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

var _ ports.ShipmentRepository = (*ShipmentRepository)(nil)
var _ ports.Transactional = (*ShipmentRepository)(nil)

type ShipmentRepository struct {
	*database.Repository
}

func NewRepository(repo *database.Repository) *ShipmentRepository {
	return &ShipmentRepository{
		Repository: repo,
	}
}

// GetShipmentByID implements [ports.ShipmentRepository] and partially [policies.ShipmentPolicyRepository].
func (r *ShipmentRepository) GetShipmentByID(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error) {
	query, args, err := r.Builder.
		Select(
			"shipment_id",
			"shipment_short_id",
			"shipment_status",
			"driver_id",
			"supplier_id",
			"car_license_plate",
			"deleted_at",
		).
		From("lmt.shipment").
		Where(sq.Eq{
			"shipment_id": id,
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build shipment query: %w", err)
	}

	var rec shipmentRecord
	if err := pgxscan.Get(ctx, r.Querier(ctx), &rec, query, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get shipment %s: %w", id, ports.ErrShipmentNotFound)
		}
		return nil, fmt.Errorf("get shipment: %w", err)
	}

	if rec.DeletedAt != nil {
		return nil, fmt.Errorf("get shipment %s: %w", id, ports.ErrShipmentNotFound)
	}

	return shipmentFromRecord(rec), nil
}

// SaveShipment implements [ports.ShipmentRepository].
func (r *ShipmentRepository) SaveShipment(ctx context.Context, shipment *entities.Shipment) error {
	var licensePlate *string
	if shipment.LicensePlate != "" {
		v := string(shipment.LicensePlate)
		licensePlate = &v
	}

	query, args, err := r.Builder.
		Insert("lmt.shipment").
		SetMap(map[string]any{
			"shipment_id":       shipment.ID,
			"shipment_short_id": int64(shipment.ShortID),
			"shipment_status":   shipment.Status,
			"driver_id":         shipment.DriverID,
			"supplier_id":       uint64(shipment.SupplierID),
			"car_license_plate": licensePlate,
		}).
		Suffix(`
ON CONFLICT (shipment_id) DO UPDATE SET
	shipment_short_id = EXCLUDED.shipment_short_id,
	shipment_status = EXCLUDED.shipment_status,
	driver_id = EXCLUDED.driver_id,
	supplier_id = EXCLUDED.supplier_id,
	car_license_plate = EXCLUDED.car_license_plate,
	updated_at = NOW(),
	deleted_at = NULL`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build save shipment query: %w", err)
	}

	if _, err := r.Querier(ctx).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("save shipment: %w", err)
	}

	return nil
}

// GetShipmentCountByDriverID partially implements [policies.ShipmentPolicyRepository]
func (r *ShipmentRepository) GetShipmentCountByDriverID(
	ctx context.Context,
	driverID values.DriverID,
	statuses ...values.ShipmentStatus,
) (int, error) {
	query := r.Builder.
		Select("COUNT(*)").
		From("lmt.shipment").
		Where(sq.Eq{"driver_id": driverID}).
		Where(sq.Eq{"deleted_at": nil})
	if len(statuses) > 0 {
		query = query.Where(sq.Eq{"shipment_status": statuses})
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build shipment count query: %w", err)
	}

	var count int
	if err := pgxscan.Get(ctx, r.Querier(ctx), &count, sql, args...); err != nil {
		return 0, fmt.Errorf("get shipment count: %w", err)
	}

	return count, nil
}
