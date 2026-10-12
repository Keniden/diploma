// Package demand plans packages (demand) onto a shipment from the route:
// forward packages are loaded at the source and unloaded at the route points,
// return packages travel backwards from the mid points to the target.
package demand

import (
	"context"
	"fmt"

	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
)

// DemandProvider resolves the package demand for a route.
//
// TODO(LASTMILE-111): the real provider is backed by the cargo-core client
// (ListPackagesToLoad) and splits tare stickers; the skeleton adapter returns
// no demand until the clients are ported.
type DemandProvider interface {
	GetPackageDemandByRoute(ctx context.Context, route *entities.Route) ([]values.PackageDemand, error)
}

// DemandService plans the package demand onto a shipment: it converts the
// route into the set of packages and persists them.
type DemandService struct {
	routes         ports.RouteRepository
	packages       ports.PackageRepository
	demandProvider DemandProvider
	tx             ports.Transactional
}

// NewDemandService builds the demand service.
func NewDemandService(
	routes ports.RouteRepository,
	packages ports.PackageRepository,
	provider DemandProvider,
	tx ports.Transactional,
) *DemandService {
	return &DemandService{
		routes:         routes,
		packages:       packages,
		demandProvider: provider,
		tx:             tx,
	}
}

// PlanPackagesToShipmentByID resolves the demand for the shipment route and
// persists the resulting packages inside one transaction.
func (s *DemandService) PlanPackagesToShipmentByID(
	ctx context.Context,
	id values.ShipmentID,
) error {
	route, err := s.routes.GetRouteByShipmentID(ctx, id)
	if err != nil {
		return fmt.Errorf("get route by shipment id: %w", err)
	}

	demands, err := s.demandProvider.GetPackageDemandByRoute(ctx, route)
	if err != nil {
		return fmt.Errorf("get package demand by route: %w", err)
	}

	pkgs := entities.NewPackagesFromDemands(demands)

	return s.tx.WithTx(ctx, func(txCtx context.Context) error {
		for _, pkg := range pkgs {
			pkg.AssignToShipment(id)

			if err := s.packages.SavePackage(txCtx, pkg); err != nil {
				return fmt.Errorf("save package: %w", err)
			}
		}

		return nil
	})
}
