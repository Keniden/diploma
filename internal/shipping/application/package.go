package application

import (
	"context"
	"fmt"

	"last-mile-task/internal/primitives"
	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/values"
)

type PackageService struct {
	packages              ports.PackageRepository
	routes                ports.RouteRepository
	packageEventPublisher ports.PackageEventPublisher
	tx                    ports.Transactional
}

func NewPackageService(
	packages ports.PackageRepository,
	routes ports.RouteRepository,
	tx ports.Transactional,
	packageEventPublisher ports.PackageEventPublisher,
) *PackageService {
	return &PackageService{
		packages:              packages,
		routes:                routes,
		packageEventPublisher: packageEventPublisher,
		tx:                    tx,
	}
}

func (s *PackageService) DeletePackage(ctx context.Context, shipmentID, packageID values.ID) error {
	return s.tx.WithTx(ctx, func(txCtx context.Context) error {
		if _, err := s.packages.GetPackageByID(txCtx, shipmentID, packageID); err != nil {
			return fmt.Errorf("get package by id: %w", err)
		}

		return s.packages.DeletePackage(txCtx, shipmentID, packageID)
	})
}

func (s *PackageService) DeleteNotLoadedPackages(ctx context.Context, shipmentID values.ID) error {
	return s.tx.WithTx(ctx, func(txCtx context.Context) error {
		route, err := s.routes.GetRouteByShipmentID(ctx, shipmentID)
		if err != nil {
			return fmt.Errorf("get route by shipment id: %w", err)
		}

		startPointID := route.Points[0].ID

		pkgs, err := s.packages.GetPackagesByShipmentID(txCtx, shipmentID)
		if err != nil {
			return fmt.Errorf("get package by id: %w", err)
		}

		for _, pkg := range pkgs {
			for _, action := range pkg.Actions {
				if action.RoutePointID == startPointID &&
					action.Status == values.RoutePointPackageStatusTODO {
					err = s.packages.DeletePackage(txCtx, shipmentID, pkg.ID)
					if err != nil {
						return fmt.Errorf("delete package: %w", err)
					}
				}
			}
		}

		return nil
	})
}

type PackageSyncService struct {
	shipments       ports.ShipmentRepository
	routes          ports.RouteRepository
	packages        ports.PackageRepository
	packageProvider ports.PackageProvider
	tx              ports.Transactional
}

func NewPackageSyncService(
	shipments ports.ShipmentRepository,
	routes ports.RouteRepository,
	packages ports.PackageRepository,
	packageProvider ports.PackageProvider,
	tx ports.Transactional,
) *PackageSyncService {
	return &PackageSyncService{
		shipments:       shipments,
		routes:          routes,
		packages:        packages,
		packageProvider: packageProvider,
		tx:              tx,
	}
}

func (s *PackageService) CancelNotLoadedPackages(
	ctx context.Context,
	shipmentID values.ShipmentID,
	routePointID values.RoutePointID,
) error {
	return s.tx.WithTx(ctx, func(txCtx context.Context) error {
		pkgs, err := s.packages.GetPackagesByShipmentID(txCtx, shipmentID)
		if err != nil {
			return fmt.Errorf("get packages by shipment id: %w", err)
		}

		for _, pkg := range filterByRoutePoint(pkgs, routePointID) {
			canceled, err := pkg.CancelLoad()
			if err != nil {
				return fmt.Errorf("cancel package: %w", err)
			}
			if !canceled {
				continue
			}
			if err := s.packages.SavePackage(txCtx, pkg); err != nil {
				return fmt.Errorf("save package: %w", err)
			}
		}

		return nil
	})
}

func (s *PackageService) TransitionLoadedPackagesToInTransit(
	ctx context.Context,
	shipmentID values.ShipmentID,
	routePointID values.RoutePointID,
) error {
	return s.tx.WithTx(ctx, func(txCtx context.Context) error {
		pkgs, err := s.packages.GetPackagesByShipmentID(txCtx, shipmentID)
		if err != nil {
			return fmt.Errorf("get packages by shipment id: %w", err)
		}

		var transitioned []values.PackageID

		for _, pkg := range pkgs {
			if pkg.Status != values.PackageStatusLoaded {
				continue
			}

			if !hasCompletedLoadAtRoutePoint(pkg, routePointID) {
				continue
			}

			if err := pkg.MarkInTransit(); err != nil {
				return fmt.Errorf("mark package in transit: %w", err)
			}

			if err := s.packages.SavePackage(txCtx, pkg); err != nil {
				return fmt.Errorf("save package: %w", err)
			}

			transitioned = append(transitioned, pkg.ID)
		}

		if len(transitioned) == 0 {
			return nil
		}

		if err := s.packageEventPublisher.PublishPackageEvent(txCtx, events.PackageInTransitEvent{
			RoutePointID:   routePointID,
			PackageIDs:     transitioned,
			TransitionedAt: primitives.Now(),
		}); err != nil {
			return fmt.Errorf("publish package in transit event: %w", err)
		}

		return nil
	})
}

func (s *PackageSyncService) SyncPackagesAtRoutePoint(
	ctx context.Context,
	shipmentID values.ShipmentID,
	routePointID values.RoutePointID,
) ([]*entities.Package, error) {
	route, err := s.routes.GetRouteByShipmentID(ctx, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("get route by shipment id: %w", err)
	}

	source, err := route.Source()
	if err != nil {
		return nil, fmt.Errorf("get source: %w", err)
	}

	target, err := route.Target()
	if err != nil {
		return nil, fmt.Errorf("get target: %w", err)
	}

	pkgs, err := s.packages.GetPackagesByShipmentID(ctx, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("get packages by shipment id: %w", err)
	}

	switch routePointID {
	case source.ID:
		unloadPoints, err := route.RoutePointsForUnloading()
		if err != nil {
			return nil, fmt.Errorf("get unload points: %w", err)
		}

		return s.syncStartPoint(ctx, shipmentID, source, unloadPoints, pkgs)
	case target.ID:
		return s.syncEndPoint(ctx, shipmentID, pkgs, routePointID)
	default:
		return s.syncMidPoint(ctx, shipmentID, routePointID, route, target, pkgs)
	}
}

func (s *PackageSyncService) syncEndPoint(
	ctx context.Context,
	shipmentID values.ShipmentID,
	pkgs []*entities.Package,
	routePointID values.RoutePointID,
) ([]*entities.Package, error) {
	if txErr := s.tx.WithTx(ctx, func(txCtx context.Context) error {
		shipment, err := s.shipments.GetShipmentByID(txCtx, shipmentID)
		if err != nil {
			return fmt.Errorf("get shipment by id: %w", err)
		}

		if err := s.shipments.SaveShipment(txCtx, shipment); err != nil {
			return fmt.Errorf("save shipment: %w", err)
		}

		for _, pkg := range pkgs {
			for _, action := range pkg.Actions {
				if action.RoutePointID == routePointID &&
					action.Type == values.RoutePointActionTypeUnload &&
					action.Status == values.RoutePointPackageStatusTODO {
					if err := pkg.MarkUnloading(); err != nil {
						return fmt.Errorf("mark unloading: %w", err)
					}
					if err := s.packages.SavePackage(txCtx, pkg); err != nil {
						return fmt.Errorf("save package: %w", err)
					}
				}
			}
		}
		return nil
	}); txErr != nil {
		return nil, txErr
	}

	return filterByRoutePoint(pkgs, routePointID), nil
}

func (s *PackageSyncService) syncMidPoint(
	ctx context.Context,
	shipmentID values.ShipmentID,
	routePointID values.RoutePointID,
	route *entities.Route,
	target *entities.RoutePoint,
	allPkgs []*entities.Package,
) ([]*entities.Package, error) {
	srcRP, ok := route.FindPointByID(routePointID)
	if !ok {
		return nil, fmt.Errorf("route point %s not found", routePointID)
	}

	pkgsAtPoint := filterByRoutePoint(allPkgs, routePointID)

	existingIDs := make(map[values.PackageID]struct{}, len(pkgsAtPoint))
	for _, pkg := range pkgsAtPoint {
		existingIDs[pkg.ID] = struct{}{}
	}

	if txErr := s.tx.WithTx(ctx, func(txCtx context.Context) error {
		returnPkgs, err := s.packageProvider.ListReturnPackages(txCtx, srcRP.PointID, []values.PointID{target.PointID})
		if err != nil {
			return fmt.Errorf("list return packages: %w", err)
		}

		returnByID := make(map[values.PackageID]*ports.ReturnPackage, len(returnPkgs))
		for i := range returnPkgs {
			returnByID[returnPkgs[i].ID] = &returnPkgs[i]
		}

		var newReturns []ports.ReturnPackage
		for _, rp := range returnPkgs {
			if _, exists := existingIDs[rp.ID]; !exists {
				newReturns = append(newReturns, rp)
			}
		}

		removedIDs := make(map[values.PackageID]struct{})
		for _, pkg := range pkgsAtPoint {
			if pkg.IsReturn {
				if _, exists := returnByID[pkg.ID]; !exists && pkg.IsReadyToLoad() {
					removedIDs[pkg.ID] = struct{}{}
				}
			}
		}

		for _, rp := range newReturns {
			pkg := &entities.Package{
				ID:         rp.ID,
				ShipmentID: shipmentID,
				Status:     values.PackageStatusReadyToLoad,
				IsReturn:   true,
				TareID:     rp.TareID,
				TareType:   rp.TareType,
				Actions: []*entities.PackageAction{
					{RoutePointID: routePointID, Type: values.RoutePointActionTypeLoad, Status: values.RoutePointPackageStatusTODO},
					{RoutePointID: target.ID, Type: values.RoutePointActionTypeUnload, Status: values.RoutePointPackageStatusTODO},
				},
			}

			if saveErr := s.packages.SavePackage(txCtx, pkg); saveErr != nil {
				return fmt.Errorf("save new return package: %w", saveErr)
			}
		}

		for id := range removedIDs {
			if delErr := s.packages.DeletePackage(txCtx, shipmentID, id); delErr != nil {
				return fmt.Errorf("delete removed return package: %w", delErr)
			}
		}

		for _, pkg := range pkgsAtPoint {
			if !pkg.IsReturn {
				for _, action := range pkg.Actions {
					if action.Type == values.RoutePointActionTypeUnload &&
						action.Status == values.RoutePointPackageStatusTODO {
						if err := pkg.MarkUnloading(); err != nil {
							return fmt.Errorf("mark unloading: %w", err)
						}
						if updateErr := s.packages.SavePackage(txCtx, pkg); updateErr != nil {
							return fmt.Errorf("save package: %w", updateErr)
						}
					}
				}
			}
		}

		return nil
	}); txErr != nil {
		return nil, txErr
	}

	updatedPkgs, err := s.packages.GetPackagesByShipmentID(ctx, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("get updated packages: %w", err)
	}

	return filterByRoutePoint(updatedPkgs, routePointID), nil
}

func (s *PackageSyncService) syncStartPoint(
	ctx context.Context,
	shipmentID values.ShipmentID,
	source *entities.RoutePoint,
	unloadPoints []*entities.RoutePoint,
	allPkgs []*entities.Package,
) ([]*entities.Package, error) {
	pkgsAtPoint := filterByRoutePoint(allPkgs, source.ID)

	// TODO(LASTMILE-030): the real repository builds this map via
	// lo.SliceToMap (github.com/samber/lo). samber/lo is not available in the
	// skeleton go.mod, so the map is built with an inline loop instead.
	unloadRoutePointByPointID := make(map[values.PointID]values.RoutePointID, len(unloadPoints))
	for _, rp := range unloadPoints {
		unloadRoutePointByPointID[rp.PointID] = rp.ID
	}

	existingIDs := make(map[values.PackageID]struct{}, len(pkgsAtPoint))
	for _, pkg := range pkgsAtPoint {
		existingIDs[pkg.ID] = struct{}{}
	}

	if txErr := s.tx.WithTx(ctx, func(txCtx context.Context) error {
		// TODO(LASTMILE-030): the real repository maps unload points to point
		// ids via lo.Map (github.com/samber/lo). samber/lo is not available in
		// the skeleton go.mod, so the list is built with an inline loop instead.
		unloadPointIDs := make([]values.PointID, 0, len(unloadPoints))
		for _, rp := range unloadPoints {
			unloadPointIDs = append(unloadPointIDs, rp.PointID)
		}

		forwardPkgs, err := s.packageProvider.ListForwardPackages(
			txCtx,
			source.PointID,
			unloadPointIDs,
		)
		if err != nil {
			return fmt.Errorf("list forward packages: %w", err)
		}

		forwardByID := make(map[values.PackageID]*ports.ForwardPackage, len(forwardPkgs))
		for i := range forwardPkgs {
			forwardByID[forwardPkgs[i].ID] = &forwardPkgs[i]
		}

		var newForwards []ports.ForwardPackage
		for _, rp := range forwardPkgs {
			if _, exists := existingIDs[rp.ID]; !exists {
				newForwards = append(newForwards, rp)
			}
		}

		removedIDs := make(map[values.PackageID]struct{})
		for _, pkg := range pkgsAtPoint {
			if _, exists := forwardByID[pkg.ID]; !exists && pkg.IsReadyToLoad() {
				removedIDs[pkg.ID] = struct{}{}
			}
		}

		for _, rp := range newForwards {
			unloadRoutePointID, ok := unloadRoutePointByPointID[rp.DstPointID]
			if !ok {
				continue
			}

			pkg := &entities.Package{
				ID:         rp.ID,
				ShipmentID: shipmentID,
				Status:     values.PackageStatusReadyToLoad,
				IsReturn:   false,
				TareID:     rp.TareID,
				Actions: []*entities.PackageAction{
					{RoutePointID: source.ID, Type: values.RoutePointActionTypeLoad, Status: values.RoutePointPackageStatusTODO},
					{RoutePointID: unloadRoutePointID, Type: values.RoutePointActionTypeUnload, Status: values.RoutePointPackageStatusTODO},
				},
			}

			if saveErr := s.packages.SavePackage(txCtx, pkg); saveErr != nil {
				return fmt.Errorf("save new return package: %w", saveErr)
			}
		}

		for id := range removedIDs {
			if delErr := s.packages.DeletePackage(txCtx, shipmentID, id); delErr != nil {
				return fmt.Errorf("delete removed return package: %w", delErr)
			}
		}

		return nil
	}); txErr != nil {
		return nil, txErr
	}

	updatedPkgs, err := s.packages.GetPackagesByShipmentID(ctx, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("get updated packages: %w", err)
	}

	return filterByRoutePoint(updatedPkgs, source.ID), nil
}

func filterByRoutePoint(pkgs []*entities.Package, routePointID values.RoutePointID) []*entities.Package {
	var result []*entities.Package
	for _, pkg := range pkgs {
		for _, action := range pkg.Actions {
			if action.RoutePointID == routePointID {
				result = append(result, pkg)
				break
			}
		}
	}
	return result
}

func hasCompletedLoadAtRoutePoint(pkg *entities.Package, routePointID values.RoutePointID) bool {
	for _, action := range pkg.Actions {
		if action.RoutePointID == routePointID &&
			action.Type == values.RoutePointActionTypeLoad &&
			action.Status == values.RoutePointPackageStatusDone {
			return true
		}
	}

	return false
}
