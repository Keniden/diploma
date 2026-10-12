package pgdriver

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/samber/lo"

	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/queries/driver/shipment/readmodels"
)

func (s *Store) GetDriverShipmentsByStatuses(
	ctx context.Context,
	id values.DriverID,
	statuses ...values.ShipmentStatus,
) ([]readmodels.Shipment, error) {
	query, args, err := s.Builder.
		Select(
			"shipment_id",
			"shipment_short_id",
			"shipment_status",
			"driver_id",
			"supplier_id",
			"car_license_plate",
			"closed_at",
			"created_at",
			"updated_at",
			"deleted_at",
		).
		From("lmt.shipment").
		Where(squirrel.Eq{
			"driver_id":       id,
			"shipment_status": statuses,
		}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed when making query to get shipment: %w", err)
	}

	var records []shipmentRecord
	if err = pgxscan.Select(ctx, s.Querier(ctx), &records, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}

	return lo.Map(records, func(r shipmentRecord, _ int) readmodels.Shipment {
		return mapRecordToShipment(r)
	}), nil
}

func (s *Store) GetShipmentByID(
	ctx context.Context,
	id uuid.UUID,
) (readmodels.Shipment, error) {
	query, args, err := s.Builder.
		Select(
			"shipment_id",
			"shipment_short_id",
			"shipment_status",
			"driver_id",
			"supplier_id",
			"car_license_plate",
			"closed_at",
			"created_at",
			"updated_at",
			"deleted_at",
		).
		From("lmt.shipment").
		Where(squirrel.Eq{
			"shipment_id": id,
		}).ToSql()
	if err != nil {
		return readmodels.Shipment{}, fmt.Errorf("failed when making query to get shipment: %w", err)
	}

	var record shipmentRecord
	if err = pgxscan.Get(ctx, s.Querier(ctx), &record, query, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return readmodels.Shipment{}, notFoundError("shipment not found")
		}
		return readmodels.Shipment{}, fmt.Errorf("failed to get shipment: %w", err)
	}

	return mapRecordToShipment(record), nil
}
