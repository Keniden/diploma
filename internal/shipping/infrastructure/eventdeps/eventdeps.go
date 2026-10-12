// Package eventdeps provides the slice-scoped implementations of the narrow
// stores/offices/users interfaces required by the outbox event handlers.
//
// The real implementations are spread across queries (portal read models) and
// external clients (office, users); the skeleton serves the vertical slice
// with direct SQL plus deterministic mock data for external services.
package eventdeps

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"

	"last-mile-task/internal/platform/database"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/dto"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/warehouse"
)

// Store serves the event-handler read queries over lmt tables.
type Store struct {
	repo *database.Repository
}

// NewStore builds the SQL-backed event-handler store.
func NewStore(repo *database.Repository) *Store {
	return &Store{repo: repo}
}

// ShipmentByID loads a shipment snapshot for the WMS payload.
func (s *Store) ShipmentByID(ctx context.Context, shipmentID values.ShipmentID) (dto.Shipment, error) {
	row := s.repo.Querier(ctx).QueryRow(ctx,
		`SELECT shipment_id, driver_id, car_license_plate
		   FROM lmt.shipment
		  WHERE shipment_id = $1 AND deleted_at IS NULL`, shipmentID,
	)

	var snapshot dto.Shipment

	// A freshly created shipment may not have a license plate yet (NULL in
	// lmt.shipment.car_license_plate); scan it into a nullable string and
	// coerce only when present.
	var plate *string
	if err := row.Scan(&snapshot.ID, &snapshot.DriverID, &plate); err != nil {
		return dto.Shipment{}, fmt.Errorf("scan shipment: %w", err)
	}
	if plate != nil {
		snapshot.CarLicensePlate = values.LicensePlate(*plate)
	}

	return snapshot, nil
}

// pointUUIDNamespace namespaces the deterministic mock point UUIDs.
var pointUUIDNamespace = uuid.MustParse("9b5a0000-0000-4000-a000-000000000001")

// PackageDestinationRoutePoint loads the target route point of a package.
func (s *Store) PackageDestinationRoutePoint(
	ctx context.Context, packageID values.PackageID,
) (dto.RoutePoint, error) {
	row := s.repo.Querier(ctx).QueryRow(ctx,
		`SELECT rp.route_point_id, rp.routecar_point_id
		   FROM lmt.package_route_point prp
		   JOIN lmt.route_point rp ON rp.route_point_id = prp.route_point_id
		  WHERE prp.package_id = $1
		    AND prp.action = 2 -- UNLOAD
		  LIMIT 1`, packageID,
	)
	var point dto.RoutePoint
	if err := row.Scan(&point.ID, &point.PointID); err != nil {
		return dto.RoutePoint{}, fmt.Errorf("scan route point: %w", err)
	}
	point.PointUUID = deterministicPointUUID(point.PointID)
	return point, nil
}

// RoutePointByID loads a route point by its id.
func (s *Store) RoutePointByID(ctx context.Context, routePointID values.RoutePointID) (dto.RoutePoint, error) {
	row := s.repo.Querier(ctx).QueryRow(ctx,
		`SELECT route_point_id, routecar_point_id
		   FROM lmt.route_point
		  WHERE route_point_id = $1 AND deleted_at IS NULL`, routePointID,
	)
	var point dto.RoutePoint
	if err := row.Scan(&point.ID, &point.PointID); err != nil {
		return dto.RoutePoint{}, fmt.Errorf("scan route point: %w", err)
	}
	return point, nil
}

// PointUUID resolves the external point uuid by its numeric id. The skeleton
// has no office client, so the UUID is deterministic from the point id.
//
// TODO(LASTMILE-106): resolve through the office client (GetPointByID)
// exactly like the real handler (routePointWithPointUUID).
func (s *Store) PointUUID(_ context.Context, pointID values.PointID) (uuid.UUID, error) {
	return deterministicPointUUID(pointID), nil
}

// deterministicPointUUID derives a stable UUID from a numeric point id.
func deterministicPointUUID(pointID values.PointID) uuid.UUID {
	return uuid.NewSHA1(pointUUIDNamespace, []byte(strconv.FormatInt(int64(pointID), 10)))
}

// Offices provides deterministic office data in mock mode (no office client).
//
// TODO(LASTMILE-104): replace with the office client (office_service) once
// ported; the static values keep the WMS payload valid in mock/e2e runs.
type Offices struct{}

// NewOffices builds the static offices provider.
func NewOffices() *Offices { return &Offices{} }

// GetOfficeByPointID returns a deterministic office for any point.
func (o *Offices) GetOfficeByPointID(_ context.Context, pointID int64) (warehouse.Office, error) {
	return warehouse.Office{ID: int32(pointID), CountryID: 643}, nil // RU
}

// Users provides deterministic driver data in mock mode (no users client).
//
// TODO(LASTMILE-105): replace with the users client once ported.
type Users struct{}

// NewUsers builds the static users provider.
func NewUsers() *Users { return &Users{} }

// UserByID returns a deterministic driver.
func (u *Users) UserByID(_ context.Context, _ values.DriverID) (*warehouse.User, error) {
	return &warehouse.User{
		FirstName: "Mock",
		LastName:  "Driver",
		Phone:     "+70000000000",
	}, nil
}