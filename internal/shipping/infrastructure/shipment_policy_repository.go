package infrastructure

import (
	"context"

	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/policies"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure/adapters/postgres/packages"
	"last-mile-task/internal/shipping/infrastructure/adapters/postgres/reserves"
	"last-mile-task/internal/shipping/infrastructure/adapters/postgres/routes"
	"last-mile-task/internal/shipping/infrastructure/adapters/postgres/shipments"
)

var _ policies.ShipmentPolicyRepository = (*ShipmentPolicyRepository)(nil)

// SourcePackagesProvider reports whether the source point of a route has
// packages to carry (backed by cargo-core in the real repository).
type SourcePackagesProvider interface {
	HasSourcePackages(ctx context.Context, route *entities.Route) (bool, error)
}

// ShipmentPolicyRepository implements [policies.ShipmentPolicyRepository]
// over the shipping write repositories and the source packages provider.
type ShipmentPolicyRepository struct {
	shipments      *shipments.ShipmentRepository
	routes         *routes.RouteRepository
	packages       *packages.PackageRepository
	reserves       *reserves.ReserveRepository
	sourcePackages SourcePackagesProvider
}

// NewShipmentPolicyRepository builds the policy repository adapter.
func NewShipmentPolicyRepository(
	shipments *shipments.ShipmentRepository,
	routes *routes.RouteRepository,
	packages *packages.PackageRepository,
	reserves *reserves.ReserveRepository,
	sourcePackages SourcePackagesProvider,
) *ShipmentPolicyRepository {
	return &ShipmentPolicyRepository{
		shipments:      shipments,
		routes:         routes,
		packages:       packages,
		reserves:       reserves,
		sourcePackages: sourcePackages,
	}
}

// GetShipmentByID implements [policies.ShipmentPolicyRepository].
func (r *ShipmentPolicyRepository) GetShipmentByID(
	ctx context.Context,
	id values.ShipmentID,
) (*entities.Shipment, error) {
	return r.shipments.GetShipmentByID(ctx, id)
}

// GetRouteByShipmentID implements [policies.ShipmentPolicyRepository].
func (r *ShipmentPolicyRepository) GetRouteByShipmentID(
	ctx context.Context,
	id values.ShipmentID,
) (*entities.Route, error) {
	return r.routes.GetRouteByShipmentID(ctx, id)
}

// GetShipmentCountByDriverID implements [policies.ShipmentPolicyRepository].
func (r *ShipmentPolicyRepository) GetShipmentCountByDriverID(
	ctx context.Context,
	driverID values.DriverID,
	statuses ...values.ShipmentStatus,
) (int, error) {
	return r.shipments.GetShipmentCountByDriverID(ctx, driverID, statuses...)
}

// GetPackageCountByShipmentID implements [policies.ShipmentPolicyRepository].
func (r *ShipmentPolicyRepository) GetPackageCountByShipmentID(
	ctx context.Context,
	shipmentID values.ShipmentID,
	statuses ...values.PackageStatus,
) (int, error) {
	return r.packages.GetPackageCountByShipmentID(ctx, shipmentID, statuses...)
}

// GetPackagesByShipmentID implements [policies.ShipmentPolicyRepository].
func (r *ShipmentPolicyRepository) GetPackagesByShipmentID(
	ctx context.Context,
	shipmentID values.ShipmentID,
) ([]*entities.Package, error) {
	return r.packages.GetPackagesByShipmentID(ctx, shipmentID)
}

// GetReserveCountByShipmentID implements [policies.ShipmentPolicyRepository].
func (r *ShipmentPolicyRepository) GetReserveCountByShipmentID(
	ctx context.Context,
	shipmentID values.ShipmentID,
	statuses ...values.ReserveStatus,
) (int, error) {
	return r.reserves.GetReserveCountByShipmentID(ctx, shipmentID, statuses...)
}

// HasSourcePackages implements [policies.ShipmentPolicyRepository].
func (r *ShipmentPolicyRepository) HasSourcePackages(
	ctx context.Context,
	route *entities.Route,
) (bool, error) {
	return r.sourcePackages.HasSourcePackages(ctx, route)
}
