package application

import (
	"context"
	"errors"
	"fmt"

	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/values"
)

var (
	ErrRoutePointsNotReadyForTargetArrival = errors.New("some route points have packages which are not unloaded")
	ErrNoLoadedPackagesAtSource            = errors.New("no loaded packages at source route point")
)

// ShipmentCompleter completes a shipment when the driver departs the last
// point of the route. It is implemented by *ShipmentProgressService.
type ShipmentCompleter interface {
	CompleteShipmentByID(ctx context.Context, id values.ShipmentID) error
}

type RoutePointService struct {
	routes          ports.RouteRepository
	packages        ports.PackageRepository
	routeEvents     ports.RouteEventPublisher
	completeShipment ShipmentCompleter
	tx              ports.Transactional
}

func NewRoutePointService(
	routes ports.RouteRepository,
	packages ports.PackageRepository,
	routeEvents ports.RouteEventPublisher,
	completeShipment ShipmentCompleter,
	tx ports.Transactional,
) *RoutePointService {
	return &RoutePointService{
		routes:          routes,
		packages:        packages,
		routeEvents:     routeEvents,
		completeShipment: completeShipment,
		tx:              tx,
	}
}

const (
	ArrivedAtUnknownPoint = iota
	ArrivedAtSource
	ArrivedAtTarget
	ArrivedAtMiddlePoint
)

func (s *RoutePointService) ArriveToRoutePoint(
	ctx context.Context,
	shipmentID, routePointID values.ID,
) (arrivedAt int, packagesNotLoadedIDs []values.RoutePointID, err error) {
	txErr := s.tx.WithTx(ctx, func(txCtx context.Context) error {
		route, err := s.routes.GetRouteByShipmentID(txCtx, shipmentID)
		if err != nil {
			return fmt.Errorf("get route by shipment id: %w", err)
		}

		routePoint, ok := route.FindPointByID(routePointID)
		if !ok {
			return newInvalidRoutePointNotFoundError(routePointID.String())
		}

		source, err := route.Source()
		if err != nil {
			return fmt.Errorf("get source: %w", err)
		}

		target, err := route.Target()
		if err != nil {
			return fmt.Errorf("get target: %w", err)
		}

		switch routePoint.ID {
		case source.ID:
			arrivedAt = ArrivedAtSource
		case target.ID:
			arrivedAt = ArrivedAtTarget
		default:
			arrivedAt = ArrivedAtMiddlePoint
		}

		if routePoint.ID == target.ID {
			for _, rp := range route.Points {
				if rp.ID == routePoint.ID {
					continue
				}
				if !isRoutePointReadyForTargetArrival(rp.Status) {
					return wrapLmtError(
						newInvalidRoutePointNotReadyForTargetArrivalError("route point is not ready for target arrival").WithAttributes(
							Attributes{
								"routePointID": rp.ID.String(),
								"status":       rp.Status.ToString(),
							},
						),
						ErrRoutePointsNotReadyForTargetArrival,
					)
				}
				if rp.Status == values.RoutePointStatusPackagesNotLoaded ||
					rp.Status == values.RoutePointStatusWaitingArrival {
					packagesNotLoadedIDs = append(packagesNotLoadedIDs, rp.ID)
				}
			}
		}

		for _, rp := range route.Points {
			if rp.ID == routePointID {
				continue
			}
			if rp.Status == values.RoutePointStatusDriverArrived {
				return newInvalidRoutePointDriverAlreadyArrivedError("another route point is in driver arrived status")
			}
		}

		if err := routePoint.DriverArrived(); err != nil {
			return fmt.Errorf("driver arrived: %w", err)
		}

		if err := s.routes.SaveRoute(txCtx, route); err != nil {
			return fmt.Errorf("save route: %w", err)
		}

		return nil
	})
	if txErr != nil {
		return arrivedAt, nil, txErr
	}

	return arrivedAt, packagesNotLoadedIDs, nil
}

func isRoutePointReadyForTargetArrival(status values.RoutePointStatus) bool {
	return status == values.RoutePointStatusCompleted ||
		status == values.RoutePointStatusPackagesNotLoaded ||
		status == values.RoutePointStatusWaitingArrival
}

const (
	DepartedFromUnknownPoint = iota
	DepartedFromSource
	DepartedFromTarget
	DepartedFromMiddlePoint
)

func (s *RoutePointService) DepartFromRoutePoint(
	ctx context.Context,
	shipmentID, routePointID values.ID,
) (int, error) {
	var departedFrom int

	err := s.tx.WithTx(ctx, func(txCtx context.Context) error {
		route, err := s.routes.GetRouteByShipmentID(txCtx, shipmentID)
		if err != nil {
			return fmt.Errorf("get route by shipment id: %w", err)
		}

		routePoint, ok := route.FindPointByID(routePointID)
		if !ok {
			return newInvalidRoutePointNotFoundError(routePointID.String())
		}

		pkgs, err := s.packages.GetPackagesByShipmentID(txCtx, shipmentID)
		if err != nil {
			return fmt.Errorf("get packages by shipment id: %w", err)
		}

		source, err := route.Source()
		if err != nil {
			return fmt.Errorf("get source: %w", err)
		}

		target, err := route.Target()
		if err != nil {
			return fmt.Errorf("get target: %w", err)
		}

		switch routePoint.ID {
		case source.ID:
			if !hasLoadedPackage(pkgs) {
				return wrapLmtError(
					newInvalidRoutePointNoLoadedPackagesAtSourceError("no loaded packages at source route point").WithAttributes(
						Attributes{
							"shipmentID":   shipmentID.String(),
							"routePointID": routePointID.String(),
						},
					),
					ErrNoLoadedPackagesAtSource,
				)
			}
			if err := routePoint.Completed(); err != nil {
				return fmt.Errorf("set route point status: %w", err)
			}
			departedFrom = DepartedFromSource
		default:
			var setStatusErr error
			switch {
			case hasPendingUnloadAtRoutePoint(pkgs, routePointID):
				setStatusErr = routePoint.PackagesNotUnloaded()
			case hasPendingLoadAtRoutePoint(pkgs, routePointID):
				setStatusErr = routePoint.PackagesNotLoaded()
			default:
				setStatusErr = routePoint.Completed()
			}
			if setStatusErr != nil {
				return fmt.Errorf("set route point status: %w", setStatusErr)
			}
			if routePoint.ID == target.ID {
				departedFrom = DepartedFromTarget
			} else {
				departedFrom = DepartedFromMiddlePoint
			}
		}

		if err := s.routes.SaveRoute(txCtx, route); err != nil {
			return fmt.Errorf("save route: %w", err)
		}

		if err := s.routeEvents.PublishRouteEvent(txCtx, events.PointCompletedEvent{
			ShipmentID:   shipmentID,
			RoutePointID: routePointID,
		}); err != nil {
			return fmt.Errorf("publish route point completed: %w", err)
		}

		if departedFrom == DepartedFromTarget {
			if routePoint.Status != values.RoutePointStatusCompleted {
				// The last point is marked PACKAGES_NOT_UNLOADED/
				// PACKAGES_NOT_LOADED above; the shipment must not complete.
				// The codebase maps the unfinished-goods case to InvalidArgument
				// (mirrors the real repository's NewInvalidPackagePendingUnloadError),
				// while the delta-spec scenario generically calls it a
				// FailedPrecondition - kept as the codebase mapping here.
				return newInvalidPackagePendingUnloadError("shipment has packages that have not been unloaded yet").WithAttributes(
					Attributes{
						"shipmentID":   shipmentID.String(),
						"routePointID": routePointID.String(),
						"status":       routePoint.Status.ToString(),
					},
				)
			}
			if err := s.completeShipment.CompleteShipmentByID(txCtx, shipmentID); err != nil {
				return fmt.Errorf("complete shipment after last point departure: %w", err)
			}
		}

		return nil
	})
	return departedFrom, err
}

func hasPendingUnloadAtRoutePoint(pkgs []*entities.Package, routePointID values.RoutePointID) bool {
	for _, pkg := range pkgs {
		for _, action := range pkg.Actions {
			if action.RoutePointID == routePointID &&
				action.Type == values.RoutePointActionTypeUnload &&
				action.Status == values.RoutePointPackageStatusTODO {
				return true
			}
		}
	}
	return false
}

func hasPendingLoadAtRoutePoint(pkgs []*entities.Package, routePointID values.RoutePointID) bool {
	for _, pkg := range pkgs {
		for _, action := range pkg.Actions {
			if action.RoutePointID == routePointID &&
				action.Type == values.RoutePointActionTypeLoad &&
				action.Status == values.RoutePointPackageStatusTODO {
				return true
			}
		}
	}
	return false
}

func hasLoadedPackage(pkgs []*entities.Package) bool {
	for _, pkg := range pkgs {
		if pkg.IsLoaded() {
			return true
		}
	}
	return false
}

func (s *RoutePointService) ForcedCompleteRoutePoint(
	ctx context.Context,
	shipmentID, routePointID values.ID,
) error {
	return s.tx.WithTx(ctx, func(txCtx context.Context) error {
		route, err := s.routes.GetRouteByShipmentID(txCtx, shipmentID)
		if err != nil {
			return fmt.Errorf("get route by shipment id: %w", err)
		}

		routePoint, ok := route.FindPointByID(routePointID)
		if !ok {
			return newInvalidRoutePointNotFoundError(routePointID.String())
		}

		pkgs, err := s.packages.GetPackagesByShipmentID(txCtx, shipmentID)
		if err != nil {
			return fmt.Errorf("get packages by shipment id: %w", err)
		}

		if hasPendingUnloadAtRoutePoint(pkgs, routePointID) {
			return newInvalidPackagePendingUnloadError("route point has packages with pending unload")
		}

		for _, pkg := range filterByRoutePoint(pkgs, routePointID) {
			canceled, err := pkg.CancelLoad()
			if err != nil {
				return fmt.Errorf("cancel load: %w", err)
			}
			if !canceled {
				continue
			}
			if err := s.packages.SavePackage(txCtx, pkg); err != nil {
				return fmt.Errorf("save package: %w", err)
			}
		}

		if err := routePoint.Completed(); err != nil {
			return fmt.Errorf("set route point status: %w", err)
		}

		if err := s.routes.SaveRoute(txCtx, route); err != nil {
			return fmt.Errorf("save route: %w", err)
		}

		return nil
	})
}
