package handlers

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/events"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/cargo"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/dto"
	"last-mile-task/internal/shipping/infrastructure/events/handlers/warehouse"
)

// RoutePointStore resolves route points and their point UUIDs, the data
// required to publish package events to cargo/wms.
//
// In the real repository this is split between *database.Repository
// (lmt.route_point query) and the office client (GetPointByID -> point UUID);
// the skeleton has neither yet, so the handler is written against a narrow
// interface until the adapters are ported. TODO(LASTMILE-052).
type RoutePointStore interface {
	RoutePointByID(ctx context.Context, routePointID values.RoutePointID) (dto.RoutePoint, error)
	PointUUID(ctx context.Context, pointID values.PointID) (uuid.UUID, error)
}

type PackageEventHandler struct {
	routePoints        RoutePointStore
	warehousePublisher *warehouse.Publisher
	cargoPublisher     *cargo.Publisher
}

func NewPackageEventHandler(
	routePoints RoutePointStore,
	warehousePublisher *warehouse.Publisher,
	cargoPublisher *cargo.Publisher,
) *PackageEventHandler {
	return &PackageEventHandler{
		routePoints:        routePoints,
		warehousePublisher: warehousePublisher,
		cargoPublisher:     cargoPublisher,
	}
}

func (o *PackageEventHandler) HandlePackageScannedEvent(
	ctx context.Context,
	event events.PackageScannedEvent,
) error {
	scannedByShipmentID := make(map[values.ShipmentID]map[values.RoutePointID][]dto.Package)
	for _, pkg := range event.Packages {
		scannedByRoutePointID, ok := scannedByShipmentID[pkg.ShipmentID]
		if !ok {
			scannedByRoutePointID = make(map[values.RoutePointID][]dto.Package)
		}

		scannedByRoutePointID[pkg.RoutePointID] = append(
			scannedByRoutePointID[pkg.RoutePointID],
			dto.Package{
				ID:          pkg.PackageID,
				TareID:      pkg.TareID,
				TareType:    pkg.TareType,
				TareSticker: pkg.TareSticker,
				ActionType:  pkg.ActionType,
				ScannedAt:   pkg.Timestamp,
			},
		)

		scannedByShipmentID[pkg.ShipmentID] = scannedByRoutePointID
	}

	for shipmentID, scannedByRoutePointID := range scannedByShipmentID {
		for routePointID, scanned := range scannedByRoutePointID {
			if err := o.publishScannedPackages(
				ctx,
				shipmentID,
				routePointID,
				scanned,
			); err != nil {
				return fmt.Errorf("publish scanned packages: %w", err)
			}
		}
	}

	return nil
}

func (o *PackageEventHandler) HandlePackageInTransitEvent(
	ctx context.Context,
	event events.PackageInTransitEvent,
) error {
	if len(event.PackageIDs) == 0 {
		return nil
	}

	routePoint, err := o.routePointByID(ctx, event.RoutePointID)
	if err != nil {
		return fmt.Errorf("get route point by id: %w", err)
	}

	routePoint, err = o.routePointWithPointUUID(ctx, routePoint)
	if err != nil {
		return fmt.Errorf("get route point uuid: %w", err)
	}

	if err = o.cargoPublisher.PublishInTransitStatus(
		ctx,
		routePoint,
		event.PackageIDs,
		event.TransitionedAt,
	); err != nil {
		return fmt.Errorf("publish in transit status: %w", err)
	}

	return nil
}

func (o *PackageEventHandler) publishScannedPackages(
	ctx context.Context,
	shipmentID uuid.UUID,
	routePointID uuid.UUID,
	pkgs []dto.Package,
) error {
	if len(pkgs) == 0 {
		return nil
	}

	routePoint, err := o.routePointByID(ctx, routePointID)
	if err != nil {
		return fmt.Errorf("get route point by id: %w", err)
	}

	routePoint, err = o.routePointWithPointUUID(ctx, routePoint)
	if err != nil {
		return fmt.Errorf("get route point uuid: %w", err)
	}

	if err = o.warehousePublisher.Publish(ctx, shipmentID, routePoint, pkgs); err != nil {
		return fmt.Errorf("publish wms: %w", err)
	}

	if err = o.cargoPublisher.Publish(ctx, routePoint, pkgs); err != nil {
		return fmt.Errorf("publish cargo: %w", err)
	}

	return nil
}

func (o *PackageEventHandler) routePointByID(ctx context.Context, routePointID uuid.UUID) (dto.RoutePoint, error) {
	return o.routePoints.RoutePointByID(ctx, routePointID)
}

func (o *PackageEventHandler) routePointWithPointUUID(
	ctx context.Context,
	routePoint dto.RoutePoint,
) (dto.RoutePoint, error) {
	pointUUID, err := o.routePoints.PointUUID(ctx, routePoint.PointID)
	if err != nil {
		return dto.RoutePoint{}, fmt.Errorf("get point uuid: %w", err)
	}

	routePoint.PointUUID = pointUUID

	return routePoint, nil
}
