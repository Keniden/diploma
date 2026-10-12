package pgdriver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/queries/driver/shipment/readmodels"
)

func (s *Store) GetActiveShipmentReserveByShipmentID(
	ctx context.Context,
	shipmentID uuid.UUID,
) (*readmodels.ShipmentReserve, error) {
	query, args, err := s.Builder.
		Select(
			"expires_at",
			"updated_at",
		).
		From("lmt.reserve").
		Where(squirrel.Eq{
			"shipment_id": shipmentID,
		}).
		Where(squirrel.NotEq{"status": []int16{
			int16(values.ReserveStatusExpiredWaitingApproval),
			int16(values.ReserveStatusExpiredActive),
			int16(values.ReserveStatusReleased),
			int16(values.ReserveStatusCanceled),
		}}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build active shipment reserve query: %w", err)
	}

	var expiresAt *time.Time
	var updatedAt time.Time
	if err := s.Querier(ctx).QueryRow(ctx, query, args...).Scan(&expiresAt, &updatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("get active shipment reserve: %w", err)
	}

	return &readmodels.ShipmentReserve{
		ExpiresAt: expiresAt,
		UpdatedAt: updatedAt,
	}, nil
}

func (s *Store) GetActiveShipmentReservesByShipmentIDs(
	ctx context.Context,
	shipmentIDs []uuid.UUID,
) (map[uuid.UUID]readmodels.ShipmentReserve, error) {
	if len(shipmentIDs) == 0 {
		return map[uuid.UUID]readmodels.ShipmentReserve{}, nil
	}

	query, args, err := s.Builder.
		Select(
			"DISTINCT ON (shipment_id) shipment_id",
			"expires_at",
			"updated_at",
		).
		From("lmt.reserve").
		Where(squirrel.Eq{"shipment_id": shipmentIDs}).
		Where(squirrel.NotEq{"status": []int16{
			int16(values.ReserveStatusExpiredWaitingApproval),
			int16(values.ReserveStatusExpiredActive),
			int16(values.ReserveStatusReleased),
			int16(values.ReserveStatusCanceled),
		}}).
		OrderBy("shipment_id", "created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build active shipment reserves query: %w", err)
	}

	rows, err := s.Querier(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get active shipment reserves: %w", err)
	}
	defer rows.Close()

	out := make(map[uuid.UUID]readmodels.ShipmentReserve, len(shipmentIDs))
	for rows.Next() {
		var shipmentID uuid.UUID
		var expiresAt *time.Time
		var updatedAt time.Time
		if err := rows.Scan(&shipmentID, &expiresAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan active shipment reserve: %w", err)
		}

		out[shipmentID] = readmodels.ShipmentReserve{
			ExpiresAt: expiresAt,
			UpdatedAt: updatedAt,
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active shipment reserves: %w", err)
	}

	return out, nil
}
