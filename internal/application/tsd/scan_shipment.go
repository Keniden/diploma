package tsd

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	lmterrors "last-mile-task/internal/shipping/domain/errors"
	"last-mile-task/internal/shipping/domain/values"
)

// ScanShipment confirms the gate reserve carried by the QR handshake token
// (ARCHITECTURE.md §23 flows 4 and 7). It decodes the token, approves the
// reserve via ReserveApprover (which moves the shipment into LOADING /
// UNLOADING) and transitions the scanned route point per routePointStatusModel.
func (a *Application) ScanShipment(ctx context.Context, token string) error {
	session, err := a.handshake.DecodeQR(token)
	if err != nil {
		return fmt.Errorf("decode qr: %w", err)
	}

	shipmentID, err := parseShipmentID(session.ShipmentID)
	if err != nil {
		return fmt.Errorf("parse shipment id: %w", err)
	}
	routePointID, err := parseRoutePointID(session.RoutePointID)
	if err != nil {
		return fmt.Errorf("parse route point id: %w", err)
	}

	if err := a.reserveApprover.ApproveReserveByRoutePointID(ctx, shipmentID, routePointID); err != nil {
		return fmt.Errorf("approve reserve: %w", err)
	}

	if err := a.markRoutePointDriverArrived(ctx, shipmentID, routePointID); err != nil {
		return err
	}

	return nil
}

// markRoutePointDriverArrived loads the shipment route and transitions the
// scanned route point into DRIVER_ARRIVED per routePointStatusModel.
func (a *Application) markRoutePointDriverArrived(
	ctx context.Context,
	shipmentID values.ShipmentID,
	routePointID values.RoutePointID,
) error {
	route, err := a.routeRepository.GetRouteByShipmentID(ctx, shipmentID)
	if err != nil {
		return fmt.Errorf("get route by shipment id: %w", err)
	}

	point, ok := route.FindPointByID(routePointID)
	if !ok {
		return lmterrors.NewInvalidRoutePointNotFoundError("route point not found")
	}

	if err := point.DriverArrived(); err != nil {
		return fmt.Errorf("mark route point driver arrived: %w", err)
	}

	if err := a.routeRepository.SaveRoute(ctx, route); err != nil {
		return fmt.Errorf("save route: %w", err)
	}

	return nil
}

func parseShipmentID(raw string) (values.ShipmentID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return values.ShipmentID{}, err
	}
	return values.ShipmentID(id), nil
}

func parseRoutePointID(raw string) (values.RoutePointID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return values.RoutePointID{}, err
	}
	return values.RoutePointID(id), nil
}
