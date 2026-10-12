package shipment

import (
	"context"

	"github.com/google/uuid"

	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/queries/driver/shipment/readmodels"
)

type (
	Shipment      = readmodels.Shipment
	Route         = readmodels.Route
	RoutePoint    = readmodels.RoutePoint
	Package       = readmodels.Package
	PackageAction = readmodels.PackageAction
)

type AddressResolver interface {
	AddressesByPointIDs(ctx context.Context, pointIDs []int64) (map[int64]string, error)
}

type ShipmentStore interface {
	GetShipmentByID(ctx context.Context, id uuid.UUID) (Shipment, error)
	GetDriverShipmentsByStatuses(ctx context.Context, driverID uuid.UUID, statuses ...values.ShipmentStatus) ([]Shipment, error)
	GetRouteByShipmentID(ctx context.Context, id uuid.UUID) (Route, error)
	GetRoutePointsByShipmentID(ctx context.Context, id uuid.UUID) ([]RoutePoint, error)
	GetPackageActionsByShipmentID(ctx context.Context, id uuid.UUID) ([]PackageAction, error)
	GetPackagesByShipmentID(ctx context.Context, id uuid.UUID) ([]Package, error)

	GetRoutesByShipmentIDs(ctx context.Context, ids []uuid.UUID) ([]Route, error)
	GetRoutePointsByShipmentIDs(ctx context.Context, ids []uuid.UUID) ([]RoutePoint, error)
	GetPackagesByShipmentIDs(ctx context.Context, ids []uuid.UUID) ([]Package, error)
	GetPackageActionsByShipmentIDs(ctx context.Context, ids []uuid.UUID) ([]PackageAction, error)
	GetDstPointByPackageIDs(ctx context.Context, ids []uuid.UUID, shipmentID uuid.UUID) (map[uuid.UUID]uuid.UUID, error)

	GetActiveShipmentReserveByShipmentID(ctx context.Context, shipmentID uuid.UUID) (*readmodels.ShipmentReserve, error)
	GetActiveShipmentReservesByShipmentIDs(ctx context.Context, shipmentIDs []uuid.UUID) (map[uuid.UUID]readmodels.ShipmentReserve, error)
	HasLoadedReturnPackagesAtRoutePoint(ctx context.Context, routePointID uuid.UUID) (bool, error)
}
