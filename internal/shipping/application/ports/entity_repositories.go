package ports

import (
	"context"

	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
)

type ShipmentRepository interface {
	GetShipmentByID(ctx context.Context, id values.ShipmentID) (*entities.Shipment, error)
	SaveShipment(ctx context.Context, shipment *entities.Shipment) error
}

type RouteRepository interface {
	GetRouteByID(ctx context.Context, id values.RouteID) (*entities.Route, error)
	GetRouteByShipmentID(ctx context.Context, id values.ShipmentID) (*entities.Route, error)
	SaveRoute(ctx context.Context, route *entities.Route) error
}

type PackageRepository interface {
	GetPackageByID(ctx context.Context, shipmentID values.ShipmentID, id values.PackageID) (*entities.Package, error)
	GetPackagesByShipmentID(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error)
	GetLoadedPackagesByShipmentID(ctx context.Context, shipmentID values.ShipmentID) ([]*entities.Package, error)
	GetPackagesByIDs(ctx context.Context, shipmentID values.ShipmentID, ids ...values.PackageID) ([]*entities.Package, error)
	SavePackage(ctx context.Context, pkg *entities.Package) error
	SavePackageScan(ctx context.Context, pkg *entities.Package, action *entities.PackageAction) error
	DeletePackage(ctx context.Context, shipmentID values.ShipmentID, id values.PackageID) error
}

type ReserveRepository interface {
	GetReserveByID(ctx context.Context, id values.ReserveID) (*entities.Reserve, error)
	GetReserveByShipmentID(ctx context.Context, shipmentID values.ShipmentID) (*entities.Reserve, error)
	GetReserveByRoutePointID(ctx context.Context, routePointID values.RoutePointID) (*entities.Reserve, error)
	ListExpiredWaitingApprovalReserves(ctx context.Context, limit int) ([]*entities.Reserve, error)
	ListExpiredActiveReserves(ctx context.Context, limit int) ([]*entities.Reserve, error)
	SaveReserve(ctx context.Context, reserve *entities.Reserve) error
}
