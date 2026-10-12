package policies

import (
	"context"

	"last-mile-task/internal/shipping/domain/entities"
	lmterrors "last-mile-task/internal/shipping/domain/errors"
	"last-mile-task/internal/shipping/domain/values"
)

type ShipmentPolicyRepository interface {
	GetShipmentByID(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error)
	GetRouteByShipmentID(ctx context.Context, id values.ShipmentID) (*entities.Route, error)

	GetShipmentCountByDriverID(
		ctx context.Context,
		driverID values.DriverID,
		statuses ...values.ShipmentStatus,
	) (int, error)

	GetPackageCountByShipmentID(
		ctx context.Context,
		shipmentID values.ShipmentID,
		statuses ...values.PackageStatus,
	) (int, error)

	GetPackagesByShipmentID(
		ctx context.Context,
		shipmentID values.ShipmentID,
	) ([]*entities.Package, error)

	GetReserveCountByShipmentID(
		ctx context.Context,
		shipmentID values.ShipmentID,
		statuses ...values.ReserveStatus,
	) (int, error)

	HasSourcePackages(ctx context.Context, route *entities.Route) (bool, error)
}

type ShipmentPolicy struct {
	repo ShipmentPolicyRepository
}

func NewShipmentPolicy(
	repo ShipmentPolicyRepository,
) *ShipmentPolicy {
	return &ShipmentPolicy{
		repo: repo,
	}
}

func (p *ShipmentPolicy) CanReserveGate(
	ctx context.Context,
	id values.ShipmentID,
	routePointID values.RoutePointID,
) error {
	count, err := p.repo.GetReserveCountByShipmentID(
		ctx,
		id,
		values.ReserveStatusQueued,
		values.ReserveStatusWaitingApproval,
		values.ReserveStatusActive,
	)
	if err != nil {
		return err
	}

	if count > 0 {
		return lmterrors.NewInvalidReserveAlreadyExistsError("shipment already has queued or active reserve")
	}

	route, err := p.repo.GetRouteByShipmentID(ctx, id)
	if err != nil {
		return err
	}

	if _, ok := route.FindPointByID(routePointID); !ok {
		return lmterrors.NewInvalidRoutePointNotFoundError(routePointID.String())
	}

	source, err := route.Source()
	if err != nil {
		return err
	}

	if routePointID != source.ID {
		return lmterrors.NewInvalidReserveMustBeSourceError("route point must be source to reserve gate")
	}

	shipment, err := p.repo.GetShipmentByID(ctx, id)
	if err != nil {
		return err
	}

	if !shipment.IsAssigned() {
		return lmterrors.NewInvalidReserveShipmentNotAssignedError("shipment must be in assigned status to reserve gate at source")
	}

	return nil
}

// CanReserveUnloadingGate verifies a gate reserve can be created for the last
// (target) route point: the shipment must be in progress, the point must be
// the target and no active reserve may exist.
func (p *ShipmentPolicy) CanReserveUnloadingGate(
	ctx context.Context,
	id values.ShipmentID,
	routePointID values.RoutePointID,
) error {
	count, err := p.repo.GetReserveCountByShipmentID(
		ctx,
		id,
		values.ReserveStatusQueued,
		values.ReserveStatusWaitingApproval,
		values.ReserveStatusActive,
	)
	if err != nil {
		return err
	}

	if count > 0 {
		return lmterrors.NewInvalidReserveAlreadyExistsError("shipment already has queued or active reserve")
	}

	route, err := p.repo.GetRouteByShipmentID(ctx, id)
	if err != nil {
		return err
	}

	if _, ok := route.FindPointByID(routePointID); !ok {
		return lmterrors.NewInvalidRoutePointNotFoundError(routePointID.String())
	}

	target, err := route.Target()
	if err != nil {
		return err
	}

	if routePointID != target.ID {
		return lmterrors.NewInvalidReserveMustBeSourceError("route point must be the target to reserve the unloading gate")
	}

	shipment, err := p.repo.GetShipmentByID(ctx, id)
	if err != nil {
		return err
	}

	if !shipment.IsInProgress() {
		return lmterrors.NewInvalidReserveShipmentNotAssignedError("shipment must be in progress to reserve the unloading gate")
	}

	return nil
}

func hasLoadDone(pkg *entities.Package) bool {
	for _, action := range pkg.Actions {
		if action.Type == values.RoutePointActionTypeLoad &&
			action.Status == values.RoutePointPackageStatusDone {
			return true
		}
	}

	return false
}

func hasPendingUnloadAtOtherRoutePoints(pkgs []*entities.Package, routePointID values.RoutePointID) bool {
	for _, pkg := range pkgs {
		if !hasLoadDone(pkg) {
			continue
		}

		for _, action := range pkg.Actions {
			if action.RoutePointID != routePointID &&
				action.Type == values.RoutePointActionTypeUnload &&
				action.Status == values.RoutePointPackageStatusTODO {
				return true
			}
		}
	}

	return false
}

func hasLoadedPackageAtRoutePoint(pkgs []*entities.Package, routePointID values.RoutePointID) bool {
	for _, pkg := range pkgs {
		if !hasLoadDone(pkg) {
			continue
		}

		for _, action := range pkg.Actions {
			if action.RoutePointID == routePointID {
				return true
			}
		}
	}

	return false
}

func hasPendingUnloadActions(pkgs []*entities.Package) bool {
	for _, pkg := range pkgs {
		if !hasLoadDone(pkg) {
			continue
		}

		for _, action := range pkg.Actions {
			if action.Type == values.RoutePointActionTypeUnload &&
				action.Status == values.RoutePointPackageStatusTODO {
				return true
			}
		}
	}

	return false
}

func (p *ShipmentPolicy) CanProceedToUnloadingOnArrivalByID(
	ctx context.Context,
	id values.ShipmentID,
) error {
	route, err := p.repo.GetRouteByShipmentID(ctx, id)
	if err != nil {
		return err
	}

	target, err := route.Target()
	if err != nil {
		return err
	}

	pkgs, err := p.repo.GetPackagesByShipmentID(ctx, id)
	if err != nil {
		return err
	}

	if hasPendingUnloadAtOtherRoutePoints(pkgs, target.ID) {
		return lmterrors.NewInvalidShipmentPendingUnloadAtAnotherPointError("shipment has pending unload at another route point")
	}

	if !hasLoadedPackageAtRoutePoint(pkgs, target.ID) {
		return lmterrors.NewInvalidShipmentNoLoadedPackagesAtTargetError("shipment has no loaded packages at target route point")
	}

	return nil
}

func (p *ShipmentPolicy) CanProceedToLoadingByID(
	ctx context.Context,
	id values.ShipmentID,
) error {
	shipment, err := p.repo.GetShipmentByID(ctx, id)
	if err != nil {
		return err
	}

	if !shipment.IsReserveLoading() && !shipment.IsReserveUnloading() {
		return lmterrors.NewInvalidShipmentMustBeReserveStatusError("shipment must be in reserve loading or reserve unloading status")
	}

	count, err := p.repo.GetReserveCountByShipmentID(
		ctx,
		id,
		values.ReserveStatusActive,
	)
	if err != nil {
		return err
	}

	if count == 0 {
		return lmterrors.NewInvalidShipmentNoActiveReserveError("no active shipment reserve")
	}

	return nil
}

func (p *ShipmentPolicy) CanCompleteShipmentByID(
	ctx context.Context,
	id values.ShipmentID,
) error {
	shipment, err := p.repo.GetShipmentByID(ctx, id)
	if err != nil {
		return err
	}

	if !shipment.IsUnloading() && !shipment.IsInProgress() {
		return lmterrors.NewInvalidShipmentCannotCompleteError("shipment must be in unloading or in progress status to complete")
	}

	pkgs, err := p.repo.GetPackagesByShipmentID(ctx, shipment.ID)
	if err != nil {
		return err
	}

	if hasPendingUnloadActions(pkgs) {
		return lmterrors.NewInvalidPackagePendingUnloadError("shipment has packages with pending unload actions")
	}

	return nil
}

func (p *ShipmentPolicy) CanStartShipmentByID(
	ctx context.Context,
	id values.ShipmentID,
) error {
	shipment, err := p.repo.GetShipmentByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if shipment has completed loading
	if shipment.Status != values.ShipmentStatusLoading {
		return lmterrors.NewInvalidShipmentCannotStartError("shipment must have completed loading to start")
	}

	// Check if shipment has route
	route, err := p.repo.GetRouteByShipmentID(ctx, id)
	if err != nil {
		return err
	}

	if len(route.Points) < entities.MinimumRouteLength {
		return lmterrors.NewInvalidShipmentInvalidRouteError("shipment has invalid route")
	}

	count, err := p.repo.GetPackageCountByShipmentID(
		ctx,
		shipment.ID,
		values.PackageStatusLoaded,
		values.PackageStatusInTransit,
	)
	if err != nil {
		return err
	}

	if count == 0 {
		return lmterrors.NewInvalidShipmentNoLoadedPackagesError("shipment has no loaded packages")
	}

	return nil
}

func (p *ShipmentPolicy) CanCreateShipment(
	ctx context.Context,
	route *entities.Route,
) error {
	has, err := p.repo.HasSourcePackages(ctx, route)
	if err != nil {
		return err
	}

	if !has {
		return lmterrors.NewInvalidRouteNoPackagesAtSourceError("route has no packages at source")
	}

	return nil
}

func (p *ShipmentPolicy) CanCreateShipmentByDriverID(
	ctx context.Context,
	driverID values.DriverID,
) error {
	count, err := p.repo.GetShipmentCountByDriverID(
		ctx,
		driverID,
		values.ActiveShipmentStatuses()...,
	)
	if err != nil {
		return err
	}

	if count > 0 {
		return lmterrors.NewInvalidDriverHasActiveShipmentsError("driver already has an active shipment").WithAttributes(
			lmterrors.Attributes{"count": count},
		)
	}

	return nil
}
