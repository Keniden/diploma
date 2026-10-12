package shipment

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/queries/driver/shipment/readmodels"
	"last-mile-task/internal/utils"
)

type Queries struct {
	store     ShipmentStore
	addresses AddressResolver
}

func NewQueries(store ShipmentStore, addresses AddressResolver) *Queries {
	return &Queries{
		store:     store,
		addresses: addresses,
	}
}

func (q *Queries) GetShipmentByDriverInStatuses(
	ctx context.Context,
	driverID uuid.UUID,
	statuses ...values.ShipmentStatus,
) ([]Shipment, error) {
	shipments, err := q.store.GetDriverShipmentsByStatuses(ctx, driverID, statuses...)
	if err != nil {
		return nil, fmt.Errorf("get driver shipments by statuses: %w", err)
	}

	return shipments, nil
}

func (q *Queries) GetShipmentByID(ctx context.Context, id uuid.UUID) (Shipment, error) {
	shipment, err := q.store.GetShipmentByID(ctx, id)
	if err != nil {
		return Shipment{}, fmt.Errorf("get shipment by id: %w", err)
	}

	return shipment, nil
}

func (q *Queries) GetDetailedShipmentByID(ctx context.Context, id uuid.UUID) (DetailedShipment, error) {
	shipment, err := q.GetShipmentByID(ctx, id)
	if err != nil {
		return DetailedShipment{}, err
	}

	route, err := q.GetDetailedRouteByShipmentID(ctx, id)
	if err != nil {
		return DetailedShipment{}, err
	}
	reserve, err := q.store.GetActiveShipmentReserveByShipmentID(ctx, id)
	if err != nil {
		return DetailedShipment{}, fmt.Errorf("get active reserve by shipment id: %w", err)
	}

	return DetailedShipment{
		Shipment:      shipment,
		DetailedRoute: route,
		Reserve:       reserve,
	}, nil
}

func (q *Queries) GetDetailedRouteByShipmentID(ctx context.Context, id uuid.UUID) (DetailedRoute, error) {
	route, err := q.store.GetRouteByShipmentID(ctx, id)
	if err != nil {
		return DetailedRoute{}, fmt.Errorf("get route by shipment id: %w", err)
	}

	routePoints, err := q.GetDetailedRoutePointsByShipmentID(ctx, id)
	if err != nil {
		return DetailedRoute{}, err
	}

	return DetailedRoute{
		Route:       route,
		RoutePoints: routePoints,
	}, nil
}

func (q *Queries) GetDetailedRoutePointsByShipmentID(ctx context.Context, id uuid.UUID) ([]DetailedRoutePoint, error) {
	routePoints, err := q.store.GetRoutePointsByShipmentID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get route points by shipment id: %w", err)
	}

	packages, err := q.store.GetPackagesByShipmentID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get packages by shipment id: %w", err)
	}

	packageActions, err := q.store.GetPackageActionsByShipmentID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get package actions by shipment id: %w", err)
	}

	detailedPackages := NewDetailedPackages(packages, packageActions)

	packagesByRoutePointID := make(map[uuid.UUID][]DetailedPackage)

	for _, dp := range detailedPackages {
		packagesByRoutePointID[dp.RoutePointID] = append(packagesByRoutePointID[dp.RoutePointID], dp)
	}

	pointIDs := utils.Uniq(utils.Map(routePoints, func(rp RoutePoint) int64 {
		return rp.PointID
	}))
	addressByPointID, err := q.addresses.AddressesByPointIDs(ctx, pointIDs)
	if err != nil {
		return nil, fmt.Errorf("get route point addresses: %w", err)
	}

	return utils.Map(routePoints, func(rp RoutePoint) DetailedRoutePoint {
		rp.Address = addressByPointID[rp.PointID]
		return DetailedRoutePoint{
			RoutePoint: rp,
			Packages:   packagesByRoutePointID[rp.ID],
		}
	}), nil
}

func (q *Queries) ListPackagesByShipmentID(ctx context.Context, shipmentID uuid.UUID) ([]Package, error) {
	pkgs, err := q.store.GetPackagesByShipmentID(ctx, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("list packages by shipment id: %w", err)
	}

	return pkgs, nil
}

func (q *Queries) ListPackageActionsByShipmentID(ctx context.Context, shipmentID uuid.UUID) ([]PackageAction, error) {
	actions, err := q.store.GetPackageActionsByShipmentID(ctx, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("list package actions by shipment id: %w", err)
	}

	return actions, nil
}

func (q *Queries) GetDstPointsByPackageIDs(ctx context.Context, ids []uuid.UUID, shipmentID uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	return q.store.GetDstPointByPackageIDs(ctx, ids, shipmentID)
}

func (q *Queries) HasLoadedReturnPackagesAtRoutePoint(ctx context.Context, routePointID uuid.UUID) (bool, error) {
	return q.store.HasLoadedReturnPackagesAtRoutePoint(ctx, routePointID)
}

func (q *Queries) GetDetailedShipmentsByDriver(
	ctx context.Context,
	driverID uuid.UUID,
	statuses ...values.ShipmentStatus,
) ([]DetailedShipment, error) {
	shipments, err := q.store.GetDriverShipmentsByStatuses(ctx, driverID, statuses...)
	if err != nil {
		return nil, fmt.Errorf("get driver shipments by statuses: %w", err)
	}

	if len(shipments) == 0 {
		return []DetailedShipment{}, nil
	}

	ids := make([]uuid.UUID, len(shipments))
	for i, s := range shipments {
		ids[i] = s.ID
	}

	return q.GetDetailedShipmentsByIDs(ctx, ids, shipments)
}

func (q *Queries) GetDetailedShipmentsByIDs(ctx context.Context, ids []uuid.UUID, shipments []Shipment) ([]DetailedShipment, error) {
	if len(ids) == 0 {
		return []DetailedShipment{}, nil
	}

	routes, err := q.store.GetRoutesByShipmentIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get routes by shipment ids: %w", err)
	}
	// TODO(LASTMILE-031): the real repository builds this map via
	// lo.SliceToMap (github.com/samber/lo). samber/lo is not available in the
	// skeleton go.mod, so the map is built with an inline loop instead.
	routeByShipmentID := make(map[uuid.UUID]Route, len(routes))
	for _, r := range routes {
		routeByShipmentID[r.ShipmentID] = r
	}

	routePoints, err := q.store.GetRoutePointsByShipmentIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get route points by shipment ids: %w", err)
	}

	packages, err := q.store.GetPackagesByShipmentIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get packages by shipment ids: %w", err)
	}

	pkgActions, err := q.store.GetPackageActionsByShipmentIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get package actions by shipment ids: %w", err)
	}

	reserves, err := q.store.GetActiveShipmentReservesByShipmentIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get active reserves by shipment ids: %w", err)
	}

	return assembleDetailedShipments(shipments, routeByShipmentID, routePoints, packages, pkgActions, reserves), nil
}

func assembleDetailedShipments(
	shipments []readmodels.Shipment,
	routeByShipmentID map[uuid.UUID]readmodels.Route,
	routePoints []readmodels.RoutePoint,
	packages []readmodels.Package,
	pkgActions []readmodels.PackageAction,
	reserves map[uuid.UUID]readmodels.ShipmentReserve,
) []DetailedShipment {
	routePointsByShipmentID := make(map[uuid.UUID][]readmodels.RoutePoint)
	for _, rp := range routePoints {
		routePointsByShipmentID[rp.ShipmentID] = append(routePointsByShipmentID[rp.ShipmentID], rp)
	}

	detailedPackages := NewDetailedPackages(packages, pkgActions)
	pkgByRoutePointID := make(map[uuid.UUID][]DetailedPackage)
	for _, dp := range detailedPackages {
		pkgByRoutePointID[dp.RoutePointID] = append(pkgByRoutePointID[dp.RoutePointID], dp)
	}

	out := make([]DetailedShipment, 0, len(shipments))
	for _, s := range shipments {
		rps := routePointsByShipmentID[s.ID]
		detailedRPs := make([]DetailedRoutePoint, 0, len(rps))
		for _, rp := range rps {
			detailedRPs = append(detailedRPs, DetailedRoutePoint{
				RoutePoint: rp,
				Packages:   pkgByRoutePointID[rp.ID],
			})
		}

		detailedShipment := DetailedShipment{
			Shipment: s,
			DetailedRoute: DetailedRoute{
				Route:       routeByShipmentID[s.ID],
				RoutePoints: detailedRPs,
			},
		}
		if reserve, ok := reserves[s.ID]; ok {
			reserveCopy := reserve
			detailedShipment.Reserve = &reserveCopy
		}

		out = append(out, detailedShipment)
	}

	return out
}
