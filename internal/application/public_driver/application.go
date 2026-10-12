package publicdriver

import (
	"context"

	shipping_application "last-mile-task/internal/shipping/application"
	"last-mile-task/internal/shipping/application/demand"
	"last-mile-task/internal/shipping/application/ports"
	"last-mile-task/internal/shipping/application/scanner"
	"last-mile-task/internal/shipping/domain/values"
	"last-mile-task/internal/shipping/queries/driver/basket"
	shipment "last-mile-task/internal/shipping/queries/driver/shipment"
	"last-mile-task/internal/shipping/queries/driver/showcase"
	"last-mile-task/pkg/qr"
)

// DriversProvider resolves the current driver from a request context.
//
// TODO(LASTMILE-081): the real provider resolves a full values.Driver with
// roles/supplier through users/client-proxy plus gateway metadata. The
// skeleton resolves only the driver ID from the interceptor context and
// leaves the supplier unresolved (0) until the users client port is wired.
type DriversProvider interface {
	GetDriverFromContext(ctx context.Context) (values.Driver, error)
}

// Transactional wraps a unit of work shared by the creation use cases.
type Transactional interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// QRHandshake issues and verifies the reserve QR handshake sessions.
type QRHandshake interface {
	CreateSession(session qr.Session) (string, error)
	DecodeQR(token string) (qr.Session, error)
}

// Config carries the dependencies the public driver application relies on.
//
// TODO(LASTMILE-082): the real Config additionally wires office/users/
// payments/documents clients, outbox readers and event handlers. The skeleton
// slice does not need them — external clients are not ported yet.
type Config struct {
	ScannerService     *scanner.PackageScanner
	PackageSyncService *shipping_application.PackageSyncService
	ShippingQueries    *shipment.Queries
	DriversProvider    DriversProvider

	AssemblerService   *shipping_application.AssemblerService
	ProgressService    *shipping_application.ShipmentProgressService
	RoutePointService  *shipping_application.RoutePointService
	DemandService      *demand.DemandService
	ReserveService     *shipping_application.ReserveService
	RouteTemplatesProvider ports.RouteTemplateProvider
	Handshake             QRHandshake
	Tx                    Transactional

	ShowcaseStore showcase.ShowcaseStore
	BasketStore   basket.BasketItemsStore
}

// Application wires the use cases of the driver mobile client slice.
type Application struct {
	scannerService     *scanner.PackageScanner
	packageSyncService *shipping_application.PackageSyncService
	shippingQueries    *shipment.Queries
	driversProvider    DriversProvider

	assemblerService       *shipping_application.AssemblerService
	progressService        *shipping_application.ShipmentProgressService
	routePointService      *shipping_application.RoutePointService
	demandService          *demand.DemandService
	reserveService         *shipping_application.ReserveService
	routeTemplatesProvider ports.RouteTemplateProvider
	handshake              QRHandshake
	tx                     Transactional

	showcaseStore showcase.ShowcaseStore
	basketStore   basket.BasketItemsStore
}

// New builds the public driver Application from already-wired infrastructure
// services.
func New(cfg Config) *Application {
	return &Application{
		scannerService:     cfg.ScannerService,
		packageSyncService: cfg.PackageSyncService,
		shippingQueries:    cfg.ShippingQueries,
		driversProvider:    cfg.DriversProvider,

		assemblerService:       cfg.AssemblerService,
		progressService:        cfg.ProgressService,
		routePointService:      cfg.RoutePointService,
		demandService:          cfg.DemandService,
		reserveService:         cfg.ReserveService,
		routeTemplatesProvider: cfg.RouteTemplatesProvider,
		handshake:              cfg.Handshake,
		tx:                     cfg.Tx,

		showcaseStore: cfg.ShowcaseStore,
		basketStore:   cfg.BasketStore,
	}
}

// NewForTesting creates a minimal Application with injectable dependencies,
// intended for unit tests only. Only the subset of methods under test needs to
// be wired.
//
// TODO(LASTMILE-083): the real NewForTesting takes only the provider and
// queries — the skeleton adds the scanner so the transport tests can cover the
// partial-failure path of SetPackagesStatus, and the creation services so the
// CreateShipment family can be covered. Revisit once the scanner has a
// portable mock.
func NewForTesting(
	driversProvider DriversProvider,
	shippingQueries *shipment.Queries,
	scannerService *scanner.PackageScanner,
	assemblerService *shipping_application.AssemblerService,
	progressService *shipping_application.ShipmentProgressService,
	routePointService *shipping_application.RoutePointService,
	demandService *demand.DemandService,
	routeTemplatesProvider ports.RouteTemplateProvider,
	tx Transactional,
) *Application {
	return &Application{
		driversProvider: driversProvider,
		shippingQueries: shippingQueries,
		scannerService:  scannerService,

		assemblerService:       assemblerService,
		progressService:        progressService,
		routePointService:      routePointService,
		demandService:          demandService,
		routeTemplatesProvider: routeTemplatesProvider,
		tx:                     tx,
	}
}