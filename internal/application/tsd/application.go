package tsd

import (
	"context"

	"last-mile-task/internal/shipping/domain/entities"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/domain/values/scan"
	"last-mile-task/pkg/qr"
)

// QRHandshake verifies a reserve QR handshake token and returns the session it
// carries. The production implementation is built on pkg/qr.DecodeQR with the
// configured handshake secret (see internal/platform/application/handshake.go).
type QRHandshake interface {
	DecodeQR(token string) (qr.Session, error)
}

// Scanner scans and verifies packages at a route point, publishing the package
// scan events to the outbox (cargo + WMS) inside the writer transaction.
type Scanner interface {
	ScanPackages(ctx context.Context, shipmentID values.ShipmentID, scannedPackages []scan.Package) error
}

// ReserveApprover confirms the gate reserve and moves the shipment into
// LOADING / UNLOADING (ARCHITECTURE.md §23 flows 4 and 7).
type ReserveApprover interface {
	ApproveReserveByRoutePointID(ctx context.Context, shipmentID values.ShipmentID, routePointID values.RoutePointID) error
}

// RouteRepository loads a shipment route so the scanned route point can be
// transitioned per routePointStatusModel.
type RouteRepository interface {
	GetRouteByShipmentID(ctx context.Context, shipmentID values.ShipmentID) (*entities.Route, error)
	SaveRoute(ctx context.Context, route *entities.Route) error
}

// PackagesToScanLister lists packages awaiting a scan at a route point.
type PackagesToScanLister interface {
	ListPackagesToScan(ctx context.Context, shipmentID values.ShipmentID, routePointID values.RoutePointID) ([]PackageToScan, error)
}

// PackageToScan is the read model of a package awaiting a scan at a point.
type PackageToScan struct {
	ID         values.PackageID
	ShipmentID values.ShipmentID
	Status     values.PackageStatus
	TareID     values.TareID
	TareType   values.TareType
}

// Config carries the dependencies of the TSD use cases.
type Config struct {
	Handshake       QRHandshake
	Scanner         Scanner
	ReserveApprover ReserveApprover
	RouteRepository RouteRepository
	PackagesToScan  PackagesToScanLister
}

// Application wires the TSD use cases.
type Application struct {
	handshake       QRHandshake
	scanner         Scanner
	reserveApprover ReserveApprover
	routeRepository RouteRepository
	packagesToScan  PackagesToScanLister
}

// New builds a TSD Application from already-wired dependencies.
func New(cfg Config) *Application {
	return &Application{
		handshake:       cfg.Handshake,
		scanner:         cfg.Scanner,
		reserveApprover: cfg.ReserveApprover,
		routeRepository: cfg.RouteRepository,
		packagesToScan:  cfg.PackagesToScan,
	}
}
