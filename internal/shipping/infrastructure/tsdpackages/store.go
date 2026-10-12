// Package tsdpackages is the read adapter serving the TSD list-packages-to-scan
// query over the lmt tables. It implements the tsd.PackagesToScanLister port.
//
// TODO(LASTMILE-008): the slice contract is Empty→Empty, so the list is not
// surfaced on the wire yet; the adapter keeps the read model ready for the real
// TSD proto once it lands.
package tsdpackages

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"

	"last-mile-task/internal/application/tsd"
	"last-mile-task/internal/platform/database"
	"last-mile-task/internal/shipping/domain/values"
)

// Store lists packages awaiting a scan at a route point.
type Store struct {
	repo *database.Repository
}

// NewStore builds the SQL-backed packages-to-scan store.
func NewStore(repo *database.Repository) *Store {
	return &Store{repo: repo}
}

// ListPackagesToScan returns the READY_TO_LOAD / IN_TRANSIT packages of the
// shipment that are awaiting a scan at the given route point.
func (s *Store) ListPackagesToScan(
	ctx context.Context,
	shipmentID values.ShipmentID,
	routePointID values.RoutePointID,
) ([]tsd.PackageToScan, error) {
	query, args, err := s.repo.Builder.
		Select(
			"p.package_id",
			"p.shipment_id",
			"p.package_status",
			"p.tare_id",
			"p.tare_type",
		).
		From("lmt.package_route_point prp").
		Join("lmt.package p ON p.package_id = prp.package_id").
		Where(squirrel.And{
			squirrel.Eq{"prp.route_point_id": routePointID},
			squirrel.Eq{"p.shipment_id": shipmentID},
			squirrel.And{
				squirrel.Or{
					squirrel.Eq{"p.package_status": int(values.PackageStatusReadyToLoad)},
					squirrel.Eq{"p.package_status": int(values.PackageStatusInTransit)},
				},
			},
			squirrel.Eq{"p.deleted_at": nil},
			squirrel.Eq{"prp.deleted_at": nil},
			squirrel.NotEq{"prp.action_status": int(values.RoutePointPackageStatusCanceled)},
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list packages to scan: %w", err)
	}

	var records []record
	if err := pgxscan.Select(ctx, s.repo.Querier(ctx), &records, query, args...); err != nil {
		return nil, fmt.Errorf("list packages to scan: %w", err)
	}

	packages := make([]tsd.PackageToScan, 0, len(records))
	for _, r := range records {
		packages = append(packages, tsd.PackageToScan{
			ID:         values.PackageID(r.PackageID),
			ShipmentID: values.ShipmentID(r.ShipmentID),
			Status:     values.PackageStatus(r.Status),
			TareID:     values.TareID(r.TareID),
			TareType:   values.TareType(r.TareType),
		})
	}

	return packages, nil
}

type record struct {
	PackageID  uuid.UUID `db:"package_id"`
	ShipmentID uuid.UUID `db:"shipment_id"`
	Status     int       `db:"package_status"`
	TareID     int64     `db:"tare_id"`
	TareType   string    `db:"tare_type"`
}
