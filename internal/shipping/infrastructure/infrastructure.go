// Package infrastructure is the composition root of the shipping context: it
// aggregates persistence adapters into the repository set used by the
// application layer and holds slice-scoped stubs for external integrations.
package infrastructure

import (
	"context"

	"last-mile-task/internal/platform/database"
	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure/adapters/postgres/packages"
	"last-mile-task/internal/shipping/infrastructure/adapters/postgres/reserves"
	"last-mile-task/internal/shipping/infrastructure/adapters/postgres/routes"
	"last-mile-task/internal/shipping/infrastructure/adapters/postgres/shipments"
)

// EntityRepository bundles the write-side aggregate repositories of the
// shipping context (mirrors the real infrastructure.NewEntityRepository).
type EntityRepository struct {
	ShipmentRepository *shipments.ShipmentRepository
	RouteRepository    *routes.RouteRepository
	PackageRepository  *packages.PackageRepository
	ReserveRepository  *reserves.ReserveRepository
}

// NewEntityRepository builds the repository set over a shared database
// repository.
func NewEntityRepository(repo *database.Repository) *EntityRepository {
	return &EntityRepository{
		ShipmentRepository: shipments.NewRepository(repo),
		RouteRepository:    routes.NewRepository(repo),
		PackageRepository:  packages.NewRepository(repo),
		ReserveRepository:  reserves.NewRepository(repo),
	}
}

// PackageProvider is the slice-scoped stand-in for the real package
// provider built on cargo-core + package-content clients.
//
// TODO(LASTMILE-102): replace with the client-backed provider once the
// external clients are ported. Empty results keep the load/unload slice
// (scanner + sync) functional; disabled paths are never exercised by it.
type Provider struct{}

// NewProvider builds the stub provider.
func NewProvider() ports.PackageProvider { return &Provider{} }

// ListReturnPackages returns no return packages (client not ported).
func (p *Provider) ListReturnPackages(
	_ context.Context, _ values.PointID, _ []values.PointID,
) ([]ports.ReturnPackage, error) {
	return nil, nil
}

// ListForwardPackages returns no forward packages (client not ported).
func (p *Provider) ListForwardPackages(
	_ context.Context, _ values.PointID, _ []values.PointID,
) ([]ports.ForwardPackage, error) {
	return nil, nil
}

// GetTareSticker returns an empty sticker (client not ported).
func (p *Provider) GetTareSticker(_ context.Context, _ values.PackageID) (string, error) {
	return "", nil
}

// AddressResolver resolves point addresses for read models. The skeleton has
// no office client, so addresses resolve to empty strings.
//
// TODO(LASTMILE-103): replace with the office-backed resolver.
type AddressResolver struct{}

// NewAddressResolver builds the stub resolver.
func NewAddressResolver() *AddressResolver { return &AddressResolver{} }

// AddressesByPointIDs returns empty addresses for the requested points.
func (a *AddressResolver) AddressesByPointIDs(_ context.Context, pointIDs []int64) (map[int64]string, error) {
	return map[int64]string{}, nil
}